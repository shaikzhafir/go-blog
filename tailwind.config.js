/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ['./*/*.html', '*.html', './templates/*.html', './templates/notionBlocks/*.html'],
  theme: {
    extend: {
      fontFamily: {
        geist: ['geist', 'sans-serif'],
      },
      colors: {
        activity: {
          0: '#ebedf0',
          1: '#9be9a8',
          2: '#40c463',
          3: '#30a14e',
          4: '#216e39',
        }
      },
    },
  },

  plugins: [
    require('@tailwindcss/typography'),
  ],
}
