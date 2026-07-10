# Xun Web Starter

A reference implementation that demonstrates **idiomatic use of the [xun](https://github.com/yaitoo/xun) Go web framework**.

Every file in this repository is a deliberate example of one xun pattern. Read the source top-down — `cmd/app/main.go`, `cmd/app/routes.go`, `cmd/app/handlers.go`, `cmd/app/middleware.go` — and you have seen the standard xun application shape. Use it as a template when generating a new xun app or when extending an existing one.

> **Audience:** This README is written for an AI agent learning how to write xun applications. It focuses on **which xun API does what** and **where in the code each pattern lives**, not on the demo product itself.

---

## 1. Repository Map (xun concepts)

```
cmd/app/
├── main.go               # App construction: xun.New, WithMux, WithFsys, WithWatch,
│                          # WithHandlerViewers, WithInterceptor, WithBuildAssetURL, app.Use,
│                          # plus the call to setupNativeRoutes (see §10)
├── app.yml               # viper config; loaded by main.go → drives listeners, DSN, session secret
├── routes.go             # app.HandleFunc, app.Get/Post/Put/Delete, app.Group + Use (sub-app),
│                          # AND setupNativeRoutes (the app.Mux() escape hatch)
├── handlers.go           # xun.HandleFunc signature; c.View / c.Set / c.Get / c.Redirect / c.WriteHeader
├── middleware.go         # xun.HandleFunc composition (next ... error)
├── auth.go               # helpers; session encoding (no xun API)
├── db.go                 # pure stdlib + yaitoo/sqle (no xun API)
├── ws.go                 # WebSocket server (RFC 6455) wired via app.Mux(); shows why
│                          # xun's high-level API is incompatible with Hijack() and how
│                          # the Mux() escape hatch re-implements session auth by hand
└── app/
    ├── layouts/          # HTML page skeletons. Referenced via <!--layout:NAME--> in pages/views
    ├── pages/            # SSR pages; each page auto-registers a route via xun's PageRoute
    │   │                  # Pages sit in sub-dirs to form path params:
    │   │                  #   pages/dashboard/index.html    → GET /dashboard/{$}
    │   │                  #   pages/dashboard/users.html    → GET /dashboard/users
    │   │                  #   pages/dashboard/users/{id}.html → GET /dashboard/users/{id}
    │   └── ws.html        # Demo page: dashboard/ws → /api/ws upgrade (browser console)
    ├── components/       # Reusable HTML fragments loaded via {{template "name" .}}
    ├── views/            # htmx partial responses: handler calls c.View(..., "views/...") (no layout)
    └── public/           # Static assets; //go:embed-mounted in main.go
```

---

## 2. The xun Application Lifecycle (`main.go`)

The standard shape of `func main()`:

```go
// 1. Load config (viper; not xun-specific but conventionally done first)
err := loadConfig()

// 2. Open the database / external resources
db, err := setupSQLite(ctx)

// 3. Construct the xun.App with options
mux := http.NewServeMux()
app := createApp(mux)

// 4. Register routes (xun-managed — go through middleware pipeline)
setupRoutes(app)

// 4b. Register native routes (raw *http.ServeMux — bypass middleware entirely;
//     needed for protocols that Hijack the connection — see §10)
setupNativeRoutes(app, newWSHub())

// 5. Start the app (registers handlers on mux; non-blocking)
app.Start()

// 6. Run HTTP/HTTPS listeners (blocking until shutdown)
runServers(mux)
```

### 2.1 `createApp` — wiring xun options

```go
app := xun.New(
    xun.WithMux(mux),                              // share an existing *http.ServeMux
    xun.WithFsys(getFsys()),                       // template/asset fs.FS (dev: ./app; prod: embed)
    xun.WithWatch(),                               // live-reload templates in dev
    xun.WithHandlerViewers(&xun.JsonViewer{}),     // content negotiation: *.json → JSON
    xun.WithInterceptor(htmx.New()),               // installs htmx request/response helpers
    xun.WithBuildAssetURL(func(path string) bool {
        return strings.HasPrefix(path, "/assets/")
    }),
)
```

**Patterns demonstrated:**

| Option | Purpose | When to use |
|--------|---------|-------------|
| `WithMux` | Inject your own `*http.ServeMux` | When you need a stdlib mux (e.g. for `ListenAndServeTLS`) |
| `WithFsys` | Templates + assets `fs.FS` | Always — points to `app/` directory in dev, embed in prod |
| `WithWatch` | Live-reload templates | Dev only; adds a file watcher |
| `WithHandlerViewers` | Alternate content renderers | Add `JsonViewer`, `XmlViewer` for API responses |
| `WithInterceptor` | Cross-cutting wrappers | `htmx.New()` (this project); custom auth, logging, etc. |
| `WithBuildAssetURL` | Asset URL strategy | Return `true` to route through `{{asset ...}}` (cache-bust hashing) |

### 2.2 `app.Use` — global middleware

```go
app.Use(sessionMiddleware,
    cache.New(
        cache.Match("/assets/", "", 7*24*time.Hour),
        cache.Match("", "favicon.ico", 365*24*time.Hour),
    ),
)
```

- `app.Use(...)` registers middleware that wraps **every** handler.
- Order matters: middleware runs in registration order on the way in, reverse on the way out.
- Use it for cross-cutting concerns (session, caching, request logging, recovery).

### 2.3 Embedded filesystem pattern

```go
//go:embed app/components
//go:embed app/layouts
//go:embed app/pages
//go:embed app/public
//go:embed app/views
var fsys embed.FS
```

```go
func getFsys() fs.FS {
    if fi, err := os.Stat("./app"); err == nil && fi.IsDir() {
        return os.DirFS("./app") // dev: live files
    }
    app, _ := fs.Sub(fsys, "app") // prod: embedded
    return app
}
```

This is the canonical **"dev/prod twin fs.FS"** pattern in xun apps.

---

## 3. Routing (`routes.go`)

### 3.1 Flat handlers

```go
app.HandleFunc("GET /htmx.js", htmx.HandleFunc())
app.Get("/",      handleLanding)
app.Post("/login", handleLogin)
app.Put("/users/{id}",    handleUserUpdate)
app.Delete("/users/{id}", handleUserDelete)
```

- Method-prefixed methods (`Get`, `Post`, `Put`, `Delete`) are shortcuts over `HandleFunc("METHOD /path", h)`.
- Path parameters (`/users/{id}`) are read with `c.Request.PathValue("id")`.
- `htmx.HandleFunc()` is the built-in htmx runtime endpoint.

### 3.2 Group + scoped middleware

```go
dashboard := app.Group("/dashboard")
dashboard.Use(authMiddleware)
dashboard.Get("/",        handleDashboard)
dashboard.Get("/users",   handleUserList)
dashboard.Post("/users",  handleUserCreate)
dashboard.Put("/users/{id}",    handleUserUpdate)
dashboard.Delete("/users/{id}", handleUserDelete)
dashboard.Get("/views/users/list",    handleUserListView)
dashboard.Get("/views/users/row/{id}", handleUserRowView)
```

**This is the canonical auth-protected route group pattern in xun:**
- `app.Group(prefix)` creates a sub-app.
- Sub-app `.Use(...)` registers middleware **only for routes inside the group**.
- The middleware decorator is a normal `xun.HandleFunc` wrapper — no special API.

### 3.3 The auth middleware shape

```go
authMiddleware := func(next xun.HandleFunc) xun.HandleFunc {
    return func(c *xun.Context) error {
        session, _ := c.Get("session").(*Session)
        if session == nil || session.UserID == 0 {
            if isHTMXRequest(c) {
                c.WriteHeader(htmx.HxRedirect, "/login")
                c.WriteStatus(200)
                return nil
            }
            c.Redirect("/login")
            return nil
        }
        c.Set("user_id", session.UserID) // stash typed values on the context
        return next(c)
    }
}
```

Key facts:
- Middleware is `func(xun.HandleFunc) xun.HandleFunc`.
- Return `next(c)` to continue; return `nil` to short-circuit.
- Use `c.Set(k, v)` / `c.Get(k)` to pass typed values down the chain (like `context.Context` but app-scoped). These operate on `c.TempData` — see §4.5 for the `.Data` vs `.TempData` distinction.
- `c.Redirect(writeHeader)` and `c.WriteHeader(name, value)` are the response APIs.

---

## 4. Handlers (`handlers.go`)

The standard handler signature:

```go
func handleLanding(c *xun.Context) error {
    return c.View(map[string]any{
        "title": "Xun Web Starter",
    }, "pages/index")
}
```

### 4.1 `c.View` — the template rendering API

```go
c.View(data any, name ...string) error
```

- `data` becomes the template's `.Data` (see §4.5).
- `name` is the optional **viewer key** — a short identifier, NOT a file path. It is registered by the view engine:
  - **Pages** (`app/pages/*.html`) are registered with the key being the path WITHOUT the `pages/` prefix and the `.html` suffix. So `pages/index.html` → `"index"`, `pages/dashboard/users/{id}.html` → `"dashboard/users/{id}"`.
  - **Views** (`app/views/*.html`) keep the full relative path as key: `views/users/list` → `"views/users/list"`, `views/error-message` → `"views/error-message"`.

#### When can you omit `name`?

**Omit `name` when the route is already a PageRoute.** The HtmlViewer's MIME type (`text/html`) is in `c.Routing.Viewers` because the page file's auto-registration put it there. xun then picks the right viewer by matching the request's `Accept` header — `text/html` browser request → `HtmlViewer`, `application/json` API request → `JsonViewer`.

This means **every handler that sits on top of a `pages/*.html` file can call `c.View(data)` with no second argument**, because the page file already "is" the view — picking it is just a matter of content negotiation.

Keep `name` only when:

1. The view file lives under `app/views/` (not `app/pages/`), so no PageRoute auto-registered it on this handler's route. Example: `renderHXRetarget` calls `c.View(nil, "views/error-message")` because the error fragment is shared by many routes, none of which own it.
2. The handler serves multiple views conditionally (rare).

| Handler call | When to use |
|--------------|-------------|
| `c.View(data)` | On a PageRoute (`pages/<X>.html` auto-registered the viewer). xun picks HtmlViewer / JsonViewer from the route's viewers via Accept. |
| `c.View(data, "index")` | On a manually-registered route (`app.Get` / `dashboard.Get`) that needs to pick a specific page viewer. |
| `c.View(data, "views/users/list")` | For shared partials under `app/views/` that have no owning route. |

### 4.2 Layout declaration

Each page picks its layout via an HTML comment on line 1:

```html
<!--layout:base-->
{{define "content"}}
<div>...page body...</div>
{{end}}
```

```html
<!--layout:dashboard-->
{{define "content"}}
<div>...page body...</div>
{{end}}
```

The page's body MUST be wrapped in `{{define "content"}}...{{end}}` — this is the block that the layout's `{{block "content" .}}{{end}}` placeholder fills. (If you forget the `{{define}}`, the page body never renders — the layout runs with an empty content block.)

The layout file contains the chrome and a `{{block "content" .}}{{end}}` placeholder where the page body is injected. For including shared components (`nav`, `footer`, etc.) use `{{block "components/NAME" .}}{{end}}` — the block name must match the registered template name (path under `components/` without `.html`).

**Block resolution rules:**

| Block name | Must be provided by | Example |
|------------|---------------------|---------|
| `content` | Every page via `{{define "content"}}...{{end}}` | Required in all pages |
| `components/<name>` | File `app/components/<name>.html` | `{{block "components/nav" .}}` → `app/components/nav.html` |
| Custom optional blocks | **Either** a page `{{define "name"}}` **or** a component file `app/components/<name>.html` | See note below |

**Important naming rule - depends on call site:**

| From → To | Syntax | Example |
|-----------|--------|---------|
| **Layout → Component** | `{{block "components/<name>" .}}` | `{{block "components/nav" .}}` |
| **Page → Definition** | `{{define "content"}}` or `{{define "custom"}}` | Required for every `{{block "content"}}` in layout |
| **Component → Component** | `{{template "<name>" .}}` | `{{template "user-item" .}}` (no `components/` prefix) |

The same file `app/components/theme-toggle.html` is referenced differently depending on where it's called:
- From a layout: `{{block "components/theme-toggle" .}}`
- From another component: `{{template "theme-toggle" .}}`

**Important:** Unlike Go's default `{{block}}` behavior, xun requires that every block referenced in a layout must be backed by either:
1. A `{{define}}` in the page file, or
2. A component file in `app/components/`

If a layout contains `{{block "scripts-extra" .}}{{end}}` but no page defines it and no `app/components/scripts-extra.html` exists, you'll get a runtime error: `no such template "scripts-extra"`. 

**Current implementation in this repo:**

All pages in this repo use one of two layouts, and every block referenced has proper backing:

- `layouts/base.html` uses: `components/nav`, `content`, `components/footer`
- `layouts/dashboard.html` uses: `components/dashboard-nav`, `content`, `components/footer`

All component files exist, and all pages define `{{define "content"}}`. This is the **recommended safe pattern**.

**If you need optional per-page blocks (advanced pattern):**
```html
<!-- Layout: layouts/base.html -->
{{block "head-extra" .}}{{end}}

<!-- Page that uses it: pages/index.html -->
<!--layout:base-->
{{define "content"}}...{{end}}
{{define "head-extra"}}<script>...</script>{{end}}

<!-- Page that doesn't use it: pages/about.html -->
<!--layout:base-->
{{define "content"}}...{{end}}
{{define "head-extra"}}{{end}}  <!-- Must provide empty definition -->
```

Note: This repo does not use optional blocks. All blocks are backed by either component files or page definitions.

### 4.3 Reading request data

```go
email   := c.Request.FormValue("email")            // POST form fields
id, _   := strconv.ParseInt(c.Request.PathValue("id"), 10, 64) // path params
session := c.Get("session").(*Session)             // typed context value
```

### 4.4 Writing responses

```go
return c.View(data)                                           // render HTML on a PageRoute handler
c.Redirect("/login")                                          // 302
c.WriteHeader(htmx.HxRedirect, "/login"); c.WriteStatus(200)  // htmx redirect
c.WriteHeader(htmx.HxRetarget, "#users-list")                 // htmx swap target
c.WriteStatus(http.StatusNotFound)                            // bare status
```

### 4.5 `.Data` vs `.TempData` — what goes where

The template's `.` is bound to a `ViewModel{TempData, Data}` — two distinct maps that are passed together to the template engine. **Pick the right one per value or you'll get a subtle rendering bug** (e.g. `{{.title}}` returns empty because `title` lives on TempData, not Data).

| | `.Data` | `.TempData` |
|---|---------|-------------|
| **Holds** | The **current page's main business object** (a `User`, a `[]User`, a struct, anything passed as `c.View(data, name)`'s first argument) | **Auxiliary chrome values** shared across the request (the current `Session`, the page `title`, one-off `message` text, anything set via `c.Set(k, v)` in middleware / handler) |
| **Set in Go** | `c.View(data, name)` first arg (or `c.View(data)` on PageRoutes) | `c.Set(k, v)` and `c.Get(k)` (both operate on TempData) |
| **Accessed in templates** | `{{.Data.X}}` (or `{{.X}}` after `{{range .Data}}`) | `{{.TempData.X}}` |
| **Lifetime** | Single `c.View(...)` call — the value passed to the current render only | Whole request — middleware → handler → view all share the same map |

