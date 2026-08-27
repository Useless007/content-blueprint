import {defineConfig, devices} from '@playwright/test'

export default defineConfig({
  testDir: './tests',
  testMatch: '**/*.spec.ts',
  timeout: 20_000,
  expect: {timeout: 5_000},
  fullyParallel: false,
  workers: 1,
  reporter: 'list',
  outputDir: './test-results/update-center',
  use: {
    ...devices['Desktop Chrome'],
    baseURL: 'http://127.0.0.1:4173',
    channel: 'chrome',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  webServer: {
    command: 'npm run dev -- --host 127.0.0.1 --port 4173 --strictPort',
    url: 'http://127.0.0.1:4173/tests/update-harness.html',
    reuseExistingServer: true,
    timeout: 30_000,
  },
})
