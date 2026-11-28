const { test, expect } = require('@playwright/test');

const user = process.env.DRUK_ADMIN_USER || 'admin';
const password = process.env.DRUK_ADMIN_PASSWORD;

test.describe('Super Admin', () => {
  test.skip(!password, 'set DRUK_ADMIN_PASSWORD to run the admin tests');

  test.beforeEach(async ({ page }) => {
    await page.goto('/#dashboard');
    await page.getByRole('button', { name: 'Sign in', exact: true }).first().click();
    await page.fill('#login-user', user);
    await page.fill('#login-password', password);
    await page.locator('#signin-form').getByRole('button', { name: 'Sign in' }).click();
    await expect(page.getByText(`Welcome back, ${user}`)).toBeVisible();
  });

  test('set-up checklist covers venues and approvals', async ({ page }) => {
    const checklist = page.locator('.checklist');
    for (const step of ['Create a tournament', 'Add venues', 'Approve at least two teams', 'Review team registrations', 'Generate fixtures']) {
      await expect(checklist).toContainText(step);
    }
  });

  test('registrations page shows the approval queue', async ({ page }) => {
    await page.goto('/#registrations');
    await expect(page.getByRole('heading', { name: 'Registrations', level: 1 })).toBeVisible();
    await page.getByRole('button', { name: /^All/ }).click();
    await expect(page.locator('.table tbody tr').first()).toBeVisible();
  });

  test('venues come from the venue service', async ({ page }) => {
    await page.goto('/#venues');
    await expect(page.getByRole('heading', { name: 'Venues', level: 1 })).toBeVisible();
    await expect(page.locator('.table tbody tr').first()).toBeVisible();
    await page.getByRole('button', { name: 'Add venue' }).first().click();
    await expect(page.getByRole('dialog', { name: 'Add a venue' })).toBeVisible();
  });

  test('admin can open any squad', async ({ page }) => {
    await page.goto('/#squads');
    await expect(page.locator('#squad-team')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Add player' }).first()).toBeVisible();
  });
});
