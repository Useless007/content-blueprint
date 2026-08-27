import {expect, test, type Page} from '@playwright/test'

type Scenario = 'available' | 'current' | 'error-then-current'

async function installUpdateMock(page: Page, scenario: Scenario) {
  await page.addInitScript(({mockScenario}) => {
    const current = {
      currentVersion: '0.3.0',
      latestVersion: '0.3.0',
      state: 'up_to_date',
    }
    const available = {
      currentVersion: '0.3.0',
      latestVersion: '0.4.1',
      state: 'update_available',
      releaseUrl: 'https://github.com/Useless007/content-blueprint/releases/tag/v0.4.1',
      publishedAt: '2026-08-28T07:30:00Z',
      releaseNotes: 'เพิ่ม Update Center ที่ตรวจ checksum ก่อนติดตั้ง\nปรับปรุงประสบการณ์บนหน้าจอขนาดเล็ก',
    }
    const state = {checks: 0, downloads: 0, launched: '', openedUrl: ''}
    let progressListener: ((payload: unknown) => void) | undefined

    Object.assign(window, {
      __updateMock: state,
      runtime: {
        BrowserOpenURL: (url: string) => { state.openedUrl = url },
        EventsOnMultiple: (name: string, listener: (payload: unknown) => void) => {
          if (name === 'app:update-progress') progressListener = listener
          return () => { progressListener = undefined }
        },
      },
      go: {
        main: {
          App: {
            CheckForUpdates: async () => {
              state.checks += 1
              if (mockScenario === 'error-then-current' && state.checks === 1) {
                throw new Error('เชื่อมต่อ GitHub ไม่สำเร็จ')
              }
              return mockScenario === 'available' ? available : current
            },
            DownloadUpdate: async (version: string) => {
              state.downloads += 1
              progressListener?.({version, downloadedBytes: 2_000_000, totalBytes: 8_000_000, percent: 25})
              await new Promise((resolve) => window.setTimeout(resolve, 25))
              progressListener?.({version, downloadedBytes: 8_000_000, totalBytes: 8_000_000, percent: 100})
              return {...available, state: 'ready'}
            },
            LaunchDownloadedUpdate: async (version: string) => {
              state.launched = version
            },
          },
        },
      },
    })
  }, {mockScenario: scenario})
}

test('shows release details, verifies the download, and requires install confirmation', async ({page}) => {
  await installUpdateMock(page, 'available')
  await page.goto('/tests/update-harness.html')

  const launcher = page.getByRole('button', {name: 'ตรวจอัปเดต'})
  await launcher.click()
  await expect(page.getByRole('dialog')).toBeVisible()
  await expect(page.getByText('v0.4.1', {exact: true})).toBeVisible()
  await expect(page.getByText('เพิ่ม Update Center ที่ตรวจ checksum ก่อนติดตั้ง')).toBeVisible()
  await expect(page.getByText(/28 ส.ค. 2569/)).toBeVisible()

  await page.getByRole('button', {name: /ดูบน GitHub/}).click()
  await expect.poll(() => page.evaluate(() => (window as Window & {__updateMock?: {openedUrl: string}}).__updateMock?.openedUrl)).toContain('/releases/tag/v0.4.1')

  await page.getByRole('button', {name: 'ดาวน์โหลดอัปเดต'}).click()
  await expect(page.getByText('SHA-256 verified')).toBeVisible()
  await expect(page.getByText(/ตรงกับ checksum/)).toBeVisible()

  await page.getByRole('button', {name: 'ติดตั้งอัปเดต'}).click()
  await expect(page.getByText(/Unknown publisher/)).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(page.getByText(/Unknown publisher/)).toBeHidden()
  await expect(page.getByRole('dialog')).toBeVisible()

  await page.getByRole('button', {name: 'ติดตั้งอัปเดต'}).click()
  await page.getByRole('button', {name: 'เปิดตัวติดตั้ง Windows'}).click()
  await expect.poll(() => page.evaluate(() => (window as Window & {__updateMock?: {launched: string}}).__updateMock?.launched)).toBe('0.4.1')
})

test('reports current version and restores focus after Escape', async ({page}) => {
  await installUpdateMock(page, 'current')
  await page.goto('/tests/update-harness.html')

  const launcher = page.getByRole('button', {name: 'ตรวจอัปเดต'})
  await launcher.focus()
  await launcher.click()
  await expect(page.getByText('เป็นเวอร์ชันล่าสุดแล้ว', {exact: true})).toBeVisible()
  await page.getByRole('button', {name: 'ปิดหน้าต่างอัปเดต'}).focus()
  await page.keyboard.press('Shift+Tab')
  await expect(page.getByRole('button', {name: 'ปิด', exact: true})).toBeFocused()
  await page.keyboard.press('Escape')
  await expect(page.getByRole('dialog')).toBeHidden()
  await expect(launcher).toBeFocused()
})

