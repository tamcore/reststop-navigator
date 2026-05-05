import { expect, test } from '@playwright/test';

// A3 east-bound Frankfurt → Hanau — same fixture as docs/screenshots.
const DEMO_LAT = 50.06;
const DEMO_LON = 8.87;

// Deny geolocation to simulate a visitor with no GPS.
test.use({ permissions: [] });

test.beforeEach(async ({ context }) => {
	// Block watchPosition so it never fires — simulates denied geolocation
	// without triggering a browser permission prompt.
	await context.addInitScript(() => {
		Object.defineProperty(navigator, 'geolocation', {
			value: {
				watchPosition(_success: PositionCallback, error?: PositionErrorCallback) {
					error?.({
						code: 1,
						message: 'User denied',
						PERMISSION_DENIED: 1,
						POSITION_UNAVAILABLE: 2,
						TIMEOUT: 3
					} as GeolocationPositionError);
					return 1;
				},
				getCurrentPosition(_success: PositionCallback, error?: PositionErrorCallback) {
					error?.({
						code: 1,
						message: 'User denied',
						PERMISSION_DENIED: 1,
						POSITION_UNAVAILABLE: 2,
						TIMEOUT: 3
					} as GeolocationPositionError);
				},
				clearWatch() {}
			} as Geolocation,
			configurable: true,
			writable: true
		});
	});
});

test('demo mode loads stops without real GPS', async ({ page }) => {
	await page.goto('/');

	// Without demo, the GPS-denied hero is shown and no stop cards exist.
	await expect(page.getByText(/Location denied/i)).toBeVisible();
	await expect(page.locator('a.card').first()).not.toBeVisible();

	// Click the footer "Demo mode" button.
	const upcomingResp = page.waitForResponse(
		(r) => r.url().includes('/api/stops/upcoming') && r.status() === 200,
		{ timeout: 20_000 }
	);
	await page.getByRole('button', { name: 'Demo mode' }).click();
	await upcomingResp;

	// The demo banner is visible.
	await expect(page.getByText('DEMO MODE')).toBeVisible();

	// The road shield shows A3.
	await expect(page.getByText(/A3/)).toBeVisible({ timeout: 10_000 });

	// At least one stop card is rendered.
	await expect(page.locator('a.card').first()).toBeVisible({ timeout: 10_000 });
});

test('demo mode persists across a hard reload', async ({ page }) => {
	await page.goto('/');
	const upcomingResp = page.waitForResponse(
		(r) => r.url().includes('/api/stops/upcoming') && r.status() === 200,
		{ timeout: 20_000 }
	);
	await page.getByRole('button', { name: 'Demo mode' }).click();
	await upcomingResp;
	await expect(page.locator('a.card').first()).toBeVisible({ timeout: 10_000 });

	// Hard reload — demo flag persists via localStorage.
	const reloadResp = page.waitForResponse(
		(r) => r.url().includes('/api/stops/upcoming') && r.status() === 200,
		{ timeout: 20_000 }
	);
	await page.reload();
	await reloadResp;
	await expect(page.locator('a.card').first()).toBeVisible({ timeout: 10_000 });
	await expect(page.getByText('DEMO MODE')).toBeVisible();
});

test('exit demo returns to GPS-denied state', async ({ page }) => {
	await page.goto('/');
	const upcomingResp = page.waitForResponse(
		(r) => r.url().includes('/api/stops/upcoming') && r.status() === 200,
		{ timeout: 20_000 }
	);
	await page.getByRole('button', { name: 'Demo mode' }).click();
	await upcomingResp;
	await expect(page.locator('a.card').first()).toBeVisible({ timeout: 10_000 });

	// Click "exit" in the demo banner.
	await page.getByRole('button', { name: 'exit' }).click();

	// GPS-denied hero reappears; banner gone; footer button reads "Demo mode" again.
	await expect(page.getByText(/Location denied/i)).toBeVisible({ timeout: 5_000 });
	await expect(page.getByText('DEMO MODE')).not.toBeVisible();
	await expect(page.getByRole('button', { name: 'Demo mode' })).toBeVisible();
});

test('detail page works in demo mode', async ({ page }) => {
	await page.goto('/');
	const upcomingResp = page.waitForResponse(
		(r) => r.url().includes('/api/stops/upcoming') && r.status() === 200,
		{ timeout: 20_000 }
	);
	await page.getByRole('button', { name: 'Demo mode' }).click();
	await upcomingResp;
	await expect(page.locator('a.card').first()).toBeVisible({ timeout: 10_000 });

	// Navigate to the first stop's detail page.
	await page.locator('a.card').first().click();
	await expect(page).toHaveURL(/\/stop\//);
	// lat/lon in the URL are the stop's coordinates, not the user's — no fixed value to assert.

	// Detail page loads and shows the map.
	await expect(page.locator('.map')).toBeVisible({ timeout: 15_000 });

	// Demo banner still visible on detail page.
	await expect(page.getByText('DEMO MODE')).toBeVisible();
});