#### Rules of thumb

1. **Page-level business data → `.Data`.** If you ran a SQL query in the handler and want to render the rows, that's `.Data.users` (`{{range .Data}} ... {{end}}`).
2. **Cross-page auxiliary state → `.TempData`.** The current session, page title, error messages, request-scoped flags.
3. **Don't duplicate.** If `c.Set("session", s)` already ran in middleware, don't also pass `session` in `c.View`'s data — it's already on TempData.
4. **Viewers and layouts both see TempData.** A layout's `{{.TempData.title}}` and a page's `{{.TempData.session.Name}}` work the same way because the whole `ViewModel` is passed to the template's `.`.

#### Concrete patterns in this repo

```go
// ── Business data via .Data ─────────────────────────────
// Handler: pass a struct or slice directly as the .Data arg.
func handleUserList(c *xun.Context) error {
    var users []User
    // ... query ...
    return c.View(users)  // ← PageRoute handler; viewer name omitted
}

// Template:
{{range .Data}}
  <tr><td>{{.Name}}</td></tr>
{{end}}

// ── Auxiliary data via .TempData ────────────────────────
// Handler: page title (chrome, not business model) goes on TempData.
func handleDashboard(c *xun.Context) error {
    c.Set("title", "Dashboard")           // ← .TempData.title
    return c.View(nil)                    // ← PageRoute handler
}

// Layout: read it from TempData.
<title>{{.TempData.title}} - Xun Web</title>
```