test('shows an actionable error and retries without reloading', async ({page}) => {
  await installUpdateMock(page, 'error-then-current')
  await page.goto('/tests/update-harness.html')

  await page.getByRole('button', {name: 'ตรวจอัปเดต'}).click()
  await expect(page.getByRole('alert')).toContainText('เชื่อมต่อ GitHub ไม่สำเร็จ')
  await page.getByRole('button', {name: 'ลองอีกครั้ง'}).click()
  await expect(page.getByText('เป็นเวอร์ชันล่าสุดแล้ว', {exact: true})).toBeVisible()
})

test('does not repeat an automatic check inside 24 hours', async ({page}) => {
  await installUpdateMock(page, 'current')
  await page.addInitScript(() => {
    window.localStorage.setItem('content-blueprint:update:last-automatic-check', String(Date.now()))
  })
  await page.goto('/tests/update-harness.html')
  await page.waitForTimeout(2400)
  await expect.poll(() => page.evaluate(() => (window as Window & {__updateMock?: {checks: number}}).__updateMock?.checks)).toBe(0)
})

test('does not race a manual check with the delayed startup check', async ({page}) => {
  await installUpdateMock(page, 'current')
  await page.goto('/tests/update-harness.html')
  await page.getByRole('button', {name: 'ตรวจอัปเดต'}).click()
  await expect.poll(() => page.evaluate(() => (window as Window & {__updateMock?: {checks: number}}).__updateMock?.checks)).toBe(1)
  await page.waitForTimeout(2_400)
  await expect.poll(() => page.evaluate(() => (window as Window & {__updateMock?: {checks: number}}).__updateMock?.checks)).toBe(1)
})

test('does not retry a failed automatic check on every reload', async ({page}) => {
  await installUpdateMock(page, 'error-then-current')
  await page.goto('/tests/update-harness.html')
  await expect.poll(() => page.evaluate(() => (window as Window & {__updateMock?: {checks: number}}).__updateMock?.checks), {
    timeout: 4_000,
  }).toBe(1)
  await page.reload()
  await page.waitForTimeout(2_400)
  await expect.poll(() => page.evaluate(() => (window as Window & {__updateMock?: {checks: number}}).__updateMock?.checks)).toBe(0)
})

test('automatic check shows only a compact banner when an update exists', async ({page}) => {
  await installUpdateMock(page, 'available')
  await page.goto('/tests/update-harness.html')
  await expect(page.getByRole('complementary', {name: 'อัปเดต Content Blueprint'})).toBeVisible({timeout: 4_000})
  await expect(page.getByRole('dialog')).toHaveCount(0)
  await expect.poll(() => page.evaluate(() => (window as Window & {__updateMock?: {checks: number}}).__updateMock?.checks)).toBe(1)
})

test('keeps update actions separate from the app help control', async ({page}) => {
  await page.setViewportSize({width: 375, height: 667})
  await installUpdateMock(page, 'available')
  await page.goto('/tests/update-harness.html')
  await expect(page.getByRole('complementary', {name: 'อัปเดต Content Blueprint'})).toBeVisible({timeout: 4_000})

  const layout = await page.evaluate(() => {
    const selectors = ['.update-launcher', '.update-banner', '.onboarding-help-button']
    const rectangles = selectors.map((selector) => {
      const element = document.querySelector(selector)
      if (!(element instanceof HTMLElement)) throw new Error(`missing ${selector}`)
      const bounds = element.getBoundingClientRect()
      return {selector, left: bounds.left, right: bounds.right, top: bounds.top, bottom: bounds.bottom}
    })
    const collisions: string[] = []
    for (let leftIndex = 0; leftIndex < rectangles.length; leftIndex += 1) {
      for (let rightIndex = leftIndex + 1; rightIndex < rectangles.length; rightIndex += 1) {
        const left = rectangles[leftIndex]
        const right = rectangles[rightIndex]
        const overlaps = left.left < right.right && left.right > right.left
          && left.top < right.bottom && left.bottom > right.top
        if (overlaps) collisions.push(`${left.selector}:${right.selector}`)
      }
    }
    return {collisions, scrollWidth: document.documentElement.scrollWidth, viewportWidth: window.innerWidth}
  })
  expect(layout.collisions).toEqual([])
  expect(layout.scrollWidth).toBeLessThanOrEqual(layout.viewportWidth)
})

test('keeps the update flow usable at 375px and in landscape', async ({page}) => {
  await page.setViewportSize({width: 375, height: 667})
  await page.emulateMedia({reducedMotion: 'reduce'})
  await installUpdateMock(page, 'available')
  await page.goto('/tests/update-harness.html')
  await page.evaluate(() => { document.documentElement.style.fontSize = '20px' })
  await page.getByRole('button', {name: 'ตรวจอัปเดต'}).click()
  await expect(page.getByRole('dialog')).toBeVisible()
  await expect(page.getByRole('button', {name: 'ดาวน์โหลดอัปเดต'})).toBeVisible()
  await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
  await expect.poll(() => page.evaluate(() => window.matchMedia('(prefers-reduced-motion: reduce)').matches)).toBe(true)

  await page.setViewportSize({width: 667, height: 375})
  await expect(page.getByRole('dialog')).toBeVisible()
  await expect(page.getByRole('button', {name: 'ดาวน์โหลดอัปเดต'})).toBeVisible()
  await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
})
