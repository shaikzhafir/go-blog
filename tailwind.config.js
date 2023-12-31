/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ['./*/*.html', '*.html', './templates/*.html', './templates/notionBlocks/*.html'],
  theme: {
    extend: {
      fontFamily: {
        geist: ['geist', 'sans-serif'],
      },
    },
  },

  plugins: [
    require('@tailwindcss/typography'),
  ],
}
