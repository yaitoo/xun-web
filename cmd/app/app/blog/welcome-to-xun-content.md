# Welcome to Xun Content

> Markdown that turns into routes — no build step, no template engine to wire up.

This post exists as `app/blog/welcome-to-xun-content.md`. Drop a `.md` file
into the content directory and xun does the rest: it parses the body,
extracts a title from the first `# H1`, grabs a description from the
first blockquote or paragraph, and registers a route at the matching
slug.

## How the engine works

The renderer walks the goldmark AST once and produces a `ContentView`:

- **Title** — text of the first `# H1`.
- **Description** — first blockquote (preferred) or top-level paragraph.
- **Date** — the file's modification time.
- **Body** — the rendered HTML, already escaped so `html/template` does
  not re-escape it.

The fields are exposed to the wrapping template as `.Content` — so a
single shared layout can render every post on the site.

## Why no frontmatter?

Frontmatter is friction. If you want a different title for one post,
just change the first heading. If you want metadata that the markdown
syntax can't express, plug a custom `WithContentMeta` extractor into
xun and pull it from anywhere — a sidecar file, a database, a CMS.

## Try it yourself

Edit this file and refresh the page. xun's watch mode picks up the
change, re-renders the body, and updates the route in place.