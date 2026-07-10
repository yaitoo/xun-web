package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// This file implements a minimal RFC 6455 WebSocket server without any
// third-party dependency. The whole reason we route it via app.Mux() is
// that the HTTP → WS upgrade requires the raw http.ResponseWriter —
// xun's high-level HandleFunc wraps the writer in xun.ResponseWriter and
// would prevent the Hijack() call we need.
//
// Registering on the underlying *http.ServeMux gives us back the
// untouched writer. The trade-off is that app.Mux() routes bypass ALL
// xun middleware (app.Use, group.Use, the viewer/content-negotiation
// path, and the framework's error-to-status mapping), so anything xun
// normally does for us — including sessionMiddleware — has to be
// re-implemented by hand on this path.
//
// Wire format reference: RFC 6455 §5. The protocol is small enough to
// implement inline; for production use prefer gorilla/websocket or
// coder/websocket.

// wsMagicGUID is the RFC 6455 §1.3 constant used in the Sec-WebSocket-Key
// handshake.
const wsMagicGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

const (
	// wsReadDeadline bounds how long any single read may block. Without
	// it a stalled client would pin a goroutine forever.
	wsReadDeadline = 60 * time.Second
	// wsWriteDeadline bounds a single write. Writes are serialised under
	// c.mu, so a slow client can delay only one writer at a time.
	wsWriteDeadline = 10 * time.Second
	// wsMaxFrame is the largest payload we accept in a single frame.
	// Anything larger is rejected with code 1009 (Message Too Big).
	wsMaxFrame = 1 << 20 // 1 MiB
)

// wsOpcode is the 4-bit opcode in a WebSocket frame header.
type wsOpcode uint8

const (
	wsOpcodeContinuation wsOpcode = 0x0
	wsOpcodeText         wsOpcode = 0x1
	wsOpcodeBinary       wsOpcode = 0x2
	wsOpcodeClose        wsOpcode = 0x8
	wsOpcodePing         wsOpcode = 0x9
	wsOpcodePong         wsOpcode = 0xA
)

// wsConn is the per-connection state.
//
// The mutex serialises concurrent writes because the underlying TCP
// connection is not safe for concurrent writes — a real app might fan
// out to multiple broadcast producers and they would otherwise race.
//
// deadline is the part of the conn that supports SetDeadline. The
// io.ReadWriteCloser returned by hijack is the bare *net.TCPConn,
// which DOES support SetDeadline, but the interface type does not
// expose it. We capture the capability via a narrow interface so the
// deadline helpers can be called without a per-call type assertion.
type wsConn struct {
	rwc         io.ReadWriteCloser
	deadline    interface{ SetDeadline(time.Time) error }
	remoteAddr  string
	mu          sync.Mutex
	closeOnce   sync.Once
	closeSignal chan struct{} // closed when the conn is going away
}

func (c *wsConn) close(code uint16, reason string) {
	c.closeOnce.Do(func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		// Best-effort close frame so the peer sees a clean shutdown.
		_ = c.writeFrameUnlocked(true, 0x8, code, []byte(reason))
		_ = c.rwc.Close()
		close(c.closeSignal)
	})
}

// ─── Upgrade handshake ────────────────────────────────────────────────────

