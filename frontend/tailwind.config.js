/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{js,jsx}"],
  theme: { extend: {} },
  plugins: [],
  safelist: [
    { pattern: /bg-(cyan|violet|amber|rose|emerald|zinc)-(400|500|600|700|800|900)/ },
    { pattern: /text-(cyan|violet|amber|rose|emerald|zinc)-(400|500|600|700|800)/ },
    { pattern: /border-(cyan|violet|amber|rose|emerald|zinc)-(500|600|700|800)/ },
    { pattern: /from-(cyan|violet|amber|rose|emerald|zinc)-500\/10/ },
    { pattern: /border-(cyan|violet|amber|rose|emerald|zinc)-500\/20/ },
  ]
}
