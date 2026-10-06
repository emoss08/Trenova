import type { InboundMailbox } from "@/lib/graphql/inbox";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { ApiKeyDialog } from "../api-key-dialog";

const mocks = vi.hoisted(() => ({
  setInboundMailboxApiKey: vi.fn<(id: string, apiKey: string) => Promise<unknown>>(),
}));

vi.mock("@/lib/graphql/inbox", () => ({
  setInboundMailboxApiKey: mocks.setInboundMailboxApiKey,
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

function mailbox(id: string, name: string): InboundMailbox {
  return {
    __typename: "InboundMailbox",
    id,
    name,
    address: `${name.toLowerCase()}@acme-logistics.com`,
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
  } as InboundMailbox;
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => {
    resolve = done;
  });

  return { promise, resolve };
}

function withClient(client: QueryClient, node: ReactNode) {
  return <QueryClientProvider client={client}>{node}</QueryClientProvider>;
}

describe("ApiKeyDialog", () => {
  it("closes the dialog when the key it saved is stored", async () => {
    mocks.setInboundMailboxApiKey.mockResolvedValueOnce(mailbox("imbx_a", "Tenders"));
    const onClose = vi.fn();
    const client = new QueryClient();

    render(
      withClient(
        client,
        <ApiKeyDialog mailbox={mailbox("imbx_a", "Tenders")} onClose={onClose} onSaved={vi.fn()} />,
      ),
    );

    fireEvent.change(screen.getByLabelText("API key"), { target: { value: " re_key_a " } });
    fireEvent.click(screen.getByRole("button", { name: "Save API key" }));

    await waitFor(() => expect(onClose).toHaveBeenCalledOnce());
    expect(mocks.setInboundMailboxApiKey).toHaveBeenCalledWith("imbx_a", "re_key_a");
  });

  /*
   * A save is still in flight when the person dismisses its dialog and opens
   * another mailbox's. When the first save lands it must not close the second
   * dialog or throw away the key being typed into it.
   */
  it("leaves another mailbox's dialog open when an earlier save finishes", async () => {
    const pending = deferred<unknown>();
    mocks.setInboundMailboxApiKey.mockReturnValueOnce(pending.promise);
    const onClose = vi.fn();
    const client = new QueryClient();
    const tenders = mailbox("imbx_a", "Tenders");
    const pods = mailbox("imbx_b", "PODs");

    const { rerender } = render(
      withClient(client, <ApiKeyDialog mailbox={tenders} onClose={onClose} onSaved={vi.fn()} />),
    );

    fireEvent.change(screen.getByLabelText("API key"), { target: { value: "re_key_a" } });
    fireEvent.click(screen.getByRole("button", { name: "Save API key" }));

    rerender(
      withClient(client, <ApiKeyDialog mailbox={null} onClose={onClose} onSaved={vi.fn()} />),
    );
    rerender(
      withClient(client, <ApiKeyDialog mailbox={pods} onClose={onClose} onSaved={vi.fn()} />),
    );
    fireEvent.change(screen.getByLabelText("API key"), { target: { value: "re_key_b" } });

    await act(async () => {
      pending.resolve(tenders);
      await pending.promise;
    });

    expect(onClose).not.toHaveBeenCalled();
    expect(screen.getByLabelText("API key")).toHaveValue("re_key_b");
  });

  it("leaves the same mailbox's reopened dialog open when the earlier save finishes", async () => {
    const pending = deferred<unknown>();
    mocks.setInboundMailboxApiKey.mockReturnValueOnce(pending.promise);
    const onClose = vi.fn();
    const client = new QueryClient();
    const tenders = mailbox("imbx_a", "Tenders");

    const { rerender } = render(
      withClient(client, <ApiKeyDialog mailbox={tenders} onClose={onClose} onSaved={vi.fn()} />),
    );

    fireEvent.change(screen.getByLabelText("API key"), { target: { value: "re_first" } });
    fireEvent.click(screen.getByRole("button", { name: "Save API key" }));

    rerender(
      withClient(client, <ApiKeyDialog mailbox={null} onClose={onClose} onSaved={vi.fn()} />),
    );
    rerender(
      withClient(client, <ApiKeyDialog mailbox={tenders} onClose={onClose} onSaved={vi.fn()} />),
    );
    fireEvent.change(screen.getByLabelText("API key"), { target: { value: "re_second" } });

    await act(async () => {
      pending.resolve(tenders);
      await pending.promise;
    });

    expect(onClose).not.toHaveBeenCalled();
    expect(screen.getByLabelText("API key")).toHaveValue("re_second");
  });
});