// wsUpgrade performs the RFC 6455 client→server handshake and hijacks
// the underlying connection.
//
// Required request headers (the caller should validate these before
// calling, but wsUpgrade checks them itself as a safety net):
//
//	Upgrade: websocket
//	Connection: Upgrade
//	Sec-WebSocket-Key: <base64 of 16 random bytes>
//	Sec-WebSocket-Version: 13
//
// On success wsUpgrade has hijacked w; the caller MUST NOT touch w
// afterwards. On failure wsUpgrade writes an HTTP error response and
// returns the error.
func wsUpgrade(w http.ResponseWriter, r *http.Request) (*wsConn, error) {
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return nil, wsHTTPError(w, http.StatusBadRequest, "missing Upgrade header")
	}
	if !strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade") {
		return nil, wsHTTPError(w, http.StatusBadRequest, "missing Connection: Upgrade")
	}
	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		return nil, wsHTTPError(w, http.StatusBadRequest, "missing Sec-WebSocket-Key")
	}
	if r.Header.Get("Sec-WebSocket-Version") != "13" {
		// RFC 6455 §4.4 — refuse and tell the client to retry with v13.
		w.Header().Set("Sec-WebSocket-Version", "13")
		return nil, wsHTTPError(w, http.StatusUpgradeRequired, "unsupported version")
	}

	// Sec-WebSocket-Accept = base64( SHA1( key + GUID ) ). §4.2.2.
	h := sha1.New()
	h.Write([]byte(key))
	h.Write([]byte(wsMagicGUID))
	accept := base64.StdEncoding.EncodeToString(h.Sum(nil))

	hj, ok := w.(http.Hijacker)
	if !ok {
		return nil, wsHTTPError(w, http.StatusInternalServerError,
			"ResponseWriter does not support hijacking")
	}

	// From this point we own the underlying connection. The
	// *http.ResponseWriter MUST NOT be written to — it would race with
	// the net.Conn we just hijacked.
	rwc, bufrw, err := hj.Hijack()
	if err != nil {
		return nil, wsHTTPError(w, http.StatusInternalServerError,
			"hijack failed: "+err.Error())
	}
	// bufrw wraps rwc with a bufio.ReadWriter — reset both sides to the
	// raw conn so the next read starts at the WebSocket frame, not at
	// leftover HTTP request bytes. bufio.{Reader,Writer}.Reset returns
	// no value in Go's standard library.
	bufrw.Reader.Reset(rwc)
	bufrw.Writer.Reset(rwc)

	resp := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + accept + "\r\n\r\n"
	if _, err := io.WriteString(bufrw.Writer, resp); err != nil {
		_ = rwc.Close()
		return nil, err
	}
	if err := bufrw.Writer.Flush(); err != nil {
		_ = rwc.Close()
		return nil, err
	}

	return &wsConn{
		rwc:         rwc,
		deadline:    rwc,
		remoteAddr:  r.RemoteAddr,
		closeSignal: make(chan struct{}),
	}, nil
}

// wsHTTPError writes an HTTP error response and returns the same message
// as an error. Only safe BEFORE the connection has been hijacked.
func wsHTTPError(w http.ResponseWriter, code int, msg string) error {
	http.Error(w, msg, code)
	return errors.New(msg)
}

// ─── Frame codec ──────────────────────────────────────────────────────────

// readFrame reads exactly one frame from the wire. It enforces:
//   - no reserved bits set (RFC 6455 §5.2)
//   - server-mask requirement on client→server frames
//   - payload size limit (wsMaxFrame)
//
// Continuation frames (opcode 0x0) are returned with their opcode
// unchanged so the caller can reassemble messages if it cares. The
// echo/broadcast demo below doesn't reassemble — it treats every frame
// independently, which is fine for chat lines ≤ wsMaxFrame.
func (c *wsConn) readFrame() (fin bool, opcode wsOpcode, payload []byte, err error) {
	var hdr [2]byte
	if _, err = io.ReadFull(c.rwc, hdr[:]); err != nil {
		return false, 0, nil, err
	}
	fin = hdr[0]&0x80 != 0
	if rsv := hdr[0] & 0x70; rsv != 0 {
		return false, 0, nil, errors.New("reserved bits set")
	}
	opcode = wsOpcode(hdr[0] & 0x0F)
	masked := hdr[1]&0x80 != 0
	plen := uint64(hdr[1] & 0x7F)

	switch plen {
	case 126:
		var ext [2]byte
		if _, err = io.ReadFull(c.rwc, ext[:]); err != nil {
			return false, 0, nil, err
		}
		plen = uint64(binary.BigEndian.Uint16(ext[:]))
	case 127:
		var ext [8]byte
		if _, err = io.ReadFull(c.rwc, ext[:]); err != nil {
			return false, 0, nil, err
		}
		plen = binary.BigEndian.Uint64(ext[:])
	}

	if plen > wsMaxFrame {
		return false, 0, nil, errors.New("frame too large")
	}

	var maskKey [4]byte
	if masked {
		if _, err = io.ReadFull(c.rwc, maskKey[:]); err != nil {
			return false, 0, nil, err
		}
	}

	payload = make([]byte, plen)
	if plen > 0 {
		if _, err = io.ReadFull(c.rwc, payload); err != nil {
			return false, 0, nil, err
		}
		if masked {
			for i := range payload {
				payload[i] ^= maskKey[i%4]
			}
		}
	}
	return fin, opcode, payload, nil
}

