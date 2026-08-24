<!--layout:base-->
{{define "content"}}
<article class="prose-body">
  {{block "components/breadcrumb" .}}{{end}}

  <header class="mb-10 pb-10 border-b border-dark-300">
    <div class="flex flex-wrap items-center gap-3 text-xs text-gray-500 mb-2">
      <span class="px-2 py-1 bg-dark-200 rounded text-tech-cyan uppercase tracking-wider font-medium">
        {{formatBlogDate .Content.Date}}
      </span>
      <span class="font-mono text-gray-600">/{{.Content.Slug}}</span>
      {{if .Content.Description}}
      <span class="w-1 h-1 rounded-full bg-gray-600"></span>
      <span class="text-gray-400">{{.Content.Description}}</span>
      {{end}}
    </div>
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