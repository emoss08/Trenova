import { expect, test } from "@playwright/test";
import {
  approvalCard,
  approveAndSettle,
  BILLING_AGENT,
  DISPATCH_AGENT,
  question,
  savedReplies,
  startConversation,
  waitForTurnToEnd,
  workspace,
} from "./desk";
import { COMMIT_APPROVALS } from "./env";

// What the mock's e2e-email rule drafts.
const DRAFT_SUBJECT = "Missing POD for SEED-SHP-001";

test.describe("Desk approvals", () => {
  test("a proposed change waits on a decision card and is approved from it", async ({ page }) => {
    await startConversation(
      page,
      DISPATCH_AGENT,
      question("e2e-approval", "SEED-SHP-002 just left the dock, mark it in transit"),
    );
    await waitForTurnToEnd(page);
    await expect(savedReplies(page).last()).toContainText("It waits on your approval.");

    const card = approvalCard(page);
    await expect(card).toBeVisible({ timeout: 30_000 });
    await expect(
      card,
      "The preview refused the change; the seeded move has probably moved on. Reseed, or point the e2e-approval rule at a move that is still New.",
    ).toHaveAttribute("aria-label", "Waiting on your approval");
    await expect(card.getByRole("button", { name: "Review" })).toBeVisible();
    await expect(card.getByRole("button", { name: "Not now" })).toBeVisible();

    const outcome = await approveAndSettle(page, card, COMMIT_APPROVALS);
    if (outcome === "undone") {
      // Taken back inside its window, the change waits on the person again:
      // on the card, or behind the pill that brings the card back.
      await expect(
        approvalCard(page)
          .or(page.getByRole("button", { name: /waits? on you$/ }))
          .first(),
      ).toBeVisible();
    } else {
      await expect(approvalCard(page)).toBeHidden();
    }
  });

  test("an email draft is edited, sent for approval and approved as modified", async ({ page }) => {
    await startConversation(
      page,
      BILLING_AGENT,
      question("e2e-email", "Ask Acme for the missing POD on SEED-SHP-001"),
    );
    await waitForTurnToEnd(page);
    await expect(savedReplies(page).last()).toContainText("Read it over before it goes.");

    const card = approvalCard(page);
    await expect(card).toBeVisible({ timeout: 30_000 });
    await expect(
      card,
      "The draft's preview was refused; check the mock found an email profile (emlprof_) and the sender can send.",
    ).toHaveAttribute("aria-label", "Waiting on your approval");

    // Review opens the draft in the workspace.
    await card.getByRole("button", { name: "Review" }).click();
    const pane = workspace(page);
    const subject = pane.getByRole("textbox", { name: "Subject" });
    const shown = await subject
      .waitFor({ state: "visible", timeout: 5_000 })
      .then(() => true)
      .catch(() => false);
    if (!shown) {
      // Another artifact is in front: fan the stack and pick the draft.
      await pane.locator(".dk-ax-card.dk-front").click();
      await pane.locator(".dk-ax-card", { hasText: DRAFT_SUBJECT }).click();
    }
    await expect(subject).toHaveValue(DRAFT_SUBJECT);
    await expect(subject).toBeEditable();

    const editedSubject = `${DRAFT_SUBJECT} (second request)`;
    const message = pane.getByRole("textbox", { name: "Message", exact: true });
    const originalBody = await message.inputValue();
    const editedBody = `${originalBody}\n\nThank you, the billing team`;
    await subject.fill(editedSubject);
    await message.fill(editedBody);
    await pane.getByRole("button", { name: "Send for approval" }).click();
    await expect(pane.getByText("Sent for approval", { exact: true })).toBeVisible();

    // Approving now carries the wording as a modification of the proposal.
    const decision = page.waitForRequest(
      (request) =>
        request.method() === "POST" &&
        (request.postData() ?? "").includes('"Modified"') &&
        (request.postData() ?? "").includes("modified_from_desk"),
    );
    const [request] = await Promise.all([decision, approveAndSettle(page, card, COMMIT_APPROVALS)]);
    const sent = request.postData() ?? "";
    expect(sent).toContain(editedSubject);
    expect(sent).toContain("Thank you, the billing team");
  });
});
