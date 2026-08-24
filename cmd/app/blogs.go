package main

import (
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"
)

// BlogPost is the summary view of a blog post derived from a .md file in
// app/blog/. It carries only the fields the listing page needs; the full
// ContentView (with rendered Body HTML) is constructed by xun at load time
// and exposed to per-post templates via .Content.
type BlogPost struct {
	Slug        string    // URL slug, e.g. "welcome-to-xun-content" or "2026/nested"
	Path        string    // fs-relative path, e.g. "blog/welcome-to-xun-content.md"
	Title       string    // First # H1 from the markdown body
	Description string    // First blockquote or top-level paragraph
	Date        time.Time // File mtime in UTC
}

var (
	blogH1RE         = regexp.MustCompile(`(?m)^#\s+(.+?)\s*$`)
	blogBlockquoteRE = regexp.MustCompile(`(?ms)^>\s*(.+?)\s*$`)
	blogParaRE       = regexp.MustCompile(`(?m)^[^\s#\*\->\d].+?$`)
)

// extractBlogMeta pulls title and description out of the raw markdown bytes
// using the same conventions xun uses internally (first H1, first blockquote
// or top-level paragraph). It is intentionally cheap — the listing page
// walks the whole tree on every request in dev, so we avoid parsing goldmark
// ASTs here.
func extractBlogMeta(content []byte) (title, description string) {
	if m := blogH1RE.FindSubmatch(content); m != nil {
		title = strings.TrimSpace(string(m[1]))
	}

	// Prefer a leading blockquote; fall back to the first non-blank,
	// non-heading paragraph.
	if m := blogBlockquoteRE.FindSubmatch(content); m != nil {
		description = strings.TrimSpace(string(m[1]))
	} else if m := blogParaRE.FindSubmatch(content); m != nil {
		description = strings.TrimSpace(string(m[1]))
	}

	return title, description
}

// listBlogPosts walks the configured blog directory and returns one
// BlogPost per .md file, ordered by mtime descending (newest first).
//
// The function is request-time and re-reads the filesystem on every call;
// in production the posts will rarely change between deploys, and the dev
// loop benefits from always seeing the freshest file order.
func listBlogPosts(fsys fs.FS, dir string) ([]BlogPost, error) {
	var out []BlogPost

	err := fs.WalkDir(fsys, dir, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.EqualFold(path.Ext(p), ".md") {
			return nil
		}

		buf, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		fi, err := d.Info()
		if err != nil {
			return err
		}

		// Slug: strip the content directory prefix and the .md extension,
		// matching xun's slug convention so listing links resolve to the
		// routes it auto-registered.
		slug := strings.TrimSuffix(p, ".md")
		if dir != "" {
			slug = strings.TrimPrefix(slug, dir+"/")
		}

		title, description := extractBlogMeta(buf)

		out = append(out, BlogPost{
			Slug:        slug,
			Path:        p,
			Title:       title,
			Description: description,
			Date:        fi.ModTime().UTC(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Date.Equal(out[j].Date) {
			return out[i].Slug < out[j].Slug
		}
		return out[i].Date.After(out[j].Date)
	})

	return out, nil
}

// trimParagraph trims a markdown paragraph to a short preview by cutting
// at the first sentence boundary (period/question/exclamation followed by
// whitespace) or at max runes — whichever comes first.
func trimParagraph(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	cut := max
	for i, r := range s[:max] {
		if r == '.' || r == '?' || r == '!' {
			if i+1 < len(s) && (s[i+1] == ' ' || s[i+1] == '\n') {
				cut = i + 1
				break
			}
		}
	}
	out := strings.TrimSpace(s[:cut])
	if cut < len(s) {
		out += "…"
	}
	return out
}

// formatBlogDate formats a date as "Jan 2, 2006" in UTC. Centralised here
// so both the listing page and post cards render dates identically.
func formatBlogDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("Jan 2, 2006")
}