import type { InboundMailbox } from "@/lib/graphql/inbox";
import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";
import { MailboxCard } from "../mailbox-card";

const NOW = 1786723200;
const HOUR = 3600;
const DAY = 86_400;

const NO_SECRET_WARNING = "No signing secret is set, so every delivery to this mailbox is refused.";
const NO_KEY_WARNING =
  "No API key is set, so mail arrives without its body or attachments and waits for a person.";

function mailbox(overrides: Partial<InboundMailbox> = {}): InboundMailbox {
  return {
    __typename: "InboundMailbox",
    id: "imbx_01",
    name: "Tenders",
    address: "tenders@acme-logistics.com",
    provider: "Resend",
    purpose: "",
    reviewPolicy: "AlwaysReview",
    minConfidence: 0.8,
    status: "Active",
    hasSigningSecret: true,
    hasApiKey: true,
    lastReceivedAt: NOW - 2 * HOUR,
    health: "Healthy",
    testSentAt: null,
    testReceivedAt: null,
    blockedSenders: [],
    version: 1,
    createdAt: NOW - 30 * DAY,
    updatedAt: NOW - 30 * DAY,
    ...overrides,
  } as InboundMailbox;
}

function renderCard(
  value: InboundMailbox,
  canEdit = true,
  onSendTest = vi.fn(() => Promise.resolve()),
) {
  const handlers = {
    onEdit: vi.fn(),
    onRotate: vi.fn(),
    onSecret: vi.fn(),
    onApiKey: vi.fn(),
    onSendTest,
  };
  render(
    <MemoryRouter>
      <MailboxCard mailbox={value} canEdit={canEdit} now={NOW} {...handlers} />
    </MemoryRouter>,
  );

  return handlers;
}