### 4.6 The htmx partial pattern

This is the most important idiom in the repo — used for inline CRUD:

```go
func handleUserCreate(c *xun.Context) error {
    // ... validate, insert into db ...
    return handleUserListView(c) // <-- delegate to the partial-view handler
}

func handleUserListView(c *xun.Context) error {
    rows, _ := db.QueryContext(ctx, "SELECT ...")
    // ... scan into []User ...
    return c.View(users)  // ← PageRoute handler; .Data = []User
}
```

The page handler returns the **same fragment** that the page's list container requests via `hx-get`, so after create/update the client just swaps in fresh HTML. No JSON, no client-side rendering.

The retarget pattern for inline errors:

```go
func renderHXRetarget(c *xun.Context, target, message string) error {
    c.WriteHeader(htmx.HxRetarget, "#"+target)             // change swap target
    c.Set("message", message)                              // ← .TempData.message
    return c.View(nil, "views/error-message")              // explicit viewer (shared partial)
}
```

---

## 5. Middleware (`middleware.go`)

The session middleware demonstrates the **post-handler write** pattern — middleware that mutates the response *after* `next(c)` returns:

```go
func sessionMiddleware(next xun.HandleFunc) xun.HandleFunc {
    return func(c *xun.Context) error {
        // ── Pre-handler: read state into context ──
        value, _, err := cookie.GetSigned(c, "xun_web_session", []byte(sessionSecret))
        if err == nil {
            if s, err := decodeSession(value); err == nil && s != nil {
                c.Set("session", s)
            }
        }

        // ── Run the handler ──
        err = next(c)

        // ── Post-handler: persist state back to response ──
        if s, ok := c.Get("session").(*Session); ok && s != nil {
            cookie.SetSigned(c, http.Cookie{ /* ... */ }, []byte(sessionSecret))
        }
        return err
    }
}
```

