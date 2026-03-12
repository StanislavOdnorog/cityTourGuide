import { expect, test } from './fixtures';
import { authFile } from './fixtures';

test('authenticate as admin', async ({ adminApi, loginPage, page }) => {
  await adminApi.install();
  await loginPage.goto();
  await loginPage.login('admin@example.com', 'password123');

  await expect(page).toHaveURL('/');
  await expect(page.getByTestId('dashboard-page')).toBeVisible();

  await page.context().storageState({ path: authFile });
});