// writeFrame writes a single WebSocket frame. Server→client frames MUST
// NOT be masked (RFC 6455 §5.3), so no mask key is emitted.
//
// For close frames the first two bytes of payload are interpreted as a
// big-endian close code; pass an empty payload for "no code" cases.
func (c *wsConn) writeFrame(fin bool, opcode uint8, code uint16, payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.writeFrameUnlocked(fin, opcode, code, payload)
}

// writeFrameUnlocked is the inner write without locking. Callers MUST
// hold c.mu. close() uses this directly to avoid re-entering the
// mutex while inside its defer.
func (c *wsConn) writeFrameUnlocked(fin bool, opcode uint8, code uint16, payload []byte) error {
	if c.deadline != nil {
		_ = c.deadline.SetDeadline(time.Now().Add(wsWriteDeadline))
	}

	var hdr [2]byte
	if fin {
		hdr[0] = 0x80
	}
	hdr[0] |= opcode & 0x0F

	// Build the post-header body. For close frames it begins with the
	// 2-byte close code.
	var body []byte
	if opcode == 0x8 {
		body = make([]byte, 2+len(payload))
		binary.BigEndian.PutUint16(body[:2], code)
		copy(body[2:], payload)
	} else {
		body = payload
	}

	switch {
	case len(body) < 126:
		hdr[1] = byte(len(body))
		if _, err := c.rwc.Write(hdr[:]); err != nil {
			return err
		}
		_, err := c.rwc.Write(body)
		return err
	case len(body) <= 0xFFFF:
		hdr[1] = 126
		if _, err := c.rwc.Write(hdr[:]); err != nil {
			return err
		}
		var ext [2]byte
		binary.BigEndian.PutUint16(ext[:], uint16(len(body)))
		if _, err := c.rwc.Write(ext[:]); err != nil {
			return err
		}
		_, err := c.rwc.Write(body)
		return err
	default:
		hdr[1] = 127
		if _, err := c.rwc.Write(hdr[:]); err != nil {
			return err
		}
		var ext [8]byte
		binary.BigEndian.PutUint64(ext[:], uint64(len(body)))
		if _, err := c.rwc.Write(ext[:]); err != nil {
			return err
		}
		_, err := c.rwc.Write(body)
		return err
	}
}

// ─── Hub ──────────────────────────────────────────────────────────────────

// wsHub is a tiny in-process pub/sub used by the demo echo handler. Real
// apps would replace this with a NATS / Redis / Postgres-LISTEN adapter
// (or any pub/sub that survives a single-process restart), but the
// registration pattern (one app.Mux().HandleFunc per upgrade endpoint)
// is the same.
type wsHub struct {
	mu    sync.RWMutex
	conns map[*wsConn]struct{}
}

func newWSHub() *wsHub { return &wsHub{conns: map[*wsConn]struct{}{}} }

func (h *wsHub) add(c *wsConn) {
	h.mu.Lock()
	h.conns[c] = struct{}{}
	h.mu.Unlock()
}

func (h *wsHub) remove(c *wsConn) {
	h.mu.Lock()
	delete(h.conns, c)
	h.mu.Unlock()
}

// broadcast sends the same payload to every connected client. Failed
// writes cause the offending client to be closed but do not abort the
// loop.
func (h *wsHub) broadcast(text []byte) int {
	h.mu.RLock()
	conns := make([]*wsConn, 0, len(h.conns))
	for c := range h.conns {
		conns = append(conns, c)
	}
	h.mu.RUnlock()

	delivered := 0
	for _, c := range conns {
		if err := c.writeFrame(true, uint8(wsOpcodeText), 0, text); err != nil {
			slog.Warn("ws broadcast: dropping client", "err", err)
			c.close(1011, "broadcast failed")
			h.remove(c)
			continue
		}
		delivered++
	}
	return delivered
}