Use this pattern whenever your middleware needs to *serialize* state after the handler ran (sessions, CSRF tokens, request IDs, etc.).

---

## 6. Templates & Components

### 6.1 Asset helper

```html
<link rel="stylesheet" href="{{asset "/assets/app.css"}}">
<script src="{{asset "/htmx.js"}}"></script>
```

`{{asset PATH}}` rewrites `PATH` through `WithBuildAssetURL` — typically appending a content hash for cache-busting.

### 6.2 Component inclusion

Layouts include components using the `{{block}}` directive with the full template path:

**Example from `layouts/base.html` in this repo:**
```html
<body class="min-h-screen flex flex-col">
  {{block "components/nav" .}}{{end}}

  <main class="flex-1">
    {{block "content" .}}{{end}}
  </main>

  {{block "components/footer" .}}{{end}}
</body>
```

**Example from `layouts/dashboard.html` in this repo:**
```html
<body class="min-h-screen bg-dark-50">
  {{block "components/dashboard-nav" .}}{{end}}

  <main class="container mx-auto px-4 py-8 max-w-7xl">
    {{block "content" .}}{{end}}
  </main>

  {{block "components/footer" .}}{{end}}
</body>
```

**Important:** The block name must match the registered template name:
- `{{block "components/nav" .}}` → requires file `app/components/nav.html` ✓
- `{{block "components/dashboard-nav" .}}` → requires file `app/components/dashboard-nav.html` ✓
- `{{block "components/footer" .}}` → requires file `app/components/footer.html` ✓
- `{{block "content" .}}` → must be defined in every page via `{{define "content"}}` ✓

