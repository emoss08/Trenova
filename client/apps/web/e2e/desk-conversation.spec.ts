import { expect, test } from "@playwright/test";
import {
  DISPATCH_AGENT,
  question,
  savedReplies,
  sendButton,
  startConversation,
  stopButton,
  streamingReply,
  waitForTurnToEnd,
} from "./desk";

// The mock's e2e-basic and e2e-stop replies end on these.
const BASIC_LAST_SENTENCE = "Nothing else on the board needs you before noon.";
const STOP_FIRST_WORDS = "Here is the long version of";
const STOP_LAST_SENTENCE = "this last sentence should never arrive";

test.describe("Desk conversation", () => {
  test("a question from the home composer opens a conversation and streams its reply", async ({
    page,
  }) => {
    const asked = question("e2e-basic", "How is the day looking?");
    await startConversation(page, DISPATCH_AGENT, asked);

    // While it is written, the composer offers Stop and the words arrive
    // before the whole reply has.
    await expect(stopButton(page)).toBeVisible({ timeout: 30_000 });
    await expect(streamingReply(page)).toBeVisible({ timeout: 30_000 });
    await expect(streamingReply(page)).not.toHaveText("");

    await waitForTurnToEnd(page);
    const reply = savedReplies(page).last();
    await expect(reply).toContainText("Four loads are still New");
    await expect(reply).toContainText(BASIC_LAST_SENTENCE);
    // The question is asked once, above the reply.
    await expect(page.locator(".dk-scroll .dk-q", { hasText: asked })).toHaveCount(1);
  });

  test("Stop ends a reply partway and says so", async ({ page }) => {
    await startConversation(
      page,
      DISPATCH_AGENT,
      question("e2e-stop", "Walk me through every lane"),
    );

    await expect(streamingReply(page)).toContainText(STOP_FIRST_WORDS, { timeout: 30_000 });
    await stopButton(page).click();

    await expect(page.getByText("Reply stopped", { exact: true }).first()).toBeVisible();
    await expect(page.getByText(/You stopped it after \d+ words?\./).first()).toBeVisible();
    await expect(stopButton(page)).toBeHidden();
    await expect(sendButton(page)).toBeVisible();
    // What arrived stays; what had not arrived never does.
    await expect(page.locator(".dk-scroll")).toContainText(STOP_FIRST_WORDS);
    await page.waitForTimeout(2_000);
    await expect(page.locator(".dk-scroll")).not.toContainText(STOP_LAST_SENTENCE);
  });
});