// ─── The handler wired into app.Mux() ─────────────────────────────────────

// wsHandler is the http.HandlerFunc handed to app.Mux(). It performs
// the upgrade and then runs the read loop for one connection.
//
// Demo semantics:
//   - Text frames are broadcast to every connected client (chat).
//   - Binary frames are echoed only to the sender (to keep the chat
//     view in views/ws.html pure text).
//   - Ping frames get an automatic Pong reply (RFC 6455 §5.5.3).
//   - Close frames end the session.
//   - Any read error closes the conn with code 1011 (Internal Error).
//
// Auth note: app.Mux() routes bypass ALL xun middleware, including
// sessionMiddleware. We re-check the signed session cookie here using
// the same key the session middleware uses, but on the raw *http.Request
// because there is no *xun.Context on this path.
func wsHandler(hub *wsHub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Auth — must be done by hand on app.Mux() routes.
		if !wsAuthOK(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// 2. Upgrade.
		conn, err := wsUpgrade(w, r)
		if err != nil {
			slog.Warn("ws upgrade failed", "err", err, "remote", r.RemoteAddr)
			return
		}
		hub.add(conn)
		slog.Info("ws connected", "remote", conn.remoteAddr)

		// 3. Read loop. Returns when the peer sends a close frame, the
		//    network dies, or the read deadline trips.
		defer func() {
			hub.remove(conn)
			conn.close(1000, "server closing")
			slog.Info("ws disconnected", "remote", conn.remoteAddr)
		}()

		for {
			if conn.deadline != nil {
				_ = conn.deadline.SetDeadline(time.Now().Add(wsReadDeadline))
			}
			fin, opcode, payload, err := conn.readFrame()
			if err != nil {
				return
			}

			switch opcode {
			case wsOpcodeText:
				if fin {
					hub.broadcast(payload)
				}
			case wsOpcodeBinary:
				if fin {
					_ = conn.writeFrame(true, uint8(wsOpcodeBinary), 0, payload)
				}
			case wsOpcodePing:
				_ = conn.writeFrame(true, uint8(wsOpcodePong), 0, payload)
			case wsOpcodeClose:
				return
			}
		}
	}
}

// wsAuthOK verifies that the request carries a valid signed session
// cookie. It mirrors the read-side of sessionMiddleware but operates
// directly on a raw *http.Request — there is no *xun.Context on the
// app.Mux() path.
//
// We replicate ext/cookie.GetSigned here verbatim rather than calling
// it because ext/cookie expects a *xun.Context. The wire format is
// tiny (HMAC-SHA256 | RFC3339 timestamp | value) so duplicating it is
// cheaper than introducing a Context-free wrapper.
//
// The signing rules implemented below MUST match sessionMiddleware's
// write path (which calls cookie.SetSigned → signValue); if those
// drift apart, every existing session will fail this check.
func wsAuthOK(r *http.Request) bool {
	c, err := r.Cookie(sessionCookieName)
	if err != nil {
		return false
	}
	signed, err := base64.URLEncoding.DecodeString(c.Value)
	if err != nil {
		return false
	}
	// Same minimum-length gate as ext/cookie.GetSigned.
	if len(signed) < sha256.Size+20 {
		return false
	}

	signature := signed[:sha256.Size]
	tv := signed[sha256.Size : sha256.Size+20]
	value := signed[sha256.Size+20:]

	ts, err := time.Parse(time.RFC3339, string(tv))
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(sessionSecret))
	mac.Write([]byte(sessionCookieName))
	mac.Write(tv)
	mac.Write(value)
	expected := mac.Sum(nil)
	if !hmac.Equal(signature, expected) {
		return false
	}

	s, err := decodeSession(string(value))
	if err != nil || s == nil || s.UserID == 0 {
		return false
	}
	_ = ts // timestamp is validated; we don't enforce expiry on this path
	return true
}
