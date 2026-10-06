// Author: colin2wang (colin2wang@gmail.com)
// Date: 2026-10-06

/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,ts}'],
  theme: {
    extend: {
      colors: {
        brand: {
          DEFAULT: '#3DDC84',
          dark: '#2BB673',
          light: '#7BE3B0',
        },
        ink: {
          900: '#0F172A',
          800: '#16203A',
          700: '#1E293B',
          600: '#273449',
        },
        muted: '#94A3B8',
      },
      fontFamily: {
        sans: ['Roboto', 'Inter', 'system-ui', 'Microsoft YaHei', 'sans-serif'],
        mono: ['JetBrains Mono', 'Consolas', 'Menlo', 'monospace'],
      },
      boxShadow: {
        glass: '0 8px 32px rgba(15, 23, 42, 0.45)',
        glow: '0 0 0 1px rgba(61, 220, 132, 0.35), 0 0 24px rgba(61, 220, 132, 0.18)',
      },
      keyframes: {
        'fade-up': {
          '0%': { opacity: '0', transform: 'translateY(8px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        breathe: {
          '0%, 100%': { boxShadow: '0 0 0 0 rgba(61, 220, 132, 0.45)' },
          '50%': { boxShadow: '0 0 0 10px rgba(61, 220, 132, 0)' },
        },
      },
      animation: {
        'fade-up': 'fade-up 0.35s ease-out both',
        breathe: 'breathe 1.6s ease-in-out infinite',
      },
    },
  },
  plugins: [],
}
