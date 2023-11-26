/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ['./*/*.html', '*.html'],
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
