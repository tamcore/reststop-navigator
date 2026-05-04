import { expect, test } from '@playwright/test';

// Coordinate sits on the synthetic east-bound A8 carriageway at 48.000N 11.003E
// (see cmd/stubbedoverpass/fixtures/DE.json). Heading 90° = due east.
const FIXTURE_LAT = 48.0;
const FIXTURE_LON = 11.003;

test.describe('Upcoming stops home page', () => {
	test('renders A8 road label and at least one stop card with mocked geolocation', async ({
		page,
		context
	}) => {
		await context.grantPermissions(['geolocation'], {
			origin: process.env.E2E_BASE_URL ?? 'http://localhost:8080'
		});
		await context.setGeolocation({ latitude: FIXTURE_LAT, longitude: FIXTURE_LON });

		await page.goto('/');

		// Backend may need a beat to hydrate from the stubbed overpass.
		await expect(page.getByText(/A8/)).toBeVisible({ timeout: 20_000 });

		const cards = page.locator('a.card, .card');
		await expect(cards.first()).toBeVisible({ timeout: 20_000 });
	});

	test('toggling a fuel filter survives a reload via localStorage', async ({ page, context }) => {
		await context.grantPermissions(['geolocation'], {
			origin: process.env.E2E_BASE_URL ?? 'http://localhost:8080'
		});
		await context.setGeolocation({ latitude: FIXTURE_LAT, longitude: FIXTURE_LON });

		await page.goto('/');
		const fuelChip = page.getByRole('button', { name: 'Fuel' });
		await fuelChip.click();
		await expect(fuelChip).toHaveAttribute('aria-pressed', 'true');

		await page.reload();
		await expect(page.getByRole('button', { name: 'Fuel' })).toHaveAttribute('aria-pressed', 'true');
	});
});
