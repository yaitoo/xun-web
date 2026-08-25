# Routing With Templates

> Bubble-up template lookup lets one shared wrapper render every post on the site.

When xun mounts `app/blog/welcome-to-xun-content.md` as `GET /blog/welcome-to-xun-content`,
it also needs to know *which* template should wrap the rendered HTML.
That's where the bubble-up rules come in.

## The lookup order

For a file at `blog/<dir>/<file>.md`, xun searches in this order:

1. `blog/<dir>/<file>.html` — a per-post override when a post needs a
   unique layout.
2. `blog/<dir>/index.html` — the section landing page; covers every
   post in that subdirectory.
3. The first ancestor `index.html` walking up toward the content root.
4. `blog/index.html` — the top-level wrapper for everything in `blog/`.
5. `index.html` at the FS root — the absolute fallback.

The first match wins, and the route key is the slug, not the template
path. Multiple `.md` files can share the same bubble-up target.

## What this looks like in code

Each post in this demo resolves to `app/blog/index.html` because no
per-post `*.html` files exist. The template uses `<!--layout:base-->`
to inherit the chrome (nav, footer) and overrides the `content` block
to render `.Content.Body`.

That single template now serves:

- `GET /blog/welcome-to-xun-content`
- `GET /blog/routing-with-templates`
- `GET /blog/2026/nested-posts`
- and any future post you drop into `app/blog/`.

Add a per-post `app/blog/<slug>.html` whenever one post needs its own
visual treatment — xun will prefer it without touching the rest.