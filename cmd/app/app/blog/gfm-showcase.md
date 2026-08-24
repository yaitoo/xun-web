# GFM Showcase

> GitHub-Flavored Markdown is on by default — drop the syntax in, xun handles the rest.

xun's renderer (`github.com/yuin/goldmark` with the `extension.GFM` bundle)
ships with everything GitHub-flavored markdown supports out of the box.
No flags, no opt-in. The post you're reading uses several GFM-only
features that plain CommonMark would reject or render incorrectly.

## Tables

| Feature         | Syntax                       | Renders |
|-----------------|------------------------------|---------|
| Tables          | `\| col \| col \|`           | ✅       |
| Strikethrough   | `~~text~~`                   | ✅       |
| Task lists      | `- [ ] todo`                 | ✅       |
| Autolinks       | bare `https://...`           | ✅       |
| Fenced code     | ` ```go `                    | ✅       |
| ~~Old syntax~~  | ~~deprecated~~               | ✅       |

## Task lists

- [x] Drop a `.md` file in `app/blog/`
- [x] Get a route, rendered HTML, and a slug — free
- [ ] Decide on a custom template (per-post `*.html` works too)
- [ ] Set up an RSS feed (out of scope here, but trivial)

## Strikethrough and emphasis

Combine them: this is **bold**, this is *italic*, this is
~~struck-through~~, and this is `inline code`. A paragraph may also
mention https://github.com/yaitoo/xun directly — bare URLs become
links automatically thanks to the autolink extension.

## Fenced code with language hints

```go
// app/blog/<slug>.md becomes a route, no work required.
posts := []*xun.ContentView{ /* ... */ }
for _, p := range posts {
    fmt.Printf("%s -> %s\n", p.Slug, p.Title)
}
```

```ts
// TailwindCSS picks up class names from the prose namespace,
// so editing tailwind.css is all you need to retune the look.
export const prose = "prose-content";
```

## Inline code with backticks

Use `xun.WithContent("blog")` to enable the engine. Use
`WithContentRenderer` to plug in a fully-configured goldmark if you
need syntax highlighting or AST-level class injection.

## What GFM doesn't give you

- Footnotes — require `extension.Footnote`
- Definition lists — require `extension.DefinitionList`
- `<mark>`, `<sub>`, `<sup>` — goldmark's `extension.GFM` does include
  these as of goldmark 1.6+

Add them via `xun.WithContentRenderer` and a custom goldmark config
when you need them.

---

GFM is on by default. To turn it off (back to strict CommonMark),
pass a custom renderer via `xun.WithContentRenderer`.