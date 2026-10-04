import { expect, test, type Page } from "@playwright/test";
import { openDeskHome } from "./desk";

/** Takes focus off the composer, so a shortcut goes to the Desk rather than the text box. */
async function blur(page: Page) {
  await page.evaluate(() => (document.activeElement as HTMLElement | null)?.blur());
}

test.describe("Desk shell", () => {
  test("the search palette opens with ⌘K / Ctrl+K and closes with Escape", async ({ page }) => {
    await openDeskHome(page);
    await blur(page);
    const palette = page.getByRole("dialog", { name: "Search" });

    await page.keyboard.press("ControlOrMeta+k");
    await expect(palette).toBeVisible();
    const input = palette.getByRole("combobox", { name: "Search the Desk" });
    await expect(input).toBeFocused();
    await page.keyboard.type("board");
    await expect(input).toHaveValue("board");
    // Tab moves between what to search in, without leaving the palette.
    await page.keyboard.press("Tab");
    await expect(palette).toBeVisible();

    await page.keyboard.press("Escape");
    await expect(palette).toBeHidden();

    // And again: the shortcut still works once the palette has closed.
    await blur(page);
    await page.keyboard.press("ControlOrMeta+k");
    await expect(palette).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(palette).toBeHidden();
  });

  test("the settings dialog opens from the keyboard and closes with Escape", async ({ page }) => {
    await openDeskHome(page);
    const before = page.url();
    const trigger = page.getByRole("button", { name: "Desk settings" });
    await expect(trigger).toBeVisible();
    await trigger.focus();
    await page.keyboard.press("Enter");

    const dialog = page.getByRole("dialog", { name: "Desk settings" });
    await expect(dialog).toBeVisible();
    await expect(dialog).toBeFocused();
    await expect(dialog.getByRole("heading", { name: "Settings" })).toBeVisible();

    await page.keyboard.press("Escape");
    await expect(dialog).toBeHidden();
    // Escape closed the dialog and nothing else: the Desk stays where it was.
    expect(page.url()).toBe(before);
    await expect(page.getByRole("textbox", { name: /^Message / })).toBeVisible();
  });
});
