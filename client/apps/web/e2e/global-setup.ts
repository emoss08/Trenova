import { chromium, expect, type Page } from "@playwright/test";
import fs from "node:fs";
import path from "node:path";
import { BASE_URL, CHROMIUM_PATH, EMAIL, PASSWORD, STORAGE_STATE } from "./env";

const ON_AUTH = /\/(login|auth)(\/|\?|$)/;

/**
 * Signs in once and saves the session for every test. A storage state passed
 * in E2E_STORAGE_STATE is used as it is.
 */
export default async function globalSetup() {
  if (process.env.E2E_STORAGE_STATE) {
    if (!fs.existsSync(STORAGE_STATE)) {
      throw new Error(`E2E_STORAGE_STATE points at ${STORAGE_STATE}, which does not exist`);
    }
    return;
  }
  fs.mkdirSync(path.dirname(STORAGE_STATE), { recursive: true });

  const browser = await chromium.launch({ executablePath: CHROMIUM_PATH });
  try {
    const context = await browser.newContext({ baseURL: BASE_URL });
    const page = await context.newPage();
    await page.goto("/desk");
    // The app decides on the client whether to send the visitor to sign in.
    const composer = page.getByRole("textbox", { name: /^Message / });
    const email = page.getByPlaceholder("name@work-email.com");
    await expect(composer.or(email).first()).toBeVisible({ timeout: 30_000 });
    if (await email.isVisible()) {
      await signIn(page);
    }
    await expect(composer).toBeVisible({ timeout: 30_000 });
    await context.storageState({ path: STORAGE_STATE });
  } finally {
    await browser.close();
  }
}

async function signIn(page: Page) {
  await page.getByPlaceholder("name@work-email.com").fill(EMAIL);
  await page.locator('input[type="password"]').fill(PASSWORD);
  await page.getByRole("button", { name: /^Sign in$/ }).click();

  // After the password come the organization and the roles to act with, when
  // there is more than one of either.
  const deadline = Date.now() + 45_000;
  while (ON_AUTH.test(new URL(page.url()).pathname)) {
    if (Date.now() > deadline) {
      throw new Error(`Still signing in at ${page.url()}; check E2E_EMAIL and E2E_PASSWORD`);
    }
    const organizations = page.getByRole("radiogroup", { name: "Organizations" });
    const roles = page.getByRole("group", { name: "Authorized roles" });
    const proceed = page.getByRole("button", { name: /^(Continue|Activate \d+ roles?)$/ });
    if (await organizations.isVisible().catch(() => false)) {
      if (!(await organizations.getByRole("radio", { checked: true }).count())) {
        await organizations.getByRole("radio").first().click();
      }
    } else if (await roles.isVisible().catch(() => false)) {
      if (!(await roles.getByRole("checkbox", { checked: true }).count())) {
        await roles.getByRole("checkbox").first().click();
      }
    }
    if (await proceed.isEnabled({ timeout: 500 }).catch(() => false)) {
      await proceed.click();
    }
    await page.waitForTimeout(750);
  }
}
