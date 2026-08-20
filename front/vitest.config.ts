import { defineConfig } from 'vitest/config'

// Юнит-тесты чистой логики (валидаторы). Компонентные тесты Nuxt — отдельно (@nuxt/test-utils).
export default defineConfig({
  test: {
    include: ['layers/**/*.test.ts', 'shared/**/*.test.ts'],
    environment: 'node',
  },
})
