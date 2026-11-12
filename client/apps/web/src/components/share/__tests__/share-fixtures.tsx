import type { RecordShare, RecordShareUser, ShareRecordResult } from "@/lib/graphql/record-share";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { ShareDialog, type SharedRecord } from "../share-dialog";

// Shared by the share dialog's suites. Each suite mocks the share requests, the toast module and the
// avatar itself, since vi.mock applies only to the file that calls it.

/** An invoice, shared the way every record is. */
export const record: SharedRecord = {
  kind: "invoice",
  id: "inv_1",
  title: "invoice INV-2026-1042",
};

/** The address the invoice's page builds for the tab in front. */
export function invoicePath(tab = "overview"): string {
  return tab === "overview"
    ? "/billing/invoices?item=inv_1"
    : `/billing/invoices?item=inv_1&tab=${tab}`;
}

export const dana: RecordShareUser = {
  id: "usr_dana",
  name: "Dana Whitfield",
  username: "dana",
  emailAddress: "dana@example.com",
  profilePicUrl: "",
  thumbnailUrl: "",
};

export const priya: RecordShareUser = {
  id: "usr_priya",
  name: "Priya Nair",
  username: "priya",
  emailAddress: "priya@example.com",
  profilePicUrl: "",
  thumbnailUrl: "",
};

export function share(overrides: Partial<RecordShare> = {}): RecordShare {
  return {
    id: "rsh_1",
    recordId: "inv_1",
    path: invoicePath(),
    note: "",
    shareCount: 1,
    firstSharedAt: 1_789_000_000,
    lastSharedAt: 1_789_000_000,
    removable: false,
    sharedWith: dana,
    sharedBy: { ...priya, id: "usr_me", name: "Marcus Bell" },
    ...overrides,
  };
}

export function result(overrides: Partial<ShareRecordResult> = {}): ShareRecordResult {
  return {
    shares: [share()],
    recipientCount: 1,
    emailsQueued: 1,
    emailStatus: "Queued",
    ...overrides,
  };
}

export async function openDialog(path = invoicePath(), shared: SharedRecord | null = record) {
  const user = userEvent.setup();
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <ShareDialog record={shared} path={path} />
      </QueryClientProvider>
    </MemoryRouter>,
  );

  await user.click(screen.getByRole("button", { name: "Share" }));
  const dialog = await screen.findByRole("dialog");
  return { user, dialog };
}

export async function pickTeammate(
  user: ReturnType<typeof userEvent.setup>,
  dialog: HTMLElement,
  search: string,
  name: string,
) {
  await user.type(within(dialog).getByRole("combobox", { name: "Teammate" }), search);
  await user.click(await screen.findByRole("option", { name: new RegExp(name) }));
}
