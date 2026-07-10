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
  plugins: [],
}
