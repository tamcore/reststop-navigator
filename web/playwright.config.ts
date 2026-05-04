import { defineConfig, devices } from '@playwright/test';

const baseURL = process.env.E2E_BASE_URL ?? 'http://localhost:8080';

export default defineConfig({
	testDir: 'tests/e2e',
	timeout: 30_000,
	expect: { timeout: 5_000 },
	fullyParallel: false,
	retries: process.env.CI ? 2 : 0,
	reporter: process.env.CI ? [['github'], ['list']] : 'list',
	use: {
		baseURL,
		trace: 'on-first-retry',
		permissions: []
	},
	projects: [
		{
			name: 'chromium',
			use: { ...devices['Desktop Chrome'] }
		}
	]
});
