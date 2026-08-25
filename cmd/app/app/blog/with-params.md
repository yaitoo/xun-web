# Sharing Blog Posts with Twitter Card & Open Graph

> Sidecar `.yaml` next to a `.md` post gives templates a place to read per-post metadata from, without putting frontmatter in the body.

xun's Content engine walks each `.md` file in a `content/` directory, parses it with goldmark, and exposes the result as `ViewModel.Content`. The same engine also looks for a **sibling `.yaml` file with the same base name** — `with-params.yaml` next to `with-params.md` — and merges its top-level keys into `ContentView.Params` as `map[string]any`.

That gives templates a place to read per-post metadata from **without** touching the body. In particular, it lets the wrapper inject `<meta>` tags into the page `<head>` based on the post being rendered, not the layout.

## Why this matters

A markdown body has no good place for:

- `<meta property="og:image">` (Open Graph image)
- `<meta name="twitter:card">` (Twitter card variant)
- Custom `<link rel="alternate" type="application/rss+xml">` per post
- JSON-LD `Article` blobs for SEO

You don't want any of those in the rendered body — they're chrome, not content. Putting them in frontmatter works, but xun deliberately chose not to parse frontmatter. The sidecar `.yaml` is the lighter answer: a separate file that doesn't get rendered, with arbitrary keys the template author defines.

## What's in the sidecar

```yaml
og:
  type: "article"
  image: "https://example.com/static/blog/with-params.png"
twitter:
  card: "summary_large_image"
  site: "@yaitoo"
  creator: "@yaitoo"
```

The keys (`og`, `twitter`, `article`) are arbitrary — xun does not validate them. Templates read whatever shape they want. The wrapper for this blog uses `.Content.Params.og`, `.Content.Params.twitter`, and `.Content.Params.article.tags` to render the matching `<meta>` tags.

## Where to put the templates

The `<meta>` tags belong in `<head>`, not in `<body>`. The base layout owns `<head>`, so we add an opt-in `head-extra` block to the layout. Content pages that want to inject head tags define `{{define "head-extra"}}…{{end}}` inside their wrapper; pages that don't care leave it empty (and the layout's empty default renders nothing).

That's the same pattern xun uses for the `content` block. Reusing it for head tags keeps the layout open without making every page think about SEO.