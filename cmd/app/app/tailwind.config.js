/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./cmd/app/app/**/*.{html,js}",
  ],
  theme: {
    extend: {
      colors: {
        dark: {
          50: '#0f1117',
          100: '#161922',
          200: '#1e222d',
          300: '#282d3a',
          400: '#353c4b',
          500: '#454e60',
        },
        tech: {
          cyan: '#00d9ff',
          purple: '#7c3aed',
          green: '#10b981',
          amber: '#f59e0b',
        }
      },
      fontFamily: {
        mono: ['JetBrains Mono', 'Fira Code', 'monospace'],
        sans: ['Inter', 'system-ui', 'sans-serif'],
      },
    },
  },

  // ----------------------------------------------------------------------------
  // Blog prose plugin
  //
  // The wrapper div in app/blog/index.html uses `prose-content` to wrap the
  // markdown body produced by xun's Content feature. xun emits raw goldmark
  // HTML — `<h1>`, `<p>`, `<ul>`, `<blockquote>`, `<code>`, `<pre>`, etc. —
  // and Tailwind's purge only sees literal class names, never those tags, so
  // we have to declare the styles explicitly via addComponents.
  //
  // We define the plugin as a plain function (no `tailwindcss/plugin`
  // wrapper, no `node_modules/` dependency) so the standalone tailwindcss
  // CLI can evaluate it in its embedded JS runtime without any Node toolchain
  // on the host. The `theme()` helper lets us reuse our `dark-*` / `tech-*`
  // colour tokens so palette changes propagate in one place.
  // ----------------------------------------------------------------------------
  plugins: [
    function ({ addComponents, theme }) {
      const c = theme('colors');

      addComponents({
        '.prose-content': {
          color: c.gray[300],
          lineHeight: '1.75',
        },
        '.prose-content > * + *': {
          marginTop: '1.5rem',
        },
        '.prose-content h1': {
          color: c.white,
          fontWeight: '700',
          fontSize: '2.25rem',
          lineHeight: '1.2',
          marginTop: '2.5rem',
          marginBottom: '0.5rem',
        },
        '.prose-content h2': {
          color: c.white,
          fontWeight: '600',
          fontSize: '1.75rem',
          lineHeight: '1.3',
          marginTop: '2.5rem',
          marginBottom: '0.75rem',
          paddingBottom: '0.5rem',
          borderBottomWidth: '1px',
          borderColor: c.dark[300],
        },
        '.prose-content h3': {
          color: c.white,
          fontWeight: '600',
          fontSize: '1.25rem',
          lineHeight: '1.4',
          marginTop: '2rem',
          marginBottom: '0.5rem',
        },
        '.prose-content h4': {
          color: c.white,
          fontWeight: '600',
          fontSize: '1.125rem',
          marginTop: '1.5rem',
          marginBottom: '0.5rem',
        },
        '.prose-content p': {
          color: c.gray[300],
        },
        '.prose-content a': {
          color: c.tech.cyan,
          textDecoration: 'underline',
          textUnderlineOffset: '4px',
          textDecorationColor: `${c.tech.cyan}66`, // ~40% alpha
        },
        '.prose-content a:hover': {
          textDecorationColor: c.tech.cyan,
        },
        '.prose-content strong': {
          color: c.white,
          fontWeight: '600',
        },
        '.prose-content em': {
          fontStyle: 'italic',
          color: c.gray[200],
        },
        '.prose-content ul': {
          listStyleType: 'disc',
          paddingLeft: '1.5rem',
        },
        '.prose-content ol': {
          listStyleType: 'decimal',
          paddingLeft: '1.5rem',
        },
        '.prose-content li': {
          color: c.gray[300],
          marginTop: '0.5rem',
        },
        '.prose-content li::marker': {
          color: c.tech.cyan,
        },
        '.prose-content blockquote': {
          borderLeftWidth: '4px',
          borderLeftColor: c.tech.purple,
          paddingLeft: '1rem',
          paddingTop: '0.25rem',
          paddingBottom: '0.25rem',
          fontStyle: 'italic',
          color: c.gray[400],
          backgroundColor: `${c.dark[100]}66`, // ~40% alpha
          borderTopRightRadius: '0.375rem',
          borderBottomRightRadius: '0.375rem',
        },
        '.prose-content blockquote p': {
          color: c.gray[400],
        },
        '.prose-content code': {
          fontFamily: 'JetBrains Mono, Fira Code, monospace',
          fontSize: '0.875em',
          backgroundColor: c.dark[200],
          color: c.tech.cyan,
          paddingLeft: '0.375rem',
          paddingRight: '0.375rem',
          paddingTop: '0.125rem',
          paddingBottom: '0.125rem',
          borderRadius: '0.25rem',
        },
        '.prose-content pre': {
          backgroundColor: c.dark[200],
          padding: '1rem',
          borderRadius: '0.5rem',
          overflowX: 'auto',
          borderWidth: '1px',
          borderColor: c.dark[300],
          marginTop: '1.5rem',
          marginBottom: '1.5rem',
        },
        '.prose-content pre code': {
          backgroundColor: 'transparent',
          padding: 0,
          color: c.gray[300],
          fontSize: '0.875rem',
        },
        '.prose-content hr': {
          borderColor: c.dark[300],
          marginTop: '2rem',
          marginBottom: '2rem',
        },
        '.prose-content table': {
          width: '100%',
          fontSize: '0.875rem',
          borderCollapse: 'collapse',
          marginTop: '1.5rem',
          marginBottom: '1.5rem',
        },
        '.prose-content th': {
          backgroundColor: c.dark[200],
          paddingLeft: '0.75rem',
          paddingRight: '0.75rem',
          paddingTop: '0.5rem',
          paddingBottom: '0.5rem',
          textAlign: 'left',
          fontWeight: '500',
          color: c.gray[400],
          borderWidth: '1px',
          borderColor: c.dark[300],
        },
        '.prose-content td': {
          paddingLeft: '0.75rem',
          paddingRight: '0.75rem',
          paddingTop: '0.5rem',
          paddingBottom: '0.5rem',
          borderWidth: '1px',
          borderColor: c.dark[300],
          color: c.gray[300],
        },
        // GFM task lists render as <input type="checkbox" disabled>. Style
        // them so the box reads on the dark surface and checked items get
        // a tech-cyan accent that matches the brand.
        '.prose-content input[type="checkbox"]': {
          appearance: 'none',
          width: '1rem',
          height: '1rem',
          borderRadius: '0.25rem',
          borderWidth: '2px',
          borderColor: c.dark[400],
          backgroundColor: c.dark[200],
          marginRight: '0.5rem',
          verticalAlign: 'middle',
          position: 'relative',
          top: '-1px',
          cursor: 'default',
        },
        '.prose-content input[type="checkbox"]:checked': {
          backgroundColor: c.tech.cyan,
          borderColor: c.tech.cyan,
        },
        '.prose-content input[type="checkbox"]:checked::after': {
          content: '""',
          position: 'absolute',
          left: '3px',
          top: '0px',
          width: '4px',
          height: '8px',
          borderRightWidth: '2px',
          borderBottomWidth: '2px',
          borderColor: c.dark[50],
          transform: 'rotate(45deg)',
        },
        '.prose-content img': {
          borderRadius: '0.5rem',
          marginTop: '1.5rem',
          marginBottom: '1.5rem',
          borderWidth: '1px',
          borderColor: c.dark[300],
        },
        '.prose-content kbd': {
          fontFamily: 'JetBrains Mono, Fira Code, monospace',
          fontSize: '0.75rem',
          backgroundColor: c.dark[200],
          color: c.gray[200],
          paddingLeft: '0.375rem',
          paddingRight: '0.375rem',
          paddingTop: '0.125rem',
          paddingBottom: '0.125rem',
          borderRadius: '0.25rem',
          borderWidth: '1px',
          borderColor: c.dark[400],
        },
      });
    },
  ],
}