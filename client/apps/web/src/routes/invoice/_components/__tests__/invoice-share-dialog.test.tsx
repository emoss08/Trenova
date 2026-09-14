import { ApiRequestError } from "@trenova/shared/lib/api";
import type { Invoice } from "@trenova/shared/types/invoice";
import type {
  InvoiceShare,
  InvoiceShareUser,
  ShareInvoiceResult,
} from "@trenova/shared/types/invoice-share";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { InvoiceShareDialog } from "../invoice-share-dialog";

const mocks = vi.hoisted(() => ({
  list: vi.fn(),
  share: vi.fn(),
  candidates: vi.fn(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}));

vi.mock("@/services/api", () => ({
  apiService: {
    invoiceShareService: {
      list: mocks.list,
      share: mocks.share,
      candidates: mocks.candidates,
    },
  },
}));
vi.mock("sonner", () => ({ toast: { success: mocks.toastSuccess, error: mocks.toastError } }));
vi.mock("@/components/resolved-user-avatar", () => ({ ResolvedUserAvatar: () => null }));

const ORIGIN = window.location.origin;

const invoice = {
  id: "inv_1",
  number: "INV-2026-1042",
} as Invoice;

const dana: InvoiceShareUser = {
  id: "usr_dana",
  name: "Dana Whitfield",
  emailAddress: "dana@example.com",
  profilePicUrl: null,
  thumbnailUrl: null,
};

const priya: InvoiceShareUser = {
  id: "usr_priya",
  name: "Priya Nair",
  emailAddress: "priya@example.com",
  profilePicUrl: null,
  thumbnailUrl: null,
};

function share(overrides: Partial<InvoiceShare> = {}): InvoiceShare {
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

function result(overrides: Partial<ShareInvoiceResult> = {}): ShareInvoiceResult {
  return {
    shares: [share()],
    recipientCount: 1,
    emailsQueued: 1,
    emailStatus: "Queued",
    ...overrides,
  };
}

async function openDialog(searchParams = "?item=inv_1") {
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

async function pickTeammate(
  user: ReturnType<typeof userEvent.setup>,
  dialog: HTMLElement,
  search: string,
  name: string,
) {
  await user.type(within(dialog).getByRole("combobox", { name: "Teammate" }), search);
  await user.click(await within(dialog).findByRole("option", { name: new RegExp(name) }));
}

beforeEach(() => {
  mocks.list.mockResolvedValue([]);
  mocks.candidates.mockResolvedValue([dana, priya]);
});

afterEach(() => {
  vi.clearAllMocks();
});

describe("invoice share dialog", () => {
  it("links to the invoice on the tab that is open", async () => {
    const { dialog } = await openDialog("?item=inv_1&tab=charges");

    expect(
      within(dialog).getByRole("heading", { name: "Share invoice INV-2026-1042" }),
    ).toBeVisible();
    expect(within(dialog).getByRole("textbox", { name: "Invoice link" })).toHaveValue(
      `${ORIGIN}/billing/invoices?item=inv_1&tab=charges`,
    );
  });

  it("leaves the tab out of the link when the overview is open", async () => {
    const { dialog } = await openDialog("?item=inv_1");

    expect(within(dialog).getByRole("textbox", { name: "Invoice link" })).toHaveValue(
      `${ORIGIN}/billing/invoices?item=inv_1`,
    );
  });

  it("copies the link from the field and from the footer", async () => {
    const { user, dialog } = await openDialog("?item=inv_1&tab=documents");
    const link = `${ORIGIN}/billing/invoices?item=inv_1&tab=documents`;

    await user.click(within(dialog).getByRole("button", { name: "Copy" }));
    expect(await navigator.clipboard.readText()).toBe(link);
    const copiedButtons = within(dialog).getAllByRole("button", { name: "Copied" });
    expect(copiedButtons).toHaveLength(2);
    expect(mocks.toastSuccess).toHaveBeenCalledWith("Copied to clipboard");

    await navigator.clipboard.writeText("");
    await user.click(copiedButtons[1]);
    expect(await navigator.clipboard.readText()).toBe(link);
  });

  it("only offers teammates the server says can view invoices", async () => {
    const { user, dialog } = await openDialog();

    await user.type(within(dialog).getByRole("combobox", { name: "Teammate" }), "a");

    const listbox = await within(dialog).findByRole("listbox", {
      name: "Teammates who can view invoices",
    });
    await waitFor(() =>
      expect(mocks.candidates).toHaveBeenLastCalledWith("inv_1", "a", expect.anything()),
    );
    expect(within(listbox).getAllByRole("option")).toHaveLength(2);
    expect(within(listbox).getByRole("option", { name: /Dana Whitfield/ })).toBeVisible();
  });

  it("says when no teammate who can view invoices matches the search", async () => {
    mocks.candidates.mockResolvedValue([]);
    const { user, dialog } = await openDialog();

    await user.type(within(dialog).getByRole("combobox", { name: "Teammate" }), "normal");

    expect(
      await within(dialog).findByText("No teammates who can view invoices match."),
    ).toBeVisible();
  });

  it("picks a teammate with the keyboard", async () => {
    mocks.share.mockResolvedValue(result({ shares: [share({ sharedWith: priya })] }));
    const { user, dialog } = await openDialog();

    const combobox = within(dialog).getByRole("combobox", { name: "Teammate" });
    await user.type(combobox, "a");
    await within(dialog).findByRole("option", { name: /Priya Nair/ });
    await user.keyboard("{ArrowDown}{Enter}");

    expect(combobox).toHaveValue("Priya Nair");
    await user.click(within(dialog).getByRole("button", { name: "Invite" }));

    await waitFor(() =>
      expect(mocks.share).toHaveBeenCalledWith("inv_1", {
        userIds: [priya.id],
        note: "",
        tab: "overview",
      }),
    );
  });

  it("invites the picked teammate with a note and the open tab, then lists them", async () => {
    mocks.share.mockResolvedValue(
      result({ shares: [share({ note: "Check the detention line" })] }),
    );
    const { user, dialog } = await openDialog("?item=inv_1&tab=charges");

    expect(await within(dialog).findByText("Not shared with anyone yet.")).toBeVisible();
    expect(within(dialog).queryByRole("textbox", { name: "Note" })).toBeNull();

    await pickTeammate(user, dialog, "da", "Dana Whitfield");
    await user.type(
      within(dialog).getByRole("textbox", { name: "Note" }),
      "Check the detention line",
    );
    await user.click(within(dialog).getByRole("button", { name: "Invite" }));

    await waitFor(() =>
      expect(mocks.share).toHaveBeenCalledWith("inv_1", {
        userIds: [dana.id],
        note: "Check the detention line",
        tab: "charges",
      }),
    );
    expect(mocks.toastSuccess).toHaveBeenCalledWith(
      "Shared with 1 teammate. They'll get a notification and an email.",
    );

    const list = await within(dialog).findByRole("list", { name: "Shared with" });
    expect(within(list).getByText("Dana Whitfield")).toBeVisible();
    expect(within(list).getByText("dana@example.com")).toBeVisible();
    expect(within(list).getByText("Can view")).toBeVisible();
    expect(within(dialog).getByRole("combobox", { name: "Teammate" })).toHaveValue("");
    expect(within(dialog).queryByRole("textbox", { name: "Note" })).toBeNull();
  });

  it("clears the picked teammate", async () => {
    const { user, dialog } = await openDialog();

    await pickTeammate(user, dialog, "da", "Dana Whitfield");
    await user.click(within(dialog).getByRole("button", { name: "Clear teammate" }));

    expect(within(dialog).getByRole("combobox", { name: "Teammate" })).toHaveValue("");
    expect(within(dialog).queryByRole("textbox", { name: "Note" })).toBeNull();
  });

  it.each([
    [
      "NotConfigured",
      0,
      "Shared with 1 teammate. They'll see it in their notifications; email isn't set up for shares.",
    ],
    [
      "Partial",
      1,
      "Shared with 1 teammate. Everyone got a notification, but some emails couldn't be sent.",
    ],
    [
      "Failed",
      0,
      "Shared with 1 teammate. Everyone got a notification, but the emails couldn't be sent.",
    ],
  ] as const)("tells the sharer when email delivery is %s", async (status, queued, message) => {
    mocks.share.mockResolvedValue(result({ emailsQueued: queued, emailStatus: status }));
    const { user, dialog } = await openDialog();

    await pickTeammate(user, dialog, "da", "Dana Whitfield");
    await user.click(within(dialog).getByRole("button", { name: "Invite" }));

    await waitFor(() => expect(mocks.toastSuccess).toHaveBeenCalledWith(message));
  });

  it("asks for a teammate before sending anything", async () => {
    const { user, dialog } = await openDialog();

    await user.click(within(dialog).getByRole("button", { name: "Invite" }));

    expect(
      await within(dialog).findByText("Choose at least one teammate to share with"),
    ).toBeVisible();
    expect(mocks.share).not.toHaveBeenCalled();
  });

  it("shows why the server refused a recipient", async () => {
    mocks.share.mockRejectedValue(
      new ApiRequestError(422, {
        type: "https://api.trenova.app/problems/validation-error",
        title: "Validation error",
        status: 422,
        detail: "Validation failed",
        errors: [
          {
            field: "userIds[0]",
            code: "INVALID_OPERATION",
            message: "Dana Whitfield cannot view invoices, so the link would not open for them",
          },
        ],
      }),
    );
    const { user, dialog } = await openDialog();

    await pickTeammate(user, dialog, "da", "Dana Whitfield");
    await user.click(within(dialog).getByRole("button", { name: "Invite" }));

    expect(await within(dialog).findByText(/Dana Whitfield cannot view invoices/)).toBeVisible();
    expect(mocks.toastSuccess).not.toHaveBeenCalled();
  });

  it("lists who the invoice is already shared with in the order the server sends", async () => {
    mocks.list.mockResolvedValue([
      share({ id: "invsh_2", sharedWithId: priya.id, sharedWith: priya }),
      share(),
    ]);
    const { dialog } = await openDialog();

    const list = await within(dialog).findByRole("list", { name: "Shared with" });
    const rows = within(list).getAllByRole("listitem");
    expect(rows).toHaveLength(2);
    expect(within(rows[0]).getByText("Priya Nair")).toBeVisible();
    expect(within(rows[1]).getByText("Dana Whitfield")).toBeVisible();
    expect(mocks.list).toHaveBeenCalledWith("inv_1");
  });

  it("says so when the shared-with list cannot load", async () => {
    mocks.list.mockRejectedValue(new Error("boom"));
    const { dialog } = await openDialog();

    expect(
      await within(dialog).findByText("Couldn't load who this invoice is shared with."),
    ).toBeVisible();
  });

  it("does not load anything until the dialog opens", () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <MemoryRouter>
        <QueryClientProvider client={client}>
          <NuqsTestingAdapter searchParams="?item=inv_1">
            <InvoiceShareDialog invoice={invoice} />
          </NuqsTestingAdapter>
        </QueryClientProvider>
      </MemoryRouter>,
    );

    expect(screen.queryByRole("dialog")).toBeNull();
    expect(mocks.list).not.toHaveBeenCalled();
    expect(mocks.candidates).not.toHaveBeenCalled();
  });
});
