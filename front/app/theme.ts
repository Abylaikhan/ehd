import { definePreset } from '@primeuix/themes'
import Aura from '@primeuix/themes/aura'

// Корпоративный светлый preset ЕХД. Бренд — navy #0B2545 (primary.700).
// Статусные цвета не входят в тему (они зарезервированы) — см. tokens в main.css.
export const EhdPreset = definePreset(Aura, {
  semantic: {
    primary: {
      50: '#eef1f6',
      100: '#d3dbe8',
      200: '#a7b6d0',
      300: '#7b91b8',
      400: '#4f6ca0',
      500: '#274b78',
      600: '#163a63',
      700: '#0b2545',
      800: '#081d38',
      900: '#06152a',
      950: '#030c18',
    },
    borderRadius: { sm: '6px', md: '8px', lg: '12px', xl: '16px' },
    focusRing: { width: '2px', style: 'solid', color: '{primary.400}', offset: '2px' },
    colorScheme: {
      light: {
        primary: {
          color: '{primary.700}',
          contrastColor: '#ffffff',
          hoverColor: '{primary.800}',
          activeColor: '{primary.900}',
        },
        highlight: {
          background: '{primary.50}',
          focusBackground: '{primary.100}',
          color: '{primary.700}',
          focusColor: '{primary.800}',
        },
        surface: {
          0: '#ffffff',
          50: '#f7f8fa',
          100: '#eef1f5',
          200: '#e2e7ee',
          300: '#cdd5e0',
          400: '#a7b2c2',
          500: '#8592a6',
          600: '#647087',
          700: '#4b5568',
          800: '#333b49',
          900: '#1f2530',
          950: '#12161d',
        },
      },
    },
  },
})
