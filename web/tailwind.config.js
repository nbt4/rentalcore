/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        'accent-red': '#D0021B',
        'dark': {
          DEFAULT: '#0B0B0B',
          100: '#111111',
          200: '#161616',
          300: '#1F1F1F',
        },
        'light': {
          DEFAULT: '#FFFFFF',
          100: '#F5F5F5',
          200: '#EAEAEA',
        }
      },
      backdropBlur: {
        xs: '2px',
      }
    },
  },
  plugins: [],
}
