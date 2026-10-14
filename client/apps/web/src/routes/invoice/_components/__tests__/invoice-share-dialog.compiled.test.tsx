import { waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { dana, openDialog, pickTeammate, priya, result, share } from "./invoice-share-fixtures";

// Runs under the React Compiler, as the app is built. Each invite resets the form, so
// the note typed for the next teammate must still reach the form afterwards.

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

beforeEach(() => {
  mocks.list.mockResolvedValue([]);
  mocks.candidates.mockResolvedValue([dana, priya]);
});

afterEach(() => {
  vi.clearAllMocks();
});

describe("invoice share dialog under the React Compiler", () => {
  it("sends each teammate's own note when inviting one after another", async () => {
    mocks.share
      .mockResolvedValueOnce(result({ shares: [share({ note: "Check the detention line" })] }))
      .mockResolvedValueOnce(
        result({
          shares: [
            share({ note: "Check the detention line" }),
            share({
              id: "invsh_2",
              sharedWithId: priya.id,
              sharedWith: priya,
              note: "Confirm the lumper receipt",
            }),
          ],
        }),
      );
    const { user, dialog } = await openDialog("?item=inv_1&tab=charges");

    await pickTeammate(user, dialog, "da", "Dana Whitfield");
    await user.type(
      within(dialog).getByRole("textbox", { name: "Note" }),
      "Check the detention line",
    );
    await user.click(within(dialog).getByRole("button", { name: "Invite" }));
    await waitFor(() => expect(mocks.share).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(within(dialog).queryByRole("textbox", { name: "Note" })).toBeNull());

    await pickTeammate(user, dialog, "pr", "Priya Nair");
    const note = within(dialog).getByRole("textbox", { name: "Note" });
    expect(note).toHaveValue("");
    await user.type(note, "Confirm the lumper receipt");
    await user.click(within(dialog).getByRole("button", { name: "Invite" }));

    await waitFor(() => expect(mocks.share).toHaveBeenCalledTimes(2));
    expect(mocks.share).toHaveBeenNthCalledWith(1, "inv_1", {
      userIds: [dana.id],
      note: "Check the detention line",
      tab: "charges",
    });
    expect(mocks.share).toHaveBeenNthCalledWith(2, "inv_1", {
      userIds: [priya.id],
      note: "Confirm the lumper receipt",
      tab: "charges",
    });
  });

  it("sends the note typed for a teammate", async () => {
    mocks.share.mockResolvedValue(result({ shares: [share({ note: "Short-paid by $40" })] }));
    const { user, dialog } = await openDialog();

    await pickTeammate(user, dialog, "da", "Dana Whitfield");
    await user.type(within(dialog).getByRole("textbox", { name: "Note" }), "Short-paid by $40");
    await user.click(within(dialog).getByRole("button", { name: "Invite" }));

    await waitFor(() =>
      expect(mocks.share).toHaveBeenCalledWith("inv_1", {
        userIds: [dana.id],
        note: "Short-paid by $40",
        tab: "overview",
      }),
    );
  });
});
