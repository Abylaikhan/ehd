import { EhdPreset } from './app/theme'

// Слои из папки layers/ (base, auth, reporter) Nuxt 4 подключает автоматически.
export default defineNuxtConfig({
  compatibilityDate: '2026-08-20',
  devtools: { enabled: true },

  modules: ['@primevue/nuxt-module', '@pinia/nuxt'],

  css: ['primeicons/primeicons.css', '~/assets/css/main.css'],

  primevue: {
    options: {
      ripple: true,
      // Корпоративная светлая тема MVP; dark в MVP не используется
      theme: { preset: EhdPreset, options: { darkModeSelector: '.ehd-dark' } },
    },
  },

  runtimeConfig: {
    apiBase: 'http://localhost:8080',
    public: {},
  },

  nitro: {
    devProxy: {
      '/api': {
        target: (process.env.NUXT_API_BASE || 'http://localhost:8080') + '/api',
        changeOrigin: true,
      },
    },
  },
})
