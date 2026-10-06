import { expect, test } from "@playwright/test";
import {
  DISPATCH_AGENT,
  openWorkspace,
  question,
  savedReplies,
  seenText,
  startConversation,
  waitForTurnToEnd,
  watchForText,
  workspace,
} from "./desk";

// The mock's e2e-artifact rule looks up SEED-SHP-002 and then John Smith,
// and its reply names only John Smith.
const NAMED = "John Smith";
const UNNAMED = "SEED-SHP-002";

test.describe("Desk artifacts", () => {
  test("the artifact a reply names opens from its badge; a lookup it did not name never shows", async ({
    page,
  }) => {
    // Anywhere an artifact is drawn: the workspace pane, its cards, and the
    // badges in the reply's words.
    await watchForText(page, ".dk-sheet, .dk-ax-card, .dk-abadge", UNNAMED);

    await startConversation(
      page,
      DISPATCH_AGENT,
      question("e2e-artifact", "Pull up John Smith's driver profile"),
    );
    // Open the pane while the turn is still running, so a lookup that shows
    // before the reply says which it keeps would be drawn there.
    await openWorkspace(page);
    await waitForTurnToEnd(page);

    const reply = savedReplies(page).last();
    await expect(reply).toContainText("He is free for the next load");
    const badge = reply.locator(".dk-abadge");
    await expect(badge).toHaveCount(1);
    await expect(badge).toHaveAccessibleName(/^Open /);
    await badge.click();

    const pane = workspace(page);
    await expect(pane.locator(".dk-ax-card.dk-front")).toContainText(NAMED);

    // Read the record and give the pane a moment to settle before the last look.
    await page.waitForTimeout(1_500);
    await expect(pane).not.toContainText(UNNAMED);
    expect(await seenText(page), `${UNNAMED} was drawn as an artifact at some point`).toEqual([]);
  });
});
