import { expect, test } from './fixtures';

test.describe('Dashboard page', () => {
  test('loads stats cards and sidebar navigation', async ({ adminApi, page }) => {
    await adminApi.install();
    await page.goto('/');

    await expect(page.getByTestId('dashboard-page')).toBeVisible();
    await expect(page.getByTestId('stat-cities')).toContainText('5');
    await expect(page.getByTestId('stat-pois')).toContainText('120');
    await expect(page.getByTestId('stat-stories')).toContainText('340');
    await expect(page.getByTestId('stat-reports')).toContainText('8');
    await expect(page.getByTestId('cities-table')).toContainText('Tbilisi');
    await expect(page.getByTestId('sidebar-nav')).toBeVisible();
    await expect(page.getByTestId('nav-dashboard')).toBeVisible();
    await expect(page.getByTestId('nav-poi-map')).toBeVisible();
    await expect(page.getByTestId('nav-reports')).toBeVisible();
  });
});
