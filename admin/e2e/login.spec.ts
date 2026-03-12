import { expect, test } from './fixtures';

test.use({ storageState: { cookies: [], origins: [] } });

test.describe('Login page', () => {
  test.beforeEach(async ({ adminApi }) => {
    await adminApi.install();
  });

  test('valid credentials succeed', async ({ loginPage, page, toasts }) => {
    await loginPage.goto();
    await loginPage.login('admin@example.com', 'password123');

    await expect(page).toHaveURL('/');
    await expect(page.getByTestId('dashboard-page')).toBeVisible();
    await toasts.expectSuccess('Login successful');
  });

  test('invalid credentials show an error', async ({ loginPage, page, toasts }) => {
    await loginPage.goto();
    await loginPage.login('wrong@example.com', 'wrongpassword');

    await expect(page).toHaveURL('/login');
    await toasts.expectError('Invalid email or password');
  });

  test('session persists on refresh', async ({ loginPage, page }) => {
    await loginPage.goto();
    await loginPage.login('admin@example.com', 'password123');

    await expect(page).toHaveURL('/');
    await page.reload();

    await expect(page).toHaveURL('/');
    await expect(page.getByTestId('dashboard-page')).toBeVisible();
    await expect(page.getByText('admin@example.com')).toBeVisible();
  });
});
