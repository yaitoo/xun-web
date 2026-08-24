# Nested Posts

> Subdirectories become URL prefixes — organise posts by year, topic, or author without writing any routing code.

This file lives at `app/blog/2026/nested-posts.md`. xun turns that into
`GET /blog/2026/nested-posts` automatically.

## Why use nested directories?

The bubble-up rules mean each subdirectory can carry its own landing
page. Drop an `index.html` next to your `.md` files and it becomes
both a section page (`GET /blog/2026/`) and the wrapping template for
every post in that folder.

Conventions you can adopt out of the box:

- **By year** — `2025/`, `2026/`, `2027/` for chronological archives.
- **By series** — `series/launch-week/` for multi-part deep dives.
- **By author** — `team/<name>/` when you want a per-author RSS feed.

## A small example

```
app/blog/
├── index.html              ← top-level wrapper
├── welcome-to-xun-content.md
├── routing-with-templates.md
└── 2026/
    ├── index.html          ← year landing + bubble-up for 2026/*.md
    └── nested-posts.md     ← this post
```

The listing page (`/blogs`) walks the whole tree, so nesting doesn't
change how posts show up in the index — only how they're organised on
disk.

## Customising a single section

Add `app/blog/2026/index.html` with a `<!--layout:base-->` directive and
a `content` block that reads `.Content` the same way the top-level
wrapper does. Posts in `2026/` will pick it up automatically; posts
elsewhere keep using the top-level template.