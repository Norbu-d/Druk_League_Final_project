// Browser end-to-end tests. Start the stack first:
//   docker compose -f deploy/docker-compose.yml up -d
// Then: cd tests/e2e && npm ci && npx playwright test
// Admin tests need DRUK_ADMIN_PASSWORD (and optionally DRUK_ADMIN_USER); they are skipped without it.
// Sign-up tests create a viewer account, so run them only against a test-only database.
const { defineConfig } = require('@playwright/test');

module.exports = defineConfig({
  testDir: './specs',
  timeout: 45_000,
  retries: 0,
  workers: 1,
  reporter: [['list'], ['html', { open: 'never', outputFolder: 'report' }]],
  use: {
    baseURL: process.env.DRUK_BASE_URL || 'http://127.0.0.1:4173',
    channel: process.env.PW_CHANNEL || 'chrome',
    headless: true,
    screenshot: 'only-on-failure',
    trace: 'retain-on-failure'
  },
  projects: [
    { name: 'desktop', use: { viewport: { width: 1440, height: 960 } } },
    { name: 'mobile', use: { viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true }, grep: /@mobile/ }
  ]
});
