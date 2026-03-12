import { expect, test } from './fixtures';

test.describe('Reports page', () => {
  test.beforeEach(async ({ adminApi }) => {
    await adminApi.install();
  });

  test('loads the reports list and filters by status', async ({ page }) => {
    const row = (id: number) => page.locator('tr').filter({ has: page.getByTestId(`report-row-${id}`) });

    await page.goto('/reports');

    await expect(page.getByTestId('reports-page')).toBeVisible();
    await expect(row(1)).toContainText('Narikala Fortress');
    await expect(row(2)).toContainText('Peace Bridge');
    await expect(row(3)).toContainText('inappropriate content');

    await page.getByTestId('reports-status-filter').click();
    const dropdown = page.locator('.ant-select-dropdown:visible');
    await dropdown.getByText('New', { exact: true }).click();

    await expect(row(1)).toBeVisible();
    await expect(row(2)).toBeVisible();
    await expect(row(3)).toHaveCount(0);
  });

  test('marks a report as handled', async ({ page, toasts }) => {
    const row = (id: number) => page.locator('tr').filter({ has: page.getByTestId(`report-row-${id}`) });

    await page.goto('/reports');

    await row(1).getByRole('button', { name: 'Resolve' }).click();

    await toasts.expectSuccess('Report resolved');
    await expect(row(1)).toContainText('resolved');
    await expect(row(1)).toContainText('Closed');
  });
});
