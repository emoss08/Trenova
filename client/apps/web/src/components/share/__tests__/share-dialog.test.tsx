import { clearPermissions, seedPermissions } from "@/test/permissions";
import { Operation } from "@trenova/shared/types/permission";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ShareDialog } from "../share-dialog";
import {
  dana,
  invoicePath,
  openDialog,
  pickTeammate,
  priya,
  record,
  result,
  share,
} from "./share-fixtures";

const mocks = vi.hoisted(() => ({
  list: vi.fn(),
  share: vi.fn(),
  unshare: vi.fn(),
  candidates: vi.fn(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}));

vi.mock("@/lib/graphql/record-share", () => ({
  fetchRecordShares: mocks.list,
  shareRecord: mocks.share,
  unshareRecord: mocks.unshare,
  fetchRecordShareCandidates: mocks.candidates,
}));
vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: { success: mocks.toastSuccess, error: mocks.toastError },
}));
vi.mock("@/components/resolved-user-avatar", () => ({ ResolvedUserAvatar: () => null }));

const ORIGIN = window.location.origin;

beforeEach(() => {
  seedPermissions({ record_share: Operation.Read | Operation.Create });
  mocks.list.mockResolvedValue([]);
  mocks.candidates.mockResolvedValue([dana, priya]);
});

afterEach(() => {
  vi.clearAllMocks();
  clearPermissions();
});

