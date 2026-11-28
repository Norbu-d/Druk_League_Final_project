const { test, expect } = require('@playwright/test');

// Creates one viewer account per run (test-only database).
test('a visitor signs up and becomes a signed-in Viewer @mobile', async ({ page }) => {
  const username = `e2e.viewer.${Date.now().toString(36)}`;
  await page.goto('/#dashboard');
  await page.getByRole('button', { name: /Create (free )?account/ }).first().click();
  await expect(page.getByRole('heading', { name: /Join DrukLeague/ })).toBeVisible();
  await page.fill('#signup-user', username);
  await page.fill('#signup-password', 'e2e-password-123');
  await page.fill('#signup-confirm', 'not-the-same-1');
  await page.getByRole('button', { name: 'Create account', exact: true }).last().click();
  await expect(page.locator('.form-error')).toHaveText('The two passwords do not match.');
  await page.fill('#signup-confirm', 'e2e-password-123');
  await page.getByRole('button', { name: 'Create account', exact: true }).last().click();
  await expect(page.getByText(`Welcome, ${username}`)).toBeVisible();
  await expect(page.getByRole('heading', { name: /Follow the season/ })).toBeVisible();
  // A viewer may read squads but has no admin pages.
  await page.goto('/#squads');
  await expect(page.getByRole('heading', { name: 'Squads', level: 1 })).toBeVisible();
  await page.goto('/#registrations');
  await expect(page).toHaveURL(/#dashboard$/);
});
