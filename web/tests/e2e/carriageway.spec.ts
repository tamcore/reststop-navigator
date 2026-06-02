import { expect, test } from '@playwright/test';

// Westbound A1 position east of the synthetic St. Pölten stops (AT.json).
// lat=48.185 = the west carriageway row, lon=15.79 = east of the stops at 15.72.
// heading=270 = due west, speed=30 m/s ≈ 108 km/h.
const WESTBOUND_A1 = {
	latitude: 48.185,
	longitude: 15.79,
	heading: 270,
	speed: 30
};

test.describe('Carriageway filter — westbound A1 Austria', () => {
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
				clearWatch() {}
			};
			Object.defineProperty(navigator, 'geolocation', {
				value: fake,
				configurable: true,
				writable: true
			});
		}, WESTBOUND_A1);
	});

	test('shows west-carriageway stop and hides east-carriageway stop', async ({ page }) => {
		const upcomingResp = page.waitForResponse(
			(r) => r.url().includes('/api/stops/upcoming') && r.status() === 200,
			{ timeout: 25_000 }
		);
		await page.goto('/');
		await upcomingResp;

		await expect(page.getByText(/A1/)).toBeVisible({ timeout: 10_000 });

		// West-carriageway stop must be present.
		await expect(page.getByText('Raststation St. Pölten West')).toBeVisible({ timeout: 10_000 });

		// East-carriageway stop must NOT appear in the list.
		await expect(page.getByText('Raststation St. Pölten Ost')).not.toBeVisible();
	});
});
