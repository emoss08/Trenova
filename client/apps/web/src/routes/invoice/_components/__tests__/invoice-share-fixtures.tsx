import type { Invoice } from "@trenova/shared/types/invoice";
import type {
  InvoiceShare,
  InvoiceShareUser,
  ShareInvoiceResult,
} from "@trenova/shared/types/invoice-share";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { MemoryRouter } from "react-router";
import { InvoiceShareDialog } from "../invoice-share-dialog";

// Shared by the share dialog's suites. Each suite mocks @/services/api, sonner and the
// avatar itself, since vi.mock applies only to the file that calls it.

export const invoice = {
  id: "inv_1",
  number: "INV-2026-1042",
} as Invoice;

export const dana: InvoiceShareUser = {
  id: "usr_dana",
  name: "Dana Whitfield",
  emailAddress: "dana@example.com",
  profilePicUrl: null,
  thumbnailUrl: null,
};

export const priya: InvoiceShareUser = {
  id: "usr_priya",
  name: "Priya Nair",
  emailAddress: "priya@example.com",
  profilePicUrl: null,
  thumbnailUrl: null,
};

export function share(overrides: Partial<InvoiceShare> = {}): InvoiceShare {
  return {
    id: "invsh_1",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    invoiceId: "inv_1",
    sharedWithId: dana.id,
    sharedById: "usr_me",
    note: null,
    tab: "overview",
    shareCount: 1,
    firstSharedAt: 1_789_000_000,
    lastSharedAt: 1_789_000_000,
    sharedWith: dana,
    sharedBy: { ...priya, id: "usr_me", name: "Marcus Bell" },
    ...overrides,
  };
}

export function result(overrides: Partial<ShareInvoiceResult> = {}): ShareInvoiceResult {
  return {
    shares: [share()],
    recipientCount: 1,
    emailsQueued: 1,
    emailStatus: "Queued",
    ...overrides,
  };
}

export async function openDialog(searchParams = "?item=inv_1") {
  const user = userEvent.setup();
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <NuqsTestingAdapter searchParams={searchParams}>
          <InvoiceShareDialog invoice={invoice} />
        </NuqsTestingAdapter>
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
  await user.click(await within(dialog).findByRole("option", { name: new RegExp(name) }));
}