describe("the share dialog", () => {
  it("links to where the sharer is, on the tab that is open", async () => {
    const { dialog } = await openDialog(invoicePath("charges"));

    expect(
      within(dialog).getByRole("heading", { name: "Share invoice INV-2026-1042" }),
    ).toBeVisible();
    expect(within(dialog).getByRole("textbox", { name: "Link" })).toHaveValue(
      `${ORIGIN}/billing/invoices?item=inv_1&tab=charges`,
    );
  });

  it("links to the address it was given and nothing more", async () => {
    const { dialog } = await openDialog(invoicePath());

    expect(within(dialog).getByRole("textbox", { name: "Link" })).toHaveValue(
      `${ORIGIN}/billing/invoices?item=inv_1`,
    );
  });

  it("copies the link from the field and from the footer", async () => {
    const { user, dialog } = await openDialog(invoicePath("documents"));
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

  it("only offers teammates the server says can view the record", async () => {
    const { user, dialog } = await openDialog();

    await user.type(within(dialog).getByRole("combobox", { name: "Teammate" }), "a");

    const listbox = await screen.findByRole("listbox", {
      name: "Teammates who can view this",
    });
    await waitFor(() =>
      expect(mocks.candidates).toHaveBeenLastCalledWith("invoice", "inv_1", "a", expect.anything()),
    );
    expect(within(listbox).getAllByRole("option")).toHaveLength(2);
    expect(within(listbox).getByRole("option", { name: /Dana Whitfield/ })).toBeVisible();
  });

  it("says when no teammate who can view the record matches the search", async () => {
    mocks.candidates.mockResolvedValue([]);
    const { user, dialog } = await openDialog();

    await user.type(within(dialog).getByRole("combobox", { name: "Teammate" }), "normal");

    expect(await screen.findByText("No teammates who can view this match.")).toBeVisible();
  });

  it("picks a teammate with the keyboard", async () => {
    mocks.share.mockResolvedValue(result({ shares: [share({ sharedWith: priya })] }));
    const { user, dialog } = await openDialog();

    const combobox = within(dialog).getByRole("combobox", { name: "Teammate" });
    await user.type(combobox, "a");
    await screen.findByRole("option", { name: /Priya Nair/ });
    await user.keyboard("{ArrowDown}{Enter}");

    expect(combobox).toHaveValue("Priya Nair");
    await user.click(within(dialog).getByRole("button", { name: "Invite" }));

    await waitFor(() =>
      expect(mocks.share).toHaveBeenCalledWith({
        kind: "invoice",
        id: "inv_1",
        path: invoicePath(),
        userIds: [priya.id],
        note: "",
      }),
    );
  });

  it("invites the picked teammate with a note and the address in front, then lists them", async () => {
    mocks.share.mockResolvedValue(
      result({ shares: [share({ note: "Check the detention line" })] }),
    );
    const { user, dialog } = await openDialog(invoicePath("charges"));

    expect(await within(dialog).findByText("Not shared with anyone yet.")).toBeVisible();
    expect(within(dialog).queryByRole("textbox", { name: "Note" })).toBeNull();

    await pickTeammate(user, dialog, "da", "Dana Whitfield");
    await user.type(
      within(dialog).getByRole("textbox", { name: "Note" }),
      "Check the detention line",
    );
    await user.click(within(dialog).getByRole("button", { name: "Invite" }));

    await waitFor(() =>
      expect(mocks.share).toHaveBeenCalledWith({
        kind: "invoice",
        id: "inv_1",
        path: invoicePath("charges"),
        userIds: [dana.id],
        note: "Check the detention line",
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
            path: ["userIds", 0],
            field: "userIds.0",
            code: "INVALID_OPERATION",
            message:
              "Dana Whitfield cannot view this kind of record, so the link would not open for them",
          },
        ],
      }),
    );
    const { user, dialog } = await openDialog();

    await pickTeammate(user, dialog, "da", "Dana Whitfield");
    await user.click(within(dialog).getByRole("button", { name: "Invite" }));

    expect(
      await within(dialog).findByText(/Dana Whitfield cannot view this kind of record/),
    ).toBeVisible();
    expect(mocks.toastSuccess).not.toHaveBeenCalled();
  });

  it("lists who the record is already shared with in the order the server sends", async () => {
    mocks.list.mockResolvedValue([share({ id: "rsh_2", sharedWith: priya }), share()]);
    const { dialog } = await openDialog();

    const list = await within(dialog).findByRole("list", { name: "Shared with" });
    const rows = within(list).getAllByRole("listitem");
    expect(rows).toHaveLength(2);
    expect(within(rows[0]).getByText("Priya Nair")).toBeVisible();
    expect(within(rows[1]).getByText("Dana Whitfield")).toBeVisible();
    expect(mocks.list).toHaveBeenCalledWith("invoice", "inv_1", expect.anything());
  });

  it("offers Remove only on a share the server says the reader may take back", async () => {
    mocks.list.mockResolvedValue([
      share({ id: "rsh_2", sharedWith: priya, removable: false }),
      share({ id: "rsh_1", removable: true }),
    ]);
    const { dialog } = await openDialog();

    const list = await within(dialog).findByRole("list", { name: "Shared with" });
    const [theirs, mine] = within(list).getAllByRole("listitem");
    expect(within(theirs).queryByRole("button", { name: /Remove/ })).toBeNull();
    expect(within(mine).getByRole("button", { name: "Remove Dana Whitfield" })).toBeVisible();
  });

  it("removes a share and lists who the record is still shared with", async () => {
    mocks.list.mockResolvedValue([
      share({ id: "rsh_2", sharedWith: priya }),
      share({ id: "rsh_1", removable: true }),
    ]);
    mocks.unshare.mockResolvedValue([share({ id: "rsh_2", sharedWith: priya })]);
    const { dialog, user } = await openDialog();

    const list = await within(dialog).findByRole("list", { name: "Shared with" });
    await user.click(within(list).getByRole("button", { name: "Remove Dana Whitfield" }));

    await waitFor(() => expect(within(list).getAllByRole("listitem")).toHaveLength(1));
    expect(within(list).getByText("Priya Nair")).toBeVisible();
    expect(within(list).queryByText("Dana Whitfield")).toBeNull();
    expect(mocks.unshare).toHaveBeenCalledExactlyOnceWith("invoice", "inv_1", "rsh_1");
  });

  it("says so when the shared-with list cannot load", async () => {
    mocks.list.mockRejectedValue(new Error("boom"));
    const { dialog } = await openDialog();

    expect(await within(dialog).findByText("Couldn't load who this is shared with.")).toBeVisible();
  });

  it("does not load anything until the dialog opens", () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <MemoryRouter>
        <QueryClientProvider client={client}>
          <ShareDialog record={record} path={invoicePath()} />
        </QueryClientProvider>
      </MemoryRouter>,
    );

    expect(screen.queryByRole("dialog")).toBeNull();
    expect(mocks.list).not.toHaveBeenCalled();
    expect(mocks.candidates).not.toHaveBeenCalled();
  });

  // A conversation or a list is no one record: there is nobody to invite to
  // it, so the dialog only offers its link.
  it("offers only the link for a place that is no one record", async () => {
    const { dialog } = await openDialog("/desk/t/athr_1", null);

    expect(within(dialog).getByRole("heading", { name: "Share this page" })).toBeVisible();
    expect(within(dialog).getByRole("textbox", { name: "Link" })).toHaveValue(
      `${ORIGIN}/desk/t/athr_1`,
    );
    expect(within(dialog).queryByRole("combobox", { name: "Teammate" })).toBeNull();
    expect(within(dialog).queryByRole("button", { name: "Invite" })).toBeNull();
    expect(mocks.list).not.toHaveBeenCalled();
  });

  it("shares any kind of record the same way", async () => {
    mocks.share.mockResolvedValue(result());
    const shipment = { kind: "shipment", id: "shp_1", title: "SEED-SHP-010" };
    const { user, dialog } = await openDialog("/desk/r/shipment/shp_1", shipment);

    expect(within(dialog).getByRole("heading", { name: "Share SEED-SHP-010" })).toBeVisible();
    await pickTeammate(user, dialog, "da", "Dana Whitfield");
    await user.click(within(dialog).getByRole("button", { name: "Invite" }));

    await waitFor(() =>
      expect(mocks.share).toHaveBeenCalledWith({
        kind: "shipment",
        id: "shp_1",
        path: "/desk/r/shipment/shp_1",
        userIds: [dana.id],
        note: "",
      }),
    );
    expect(mocks.candidates).toHaveBeenCalledWith(
      "shipment",
      "shp_1",
      expect.any(String),
      expect.anything(),
    );
  });

  it("says what sharing does where the place it is opened from says so", async () => {
    const user = (await import("@testing-library/user-event")).default.setup();
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <MemoryRouter>
        <QueryClientProvider client={client}>
          <ShareDialog
            record={record}
            path={invoicePath()}
            description="A case someone else owns comes into their view."
          />
        </QueryClientProvider>
      </MemoryRouter>,
    );

    await user.click(screen.getByRole("button", { name: "Share" }));
    const dialog = await screen.findByRole("dialog");

    expect(
      within(dialog).getByText("A case someone else owns comes into their view."),
    ).toBeVisible();
    expect(within(dialog).queryByText(/Sharing doesn't change who can see it/)).toBeNull();
  });

  it("offers only the link to someone who may not share records", async () => {
    clearPermissions();
    seedPermissions({});
    const { dialog } = await openDialog(invoicePath("charges"));

    expect(within(dialog).getByDisplayValue(`${ORIGIN}${invoicePath("charges")}`)).toBeVisible();
    expect(mocks.candidates).not.toHaveBeenCalled();
    expect(mocks.list).not.toHaveBeenCalled();
  });
});
