import type { InboundMailbox } from "@/lib/graphql/inbox";
import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";
import { MailboxCard } from "../mailbox-card";

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
    hasApiKey: false,
    version: 1,
    createdAt: 1784131200,
    updatedAt: 1784131200,
    ...overrides,
  } as InboundMailbox;
}

function renderCard(value: InboundMailbox, canEdit = true) {
  const onApiKey = vi.fn();
  render(
    <MemoryRouter>
      <MailboxCard
        mailbox={value}
        canEdit={canEdit}
        onEdit={vi.fn()}
        onRotate={vi.fn()}
        onSecret={vi.fn()}
        onApiKey={onApiKey}
      />
    </MemoryRouter>,
  );

  return { onApiKey };
}

describe("MailboxCard", () => {
  it("warns that a listening Resend mailbox with no API key cannot read its mail", () => {
    renderCard(mailbox());

    expect(screen.getByText(NO_KEY_WARNING)).toBeInTheDocument();
  });

  it("offers to set the API key and opens the dialog for it", () => {
    const { onApiKey } = renderCard(mailbox());

    fireEvent.click(screen.getByRole("button", { name: "Set API key" }));

    expect(onApiKey).toHaveBeenCalledOnce();
  });

  it("offers to replace a key that is set, without the warning", () => {
    renderCard(mailbox({ hasApiKey: true }));

    expect(screen.queryByText(NO_KEY_WARNING)).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Replace API key" })).toBeInTheDocument();
  });

  it("asks nothing of a Postmark mailbox, whose webhook carries the whole message", () => {
    renderCard(mailbox({ provider: "Postmark", hasApiKey: false }));

    expect(screen.queryByText(NO_KEY_WARNING)).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /API key/ })).not.toBeInTheDocument();
  });

  it("does not warn about a mailbox that is not listening", () => {
    renderCard(mailbox({ status: "Inactive" }));

    expect(screen.queryByText(NO_KEY_WARNING)).not.toBeInTheDocument();
  });

  it("hides the key button from someone who cannot edit mailboxes", () => {
    renderCard(mailbox(), false);

    expect(screen.queryByRole("button", { name: /API key/ })).not.toBeInTheDocument();
  });
});
