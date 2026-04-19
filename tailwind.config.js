/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ['./templates/**/*.html', './*.html'],
  theme: {
    extend: {
      fontFamily: {
        display: ['"Bricolage Grotesque"', 'system-ui', 'sans-serif'],
        body: ['Literata', 'Georgia', 'serif'],
        sans: ['Literata', 'Georgia', 'serif'],
      },
      colors: {
        cream: {
          50: '#fdfcfa',
          100: '#faf7f2',
          200: '#f0ebe2',
          300: '#e2d9cc',
        },
        terra: {
          DEFAULT: '#c4654a',
          dark: '#a8513a',
          light: '#f4e6e0',
        },
        ink: {
          DEFAULT: '#2d2a24',
          light: '#6b5c50',
          muted: '#9c8e80',
        },
        activity: {
          0: '#f0ebe2',
          1: '#d4c4a8',
          2: '#c4654a',
          3: '#a8513a',
          4: '#7a3a28',
        }
      },
    },
  },

  plugins: [
    require('@tailwindcss/typography'),
  ],
}
