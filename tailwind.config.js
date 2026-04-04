/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,jsx,ts,tsx}'],
  theme: {
    extend: {
      keyframes: {
        // idle: gentle floating bob
        float: {
          '0%, 100%': { transform: 'translateY(0px)' },
          '50%':      { transform: 'translateY(-6px)' },
        },
        // hyped: hard pixel bounce (stepped for retro feel)
        bounce_pixel: {
          '0%':   { transform: 'translateY(0px)' },
          '20%':  { transform: 'translateY(-10px)' },
          '40%':  { transform: 'translateY(-4px)' },
          '60%':  { transform: 'translateY(-14px)' },
          '80%':  { transform: 'translateY(-2px)' },
          '100%': { transform: 'translateY(0px)' },
        },
        // exhausted: slow sway + slight droop
        droop: {
          '0%, 100%': { transform: 'rotate(0deg) translateY(0px)' },
          '30%':      { transform: 'rotate(-2deg) translateY(3px)' },
          '70%':      { transform: 'rotate(2deg) translateY(5px)' },
        },
      },
      animation: {
        float:         'float 2.8s ease-in-out infinite',
        bounce_pixel:  'bounce_pixel 0.45s steps(3, end) infinite',
        droop:         'droop 3.5s ease-in-out infinite',
      },
    },
  },
  plugins: [],
};
