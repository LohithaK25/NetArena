
import { expect, test } from '@playwright/test'

test('submits a scenario to the Go API and displays simulation results', async ({ page }) => {
  const responsePromise = page.waitForResponse(
    (response) =>
      response.url().includes('/api/v1/simulations') &&
      response.request().method() === 'POST',
  )

  await page.goto('/')
  await page.getByRole('button', { name: /Run simulation/i }).click()

  const response = await responsePromise
  expect(response.status()).toBe(200)

  await expect(page.getByText('COMPLETED', { exact: true })).toBeVisible()
  await expect(page.getByText('api-smoke-test')).toBeVisible()

  await expect(page.getByText('Total requests').locator('..')).toContainText('2')
  await expect(page.getByText('Succeeded').locator('..')).toContainText('2')
  await expect(page.getByText('Failed').locator('..')).toContainText('0')
  await expect(page.getByText('Cancelled').locator('..')).toContainText('0')

  await expect(page.getByText('7.00')).toBeVisible()
  await expect(page.getByText('5.00')).toBeVisible()
  await expect(page.getByRole('progressbar', { name: /Worker utilization/i })).toHaveAttribute(
    'aria-valuenow',
    '0',
  )
})