**Available components in this repo:**
- `components/nav.html` - Main navigation
- `components/dashboard-nav.html` - Dashboard navigation
- `components/footer.html` - Footer
- `components/user-item.html` - User item component

### 6.3 Nested component references (component → component)

**Important naming difference:** When one component includes another component, use the base name without the `components/` prefix:

```html
<!-- In components/nav.html -->
{{template "theme-toggle" .}}  ← Use base name only, no "components/" prefix
```

**Naming rule by call site:**

| From → To | Syntax | Example |
|-----------|--------|---------|
| Layout → Component | `{{block "components/<name>" .}}` | `{{block "components/nav" .}}` |
| Component → Component | `{{template "<name>" .}}` | `{{template "theme-toggle" .}}` |

The same file `app/components/theme-toggle.html` is referenced differently:
- From a layout: `{{block "components/theme-toggle" .}}`
- From another component: `{{template "theme-toggle" .}}`

**Why this difference?** When xun loads component files, they're registered with the `components/` path for layout references. But within the component template itself, sibling templates are referenced by base name only.

**Note:** This repo currently doesn't use nested components. All components are flat and only included from layouts.

### 6.3 Method calls on data

```html
{{.user.Initials}}
{{.user.CreatedAt.Format "2006-01-02"}}
```

The template engine resolves method calls on the bound struct, so helpers like `(*User).Initials()` in `auth.go` are first-class template functions. No `template.FuncMap` registration needed.

### 6.4 Branching & iteration

```html
{{if .session}} ... {{else}} ... {{end}}
{{range .users}} ... {{else}} (empty state) {{end}}
```

Standard Go `text/template` syntax.

---

### 6.5 PageRoute — pages with path parameters

This is the canonical xun idiom for "render a server-side page bound to a URL with optional path parameters". It's the simplest form of SSR in xun: zero JS, one HTML file, one Go handler.

#### The pattern

```text
app/pages/<sub-path>.../<name>.html     ← file
                  └──┬──┘                ← registers `GET <sub-path>.../<name>`
                     │
                     └── segments like `{id}` become Go 1.22 ServeMux path params
```

Concrete example from this repo — `pages/dashboard/users/{id}.html`:

```html
<!--layout:dashboard-->
{{define "content"}}
<h1>User #{{.Data.ID}}</h1>
<p>{{.Data.Name}} ({{.Data.Email}})</p>
{{end}}
```

The file's path auto-registers the route `GET /dashboard/users/{id}` and binds the viewer to it. The default page handler passes `nil` data, so you override it with `app.Get` / `dashboard.Get` to supply the actual model:

```go
// routes.go
dashboard.Get("/users/{id}", handleUserDetail)

// handlers.go
func handleUserDetail(c *xun.Context) error {
    id, _ := strconv.ParseInt(c.Request.PathValue("id"), 10, 64)
    var u User
    db.QueryRowContext(ctx, "SELECT ... WHERE id = ?", id).Scan(&u.ID, &u.Name, ...)
    c.Set("title", "User #"+strconv.FormatInt(u.ID, 10))   // ← .TempData.title
    return c.View(u)                                     // ← .Data = u
}
```

#### Why `/users/{id}` (Go 1.22 path syntax) and not `:id`?

xun uses Go 1.22's `http.ServeMux` patterns. The braces form (`{id}`) is the standard for Go 1.22 — it's read with `r.PathValue("id")`, same as `c.Request.PathValue("id")`. Older `:id` colon-style does NOT work with xun.

#### The `/{$}` root pattern

`pages/index.html` auto-registers `GET /{$}` (not `GET /`). If you want to supply data to the root page, you must register against the same pattern:

```go
// Right — matches the auto-registered pattern
app.Get("/{$}", handleLanding)

// Wrong — different pattern, doesn't override
app.Get("/", handleLanding)
```

This quirk comes from xun's `splitFile` appending `{$}` to any pattern ending in `/`, so `pages/index.html` becomes `GET /{$}`. Same applies to `pages/<group>/index.html` files inside a `app.Group(...)`.

---

## 6.6 Native Routing via `app.Mux()`

`app.Mux()` returns the underlying `*http.ServeMux`. It is the **escape hatch** for behaviour xun's high-level API cannot express — anything that needs the **raw `http.ResponseWriter`**:

- WebSocket / SSE / HTTP/2 stream upgrades (`http.Hijacker`, `http.Flusher`)
- Third-party `http.Handler` integrations (Prometheus, pprof, OpenTelemetry exporters)
- Catch-all 404 fallbacks (lower precedence than specific routes)

### Semantics (read carefully)

| Property | `app.Get / Post / HandleFunc` | `app.Mux().Handle / HandleFunc` |
|----------|--------------------------------|--------------------------------|
| Runs `app.Use` middleware | ✓ | **✗** |
| Runs `group.Use` middleware | ✓ | **✗** |
| Goes through viewer / content negotiation | ✓ | **✗** |
| Logs `X-Log-Id` | ✓ | **✗** |
| Listed in `app.Start()` startup log | ✓ | **✗** |
| Error → status mapping by xun | ✓ | **✗** |
| Auth (session, CSRF, etc.) | automatic | **must be done by hand** |

### The pattern

```go
// routes.go
func setupNativeRoutes(app *xun.App, hub *wsHub) {
    // Standard Go 1.22 ServeMux pattern syntax.
    app.Mux().HandleFunc("GET /api/ws", wsHandler(hub))
}
```