describe("MailboxCard", () => {
  it("shows a receiving mailbox with no callout and no setup progress", () => {
    renderCard(mailbox());

    expect(screen.getByText("Receiving")).toBeInTheDocument();
    expect(screen.getByText(/Last mail 2 hours ago/)).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /done/ })).not.toBeInTheDocument();
  });

  it("leads a listening mailbox without a secret with the refusal and the button that fixes it", async () => {
    const { onSecret } = renderCard(mailbox({ hasSigningSecret: false, health: "NeverReceived" }));

    expect(screen.getByText("Refusing mail")).toBeInTheDocument();
    expect(screen.getByRole("alert")).toHaveTextContent(NO_SECRET_WARNING);

    await userEvent.click(screen.getByRole("button", { name: "Set signing secret" }));

    expect(onSecret).toHaveBeenCalledOnce();
  });

  it("names only the missing secret when a Resend mailbox lacks both secret and key", () => {
    renderCard(mailbox({ hasSigningSecret: false, hasApiKey: false, health: "NeverReceived" }));

    expect(screen.getAllByRole("alert")).toHaveLength(1);
    expect(screen.getByRole("alert")).toHaveTextContent(NO_SECRET_WARNING);
    expect(screen.queryByText(NO_KEY_WARNING)).not.toBeInTheDocument();
  });

  it("warns that a listening Resend mailbox with no API key cannot read its mail", async () => {
    const { onApiKey } = renderCard(mailbox({ hasApiKey: false }));

    expect(screen.getByRole("alert")).toHaveTextContent(NO_KEY_WARNING);

    await userEvent.click(screen.getByRole("button", { name: "Set API key" }));

    expect(onApiKey).toHaveBeenCalledOnce();
  });

  it("asks nothing of a Postmark mailbox, whose webhook carries the whole message", async () => {
    renderCard(mailbox({ provider: "Postmark", hasApiKey: false }));

    expect(screen.queryByRole("alert")).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "Mailbox actions" }));

    expect(await screen.findByRole("menuitem", { name: "Edit" })).toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: /API key/ })).not.toBeInTheDocument();
  });

  it("says how long a quiet mailbox has gone without mail and offers a test email", async () => {
    const { onSendTest } = renderCard(
      mailbox({ health: "Quiet", lastReceivedAt: NOW - 9 * DAY - HOUR }),
    );

    expect(screen.getByText("Gone quiet")).toBeInTheDocument();
    expect(screen.getByRole("alert")).toHaveTextContent(
      "No mail in 9 days. Send a test email to check the address still works.",
    );

    await userEvent.click(screen.getByRole("button", { name: "Send a test email" }));

    expect(onSendTest).toHaveBeenCalledOnce();
  });

  it("reports a test email that was sent but has not arrived", () => {
    renderCard(
      mailbox({
        health: "NeverReceived",
        lastReceivedAt: null,
        testSentAt: NOW - 5 * 60,
        testReceivedAt: null,
      }),
    );

    expect(screen.getByText("Waiting for mail")).toBeInTheDocument();
    expect(screen.getByRole("alert")).toHaveTextContent(
      "A test email was sent 5 minutes ago and hasn't arrived yet.",
    );
  });

  it("raises nothing on a paused mailbox, even one without a secret", () => {
    renderCard(mailbox({ status: "Inactive", health: "Paused", hasSigningSecret: false }));

    expect(screen.getByText("Paused")).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("counts unfinished setup and lists every step when opened", async () => {
    const { onApiKey } = renderCard(
      mailbox({ hasApiKey: false, health: "NeverReceived", lastReceivedAt: null }),
    );

    const progress = screen.getByRole("button", { name: /1 of 3 done/ });
    expect(screen.queryByText("First delivery")).not.toBeInTheDocument();

    await userEvent.click(progress);

    expect(screen.getByText("Deliveries are verified")).toBeInTheDocument();
    expect(screen.getByText("Mail arrives without its body or attachments")).toBeInTheDocument();
    expect(screen.getByText("Nothing has arrived yet")).toBeInTheDocument();

    const keyFixes = screen.getAllByRole("button", { name: "Set API key" });
    expect(keyFixes).toHaveLength(2);
    await userEvent.click(keyFixes[1]);

    expect(onApiKey).toHaveBeenCalledOnce();
  });

  it("counts a delivered test as the first delivery", () => {
    renderCard(
      mailbox({
        hasApiKey: false,
        health: "NeverReceived",
        lastReceivedAt: null,
        testSentAt: NOW - 2 * HOUR,
        testReceivedAt: NOW - HOUR,
      }),
    );

    expect(screen.getByRole("button", { name: /2 of 3 done/ })).toBeInTheDocument();
  });

  it("offers to replace a key that is set from the actions menu", async () => {
    const { onApiKey } = renderCard(mailbox());

    await userEvent.click(screen.getByRole("button", { name: "Mailbox actions" }));
    await userEvent.click(await screen.findByRole("menuitem", { name: "Replace API key" }));

    expect(onApiKey).toHaveBeenCalledOnce();
  });

  it("shows a problem but no fixes or actions to someone who cannot edit mailboxes", () => {
    renderCard(mailbox({ hasApiKey: false }), false);

    expect(screen.getByRole("alert")).toHaveTextContent(NO_KEY_WARNING);
    expect(screen.queryByRole("button", { name: /API key/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Mailbox actions" })).not.toBeInTheDocument();
  });

  it("sends one test email however often it is clicked while the first is on its way", async () => {
    let finishSend = () => {};
    const sending = new Promise<void>((resolve) => {
      finishSend = resolve;
    });
    const onSendTest = vi.fn(() => sending);
    renderCard(mailbox({ health: "Quiet", lastReceivedAt: NOW - 9 * DAY }), true, onSendTest);

    const button = screen.getByRole("button", { name: "Send a test email" });
    await userEvent.dblClick(button);
    await userEvent.click(screen.getByRole("button", { name: /Sending/ }));

    expect(onSendTest).toHaveBeenCalledOnce();
    expect(screen.getByRole("button", { name: /Sending/ })).toBeDisabled();

    await userEvent.click(screen.getByRole("button", { name: "Mailbox actions" }));
    const menuItem = await screen.findByRole("menuitem", { name: "Send a test email" });
    expect(menuItem).toHaveAttribute("aria-disabled", "true");
    await userEvent.click(menuItem);

    expect(onSendTest).toHaveBeenCalledOnce();

    await act(async () => finishSend());

    expect(screen.getByRole("button", { name: "Send a test email" })).toBeEnabled();
  });

  it("lets a test be sent again after a send fails", async () => {
    const onSendTest = vi
      .fn<() => Promise<void>>()
      .mockRejectedValueOnce(new Error("No email profile"))
      .mockResolvedValueOnce(undefined);
    renderCard(mailbox({ health: "Quiet", lastReceivedAt: NOW - 9 * DAY }), true, onSendTest);

    await userEvent.click(screen.getByRole("button", { name: "Send a test email" }));
    await userEvent.click(await screen.findByRole("button", { name: "Send a test email" }));

    expect(onSendTest).toHaveBeenCalledTimes(2);
  });
});
