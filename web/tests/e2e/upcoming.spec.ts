import { expect, test } from '@playwright/test';

// Coordinate sits on the synthetic east-bound A8 carriageway at 48.000N 11.003E
// (see cmd/stubbedoverpass/fixtures/DE.json). Heading 90° = due east, speed
// 33 m/s ~= 119 km/h. setGeolocation can't carry heading/speed, so we replace
// the entire navigator.geolocation surface via an init script.
const FAKE_POSITION = {
	latitude: 48.0,
	longitude: 11.003,
	heading: 90,
	speed: 33
};

test.beforeEach(async ({ context }) => {
	await context.addInitScript((p) => {
		const fire = (success: PositionCallback) =>
			success({
				coords: {
					latitude: p.latitude,
					longitude: p.longitude,
					heading: p.heading,
					speed: p.speed,
					accuracy: 5,
					altitude: null,
					altitudeAccuracy: null
				} as GeolocationCoordinates,
				timestamp: Date.now()
			} as GeolocationPosition);

		const fake: Geolocation = {
			watchPosition(success: PositionCallback) {
				setTimeout(() => fire(success), 50);
				return 1;
			},
			getCurrentPosition(success: PositionCallback) {
				fire(success);
			},
			clearWatch() {
				/* no-op */
			}
		};
		Object.defineProperty(navigator, 'geolocation', {
			value: fake,
			configurable: true,
			writable: true
		});
	}, FAKE_POSITION);
});

test.describe('Upcoming stops home page', () => {
	test('renders A8 road label and at least one stop card', async ({ page }) => {
		const upcomingResp = page.waitForResponse(
			(r) => r.url().includes('/api/stops/upcoming') && r.status() === 200,
			{ timeout: 25_000 }
		);
		await page.goto('/');
		await upcomingResp;

		await expect(page.getByText(/A8/)).toBeVisible({ timeout: 10_000 });

		const cards = page.locator('a.card');
		await expect(cards.first()).toBeVisible({ timeout: 10_000 });
	});

	test('toggling a fuel filter survives a reload via localStorage', async ({ page }) => {
		await page.goto('/');
		const fuelChip = page.getByRole('button', { name: 'Fuel' });
		await fuelChip.click();
		await expect(fuelChip).toHaveAttribute('aria-pressed', 'true');

		await page.reload();
		await expect(page.getByRole('button', { name: 'Fuel' })).toHaveAttribute('aria-pressed', 'true');
	});
});
