<!--layout:base-->
{{/*
  Blog post wrapper.

  Two defined blocks:

    head-extra — <meta> tags (Open Graph, Twitter Card, JSON-LD, etc.)
                 sourced from .Content.Params, the sibling .yaml file
                 xun's Content engine merges into ContentView.Params.
                 Empty when no .yaml sidecar exists.

    content   — the article body itself.
*/}}

{{define "head-extra"}}
  {{with .Content.Params}}
    {{with .og}}
      <meta property="og:type" content="{{.type}}">
      <meta property="og:title" content="{{$.Content.Title}}">
      <meta property="og:description" content="{{$.Content.Description}}">
      {{with .image}}<meta property="og:image" content="{{.}}">{{end}}
    {{end}}
    {{with .twitter}}
      <meta name="twitter:card" content="{{.card}}">
      {{with .site}}<meta name="twitter:site" content="{{.}}">{{end}}
      {{with .creator}}<meta name="twitter:creator" content="{{.}}">{{end}}
      <meta name="twitter:title" content="{{$.Content.Title}}">
      <meta name="twitter:description" content="{{$.Content.Description}}">
    {{end}}
  {{end}}
{{end}}

{{define "content"}}
<article class="prose-body">
  <header class="flex items-center justify-between gap-4 mb-10 pb-10 border-b border-dark-300">
    <div class="min-w-0 flex-1">
      {{block "components/breadcrumb" .}}{{end}}

      <div class="flex flex-wrap items-center gap-3 text-xs text-gray-500 mt-4">
        <span class="px-2 py-1 bg-dark-200 rounded text-tech-cyan uppercase tracking-wider font-medium">
          {{formatBlogDate .Content.Date}}
        </span>
        <span class="font-mono text-gray-600">/{{.Content.Slug}}</span>
        {{with .Content.Params}}
          {{with .article}}
            {{with .tags}}
              <span class="w-1 h-1 rounded-full bg-gray-600"></span>
              <span class="text-gray-400">{{joinSlice .}}</span>
            {{end}}
            {{with .reading_time}}
              <span class="w-1 h-1 rounded-full bg-gray-600"></span>
              <span class="text-gray-400">{{.}}</span>
            {{end}}
          {{end}}
        {{end}}
      </div>
    </div>

    <a href="/blogs" class="shrink-0 inline-flex items-center gap-2 text-sm text-gray-400 hover:text-tech-cyan transition-colors">
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
      </svg>
      Back to all posts
    </a>
  </header>

  <div class="prose-content">
    {{.Content.Body}}
  </div>

  <footer>
    <a href="/blogs" class="btn btn-secondary">Browse all posts →</a>
    <span class="text-xs text-gray-500 font-mono">{{.Content.Path}}</span>
  </footer>
</article>
{{end}}