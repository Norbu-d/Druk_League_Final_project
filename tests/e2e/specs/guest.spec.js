const { test, expect } = require('@playwright/test');

test.describe('Visitor without an account', () => {
  test('landing page opens the public workspace @mobile', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('#landing')).toBeVisible();
    await page.locator('[data-lp-action="open"]').first().click();
    await expect(page).toHaveURL(/#dashboard$/);
    await expect(page.getByText('Browsing as a guest').first()).toBeVisible();
  });

  test('guest overview shows real public data', async ({ page }) => {
    await page.goto('/#dashboard');
    await expect(page.getByRole('heading', { name: /Follow every match/ })).toBeVisible();
    // Metrics come from /api/public/*; the stack holds saved tournaments.
    const tournaments = page.locator('.metric', { hasText: 'Tournaments' }).locator('b');
    await expect(tournaments).not.toHaveText('0');
    await expect(page.locator('.match-card').first()).toBeVisible();
  });

  test('public pages load without signing in and hide official accounts', async ({ page }) => {
    const requests = [];
    page.on('request', (r) => { if (r.url().includes('/api/')) requests.push(new URL(r.url()).pathname); });
    await page.goto('/#matches');
    await expect(page.locator('.match-card').first()).toBeVisible();
    await expect(page.locator('.match-card .official')).toHaveCount(0);
    expect(requests.some((p) => p.startsWith('/api/public/'))).toBe(true);
    expect(requests.filter((p) => ['/api/teams', '/api/matches', '/api/tournaments'].includes(p))).toEqual([]);
  });

  test('standings download comes from the standings-csv function', async ({ page, request }) => {
    await page.goto('/#standings');
    const link = page.getByRole('link', { name: 'Download CSV' });
    await expect(link).toHaveAttribute('href', /\/fn\/standings-csv\?tournament_id=[0-9a-f-]{36}$/);
    const response = await request.get(await link.getAttribute('href'));
    expect(response.status()).toBe(200);
    expect(response.headers()['content-type']).toContain('text/csv');
    expect((await response.text()).split('\n')[0]).toContain('Rank,Team,Dzongkhag');
  });

  test('squads ask the visitor to sign in or create an account', async ({ page }) => {
    await page.goto('/#squads');
    await expect(page.getByText('Sign in to see squads')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Create free account' })).toBeVisible();
  });
});
