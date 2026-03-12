import { expect, test } from './fixtures';

test.describe('POI map page', () => {
  test('renders markers, opens popup details, and navigates to the POI detail page', async ({
    adminApi,
    page,
  }) => {
    await adminApi.install();
    await page.goto('/poi-map');

    await expect(page.getByTestId('poi-map-page')).toBeVisible();
    await expect(page.getByTestId('poi-count')).toContainText('2 POIs');
    await expect(page.locator('.leaflet-container')).toBeVisible();
    await expect(page.locator('.leaflet-tile-loaded').first()).toBeVisible();

    const markers = page.locator('.leaflet-marker-icon');
    await expect(markers).toHaveCount(2);

    await markers.first().click();
    await expect(page.getByTestId('poi-popup-10')).toBeVisible();
    await expect(page.getByTestId('poi-popup-10')).toContainText('Narikala Fortress');
    await page.getByTestId('poi-detail-link-10').click();

    await expect(page).toHaveURL(/\/pois\/10$/);
    await expect(page.getByRole('button', { name: 'Back' })).toBeVisible();
    await expect(page.getByText('Narikala Fortress').first()).toBeVisible();
  });
});