Call this *after* `setupRoutes(app)` so xun-managed routes are registered first (route precedence doesn't actually matter for non-overlapping patterns, but the ordering is easier to reason about).

### Footguns

- **Always pass `WithMux(http.NewServeMux())`** to `xun.New`. If you skip it, `app.Mux()` falls back to `http.DefaultServeMux` and you pollute global state.
- Patterns auto-registered by xun (e.g. `GET /{$}` from `pages/index.html`) **cannot** be re-registered through `app.Mux()` — ServeMux panics on duplicate patterns.
- Registering on `app.Mux()` from multiple goroutines concurrently panics. Do all registration before serving starts.

See **§10** for the full WebSocket example that motivates this API.

---

## 7. Configuration & Lifecycle

### 7.1 viper + APP_* env convention

```go
viper.SetEnvPrefix("APP")
viper.AutomaticEnv()
viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
```

Mapping: `APP_SESSION_SECRET` → `session.secret`, `APP_DB_DSN` → `db.dsn`. Defaults come from `app.yml`; env vars override.

### 7.2 CLI flag

```go
flag.StringVar(&conf, "conf", "app.yml", "config file")
flag.Parse()
```

Single `-conf` flag. Search order: explicit path → binary dir → `cmd/app/` → cwd.

### 7.3 Graceful shutdown

```go
sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
<-sigCh
srv.Shutdown(ctx) // shutdownTimeout = 15s
```

### 7.4 Build-time identity

```go
var (
    Version = "dev"     // overridden via -ldflags -X 'main.Version=...'
    Commit  = "unknown" // overridden via -ldflags -X 'main.Commit=...'
)
```

Injected by the Makefile (`make build` → `git describe` / `git rev-parse`).

---

## 8. Database (not xun-specific, but the canonical pattern)

`db.go` uses `cnlangzi/sqlite` + `yaitoo/sqle` + `yaitoo/sqle/migrate`:

```go
//go:embed migrations
var migrations embed.FS

db, _ = sqlite.New(ctx, dsn)
migrator := migrate.New(sqle.Open(db.Writer.DB))
migrator.Discover(migrations)
migrator.Init(ctx)
migrator.Migrate(ctx)
```

Migrations live in `cmd/app/migrations/<version>/<seq>_<name>.sql` and run automatically at startup — no manual migrate step. This is the recommended pattern for any xun + SQLite app.

---

## 9. htmx Patterns Catalogue

This repo demonstrates almost every htmx-on-server idiom:

| Pattern | Where | Code |
|---------|-------|------|
| `hx-post` form submit, redirect on success | login / register forms | `handleLogin`, `handleRegister` |
| `hx-get` load fragment into target | inline edit form | `Add User` / `Edit` buttons → `views/users/form`, `views/users/edit/{id}` |
| `hx-delete` with `hx-swap="delete"` | row removal | `handleUserDelete` + `views/users/row.html` |
| `hx-trigger="reloadUsers from:body"` | manual list refresh | `pages/users.html` list container |
| `HX-Redirect` for navigation after htmx request | login/register/logout | `c.WriteHeader(htmx.HxRedirect, "/dashboard")` |
| `HX-Retarget` to swap error into specific zone | inline form errors | `renderHXRetarget` |
| `htmx.js` runtime served by framework | `app.HandleFunc("GET /htmx.js", htmx.HandleFunc())` | — |
| `hx-confirm` for destructive actions | user delete button | — |
| `htmx.New()` interceptor registered on the app | `main.go` `createApp` | — |

---

## 10. WebSocket via `app.Mux()` — full example

This is the canonical reason to reach for `app.Mux()`. The WebSocket protocol requires `http.Hijacker`, which **xun's response writer does not implement** (xun wraps the writer to do error-to-status mapping and middleware plumbing, but the wrapper has no `Hijack()` method). Any code path that upgrades the HTTP connection has to register on the underlying `*http.ServeMux`.

### 10.1 The upgrade itself

`cmd/app/ws.go → wsUpgrade`:

```go
func wsUpgrade(w http.ResponseWriter, r *http.Request) (*wsConn, error) {
    // 1. Validate RFC 6455 handshake headers.
    // 2. Compute Sec-WebSocket-Accept = base64( SHA1(key + GUID) ).
    // 3. Hijack the underlying net.Conn — this is why we MUST be on
    //    the raw *http.ServeMux; xun.ResponseWriter has no Hijacker.
    hj, ok := w.(http.Hijacker)
    if !ok {
        return nil, errors.New("ResponseWriter does not support hijacking")
    }
    rwc, bufrw, err := hj.Hijack()
    if err != nil { /* ... */ }

    // 4. Write the 101 Switching Protocols response BY HAND on the
    //    hijacked conn. From this point on, w is unusable.
    io.WriteString(bufrw.Writer,
        "HTTP/1.1 101 Switching Protocols\r\n"+
            "Upgrade: websocket\r\n"+
            "Connection: Upgrade\r\n"+
            "Sec-WebSocket-Accept: "+accept+"\r\n\r\n")
    bufrw.Writer.Flush()

    return &wsConn{rwc: rwc, deadline: rwc, /* ... */}, nil
}
```

### 10.2 Auth on the bypassed path

`app.Mux()` routes skip `sessionMiddleware`. If you need the same auth check, re-implement it on the raw request. In `cmd/app/ws.go → wsAuthOK`:

```go
// Replicate ext/cookie.GetSigned against a raw *http.Request.
// MUST stay in sync with the signing rules in cmd/app/middleware.go.
func wsAuthOK(r *http.Request) bool {
    c, err := r.Cookie(sessionCookieName)
    if err != nil { return false }
    signed, err := base64.URLEncoding.DecodeString(c.Value)
    if err != nil { return false }

    signature := signed[:sha256.Size]
    tv := signed[sha256.Size : sha256.Size+20]
    value := signed[sha256.Size+20:]

    mac := hmac.New(sha256.New, []byte(sessionSecret))
    mac.Write([]byte(sessionCookieName))
    mac.Write(tv)
    mac.Write(value)
    return hmac.Equal(signature, mac.Sum(nil))
}
```

If you change `sessionMiddleware` (e.g. switch to a different cookie library), this function must change too — there's no compile-time check that they agree.

### 10.3 Frame I/O

The RFC 6455 frame format is small enough to implement inline (see `ws.go → readFrame` and `writeFrame`). Two patterns worth noting:

```go
// Client→server frames MUST be masked (RFC 6455 §5.3). Apply the XOR
// mask immediately after reading the payload, before any further
// processing.
if masked {
    for i := range payload {
        payload[i] ^= maskKey[i%4]
    }
}

// Server→client frames MUST NOT be masked. Never write a mask key.
```

### 10.4 Deadlines

The interface returned by `hj.Hijack()` is `io.ReadWriteCloser`, which has **no** `SetDeadline`. To set per-read or per-write timeouts, capture the deadline capability at upgrade time:

```go
type wsConn struct {
    rwc      io.ReadWriteCloser
    deadline interface{ SetDeadline(time.Time) error } // ← satisfies *net.TCPConn
    // ...
}

// In the read loop:
if conn.deadline != nil {
    _ = conn.deadline.SetDeadline(time.Now().Add(wsReadDeadline))
}
```

### 10.5 Demo wiring

```go
// routes.go
func setupNativeRoutes(app *xun.App, hub *wsHub) {
    app.Mux().HandleFunc("GET /api/ws", wsHandler(hub))
}

// main.go — after setupRoutes(app)
setupNativeRoutes(app, newWSHub())
```

Visit `http://localhost:8080/dashboard/ws` (must be logged in) to see the browser console that connects to `/api/ws`.

---

## 11. Quick Reference — Cheat Sheet

| Need | API |
|------|-----|
| Build the app | `xun.New(opts...)` |
| Add global middleware | `app.Use(mw...)` |
| Group routes | `g := app.Group("/prefix"); g.Use(authMw)` |
| Register a route | `app.Get|Post|Put|Delete("/path", handler)` or `app.HandleFunc("METHOD /path", h)` |
| Render a page (PageRoute handler) | `c.View(data)` — viewer already on the route's viewer list |
| Render a page (manual handler) | `c.View(data, "X")` — explicit viewer key |
| Render a shared partial | `c.View(data, "views/X")` — must pass name (no owning route) |
| Pick a layout | `<!--layout:NAME-->` first line of the page |
| Include component in layout | `{{block "components/nav" .}}{{end}}` — must match file path |
| Include component in component | `{{template "user-item" .}}` — use base name, no `components/` prefix |
| Page body block | `{{define "content"}}...{{end}}` — required in every page |
| Optional page block | Must define in every page (can be empty): `{{define "head-extra"}}{{end}}` |
| Read form field | `c.Request.FormValue("k")` |
| Read path param | `c.Request.PathValue("k")` |
| Stash typed value | `c.Set("k", v)` |
| Get typed value | `v, ok := c.Get("k").(T)` |
| Redirect (regular) | `c.Redirect("/url")` |
| Redirect (htmx) | `c.WriteHeader(htmx.HxRedirect, "/url"); c.WriteStatus(200)` |
| Set response header | `c.WriteHeader(name, value)` |
| Set status only | `c.WriteStatus(code)` |
| Live-reload templates (dev) | `xun.WithWatch()` |
| Custom fs.FS | `xun.WithFsys(fsys)` |
| Content negotiation | `xun.WithHandlerViewers(&xun.JsonViewer{})` |
| Cross-cutting wrappers | `xun.WithInterceptor(...)` |
| Cache-bust assets | `xun.WithBuildAssetURL(...)` + `{{asset "..."}}` |
| **Escape hatch for raw handlers** | `app.Mux().HandleFunc(pattern, h)` — **bypasses all middleware** |
| Hijack conn (WebSocket / SSE) | `app.Mux().HandleFunc(...)` + `w.(http.Hijacker).Hijack()` |
| Render page with path param | Place `app/pages/<dir>/{id}.html`; override handler with `app.Get("/dir/{id}", h)` |
| Page-level main business data | `c.View(structOrSlice, "viewer")` → `{{.Data.X}}` in template |
| Cross-page auxiliary value | `c.Set("k", v)` in middleware / handler → `{{.TempData.k}}` in template |
| Read path param (Go 1.22) | `c.Request.PathValue("id")` (note: `{id}`, not `:id`) |

---

## 12. How to Run (for verification only)

```bash
make install        # downloads tailwindcss + esbuild binaries (no npm needed)
make dev            # watches CSS, runs `go run ./cmd/app` (requires .env)

make build          # compiles UI + `go build -o bin/app ./cmd/app`
make build-dist     # Docker cross-compile → ./dist/

cp .env.example .env
go run ./cmd/app    # migrate-on-startup; APP_ADDR_HTTP=:8080 to override port
```

Demo login (after migrations): `demo@example.com` / `demo123`.

---

## 13. License

MIT