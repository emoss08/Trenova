import { expect, type Locator, type Page } from "@playwright/test";

/**
 * Shared steps for the Desk flows. Questions carry a tag the mock's rules
 * match on (e2e/mockllm/script.json) and a run stamp so each test starts its
 * own conversation.
 */

/** The seeded agents the flows talk to. */
export const DISPATCH_AGENT = "Dispatch desk";
export const BILLING_AGENT = "Billing exceptions";

export function question(tag: string, text: string): string {
  return `[${tag}] ${text} (${Date.now().toString(36)})`;
}

export const composer = (page: Page) => page.getByRole("textbox", { name: /^Message / });
export const sendButton = (page: Page) => page.getByRole("button", { name: "Send", exact: true });
export const stopButton = (page: Page) => page.getByRole("button", { name: "Stop the reply" });
export const questionRow = (page: Page, text: string) =>
  page.locator(".dk-scroll .dk-q", { hasText: text });
/** The reply being written, word by word. */
export const streamingReply = (page: Page) => page.locator(".dk-scroll .dk-prose.dk-streaming");
/** Replies that have finished and been saved. */
export const savedReplies = (page: Page) =>
  page.locator(".dk-scroll .dk-r.dk-a .dk-prose:not(.dk-streaming)");
/** The workspace pane beside the conversation. */
export const workspace = (page: Page) => page.getByRole("complementary", { name: "Artifacts" });
/** The change waiting on the person, above the composer, in whichever state it is in. */
export const approvalCard = (page: Page) =>
  page.getByRole("group", { name: /^(Waiting on your approval|Would be refused|Out of date)$/ });

/**
 * Opens the Desk's front page. With "Open Desk to" set to the last
 * conversation the Desk lands there instead, so ⌘N / Ctrl+N goes back to the
 * front page.
 */
export async function openDeskHome(page: Page) {
  await page.goto("/desk");
  await expect(composer(page)).toBeVisible({ timeout: 30_000 });
  // The jump to the last conversation waits on the list of conversations.
  await page.waitForLoadState("networkidle", { timeout: 5_000 }).catch(() => undefined);
  if (/\/desk\/t\//.test(new URL(page.url()).pathname)) {
    await page.keyboard.press("ControlOrMeta+n");
  }
  await expect(page).toHaveURL(/\/desk\/?(\?.*)?$/);
  await expect(page.locator(".dk-greet")).toBeVisible();
  await expect(composer(page)).toBeVisible();
}

/** Points the composer at an agent by name, through its picker. */
export async function chooseAgent(page: Page, name: string) {
  const pill = page.getByRole("button", { name: /^Asking .+\. Choose another agent$/ });
  await expect(pill).toBeVisible({ timeout: 30_000 });
  if ((await pill.getAttribute("aria-label")) === `Asking ${name}. Choose another agent`) {
    return;
  }
  await pill.click();
  await page.getByRole("combobox", { name: "Search agents" }).fill(name);
  await page
    .getByRole("listbox")
    .getByRole("option")
    .filter({ has: page.getByText(name, { exact: true }) })
    .first()
    .click();
  await expect(pill).toHaveAccessibleName(`Asking ${name}. Choose another agent`);
}

/**
 * Asks a question from the front page and waits for the conversation it
 * starts to open with the question at its head.
 */
export async function startConversation(page: Page, agent: string, text: string) {
  await openDeskHome(page);
  await chooseAgent(page, agent);
  await composer(page).fill(text);
  await sendButton(page).click();
  await page.waitForURL(/\/desk\/t\//, { timeout: 30_000 });
  await expect(questionRow(page, text)).toBeVisible();
}

/** Waits for the turn to end: nothing is being written and the composer can send again. */
export async function waitForTurnToEnd(page: Page, timeout = 90_000) {
  await expect(stopButton(page)).toBeHidden({ timeout });
  await expect(streamingReply(page)).toHaveCount(0, { timeout });
  await expect(sendButton(page)).toBeVisible();
}

/** Opens the workspace pane if it is closed. */
export async function openWorkspace(page: Page) {
  const toggle = page.getByRole("button", { name: "Workspace", exact: true });
  await expect(toggle).toBeVisible();
  if ((await toggle.getAttribute("aria-pressed")) !== "true") {
    await toggle.click();
  }
  await expect(toggle).toHaveAttribute("aria-pressed", "true");
}

/**
 * Records, from the first frame of every page, each element under `selector`
 * whose text ever contained `needle`, however briefly. Read it back with
 * `seenText`. A MutationObserver sees every DOM change, so a lookup that
 * shows for one frame and goes is still caught.
 */
export async function watchForText(page: Page, selector: string, needle: string) {
  await page.addInitScript(
    ({ selector, needle }) => {
      const seen: string[] = [];
      (window as unknown as { __e2eSeen: string[] }).__e2eSeen = seen;
      const check = () => {
        for (const element of document.querySelectorAll(selector)) {
          if (element.textContent?.includes(needle)) {
            const what = `${element.className}: ${element.textContent.slice(0, 120)}`;
            if (!seen.includes(what)) {
              seen.push(what);
            }
          }
        }
      };
      new MutationObserver(check).observe(document, {
        subtree: true,
        childList: true,
        characterData: true,
      });
    },
    { selector, needle },
  );
}

export async function seenText(page: Page): Promise<string[]> {
  return page.evaluate(() => (window as unknown as { __e2eSeen?: string[] }).__e2eSeen ?? []);
}

/**
 * Approves the change on the card, then either lets it go through or takes it
 * back inside its undo window, so the run leaves the records as they were.
 */
export async function approveAndSettle(page: Page, card: Locator, commit: boolean) {
  const approve = card.getByRole("button", { name: /^Approve/ });
  await expect(approve).toBeEnabled({ timeout: 30_000 });
  await approve.click();

  // Held for its undo window, or carried out at once: either way the dock
  // says it was approved.
  const undoBar = page
    .getByRole("status")
    .filter({ has: page.getByRole("button", { name: "Undo" }) });
  const approved = page.getByRole("status").filter({ hasText: /Approved/ });
  await expect(approved.first()).toBeVisible();

  if (!(await undoBar.isVisible())) {
    await expect(card).toBeHidden();
    return "committed" as const;
  }
  await expect(undoBar).toContainText(/Approved/);
  if (commit) {
    await undoBar.getByRole("button", { name: "Do it now" }).click();
    await expect(undoBar).toBeHidden();
    await expect(card).toBeHidden();
    return "committed" as const;
  }
  await undoBar.getByRole("button", { name: "Undo" }).click();
  await expect(undoBar).toBeHidden();
  return "undone" as const;
}
