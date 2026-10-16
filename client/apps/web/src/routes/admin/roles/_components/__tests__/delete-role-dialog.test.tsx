import type { RoleImpact } from "@trenova/shared/types/role";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DeleteRoleDialog } from "../delete-role-dialog";

const getRoleImpact = vi.fn<(id: string) => Promise<RoleImpact[]>>();
const deleteRole = vi.fn<(id: string, options?: { force?: boolean }) => Promise<void>>();

vi.mock("@/lib/role-api", () => ({
  getRoleImpact: (id: string) => getRoleImpact(id),
  deleteRole: (id: string, options?: { force?: boolean }) => deleteRole(id, options),
}));

vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

afterEach(() => {
  cleanup();
  getRoleImpact.mockReset();
  deleteRole.mockReset();
});

function renderDialog(onDeleted = vi.fn()) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <DeleteRoleDialog
        open
        onOpenChange={vi.fn()}
        roleId="rol_dispatch"
        roleName="Dispatcher"
        onDeleted={onDeleted}
      />
    </QueryClientProvider>,
  );
  return onDeleted;
}

describe("DeleteRoleDialog", () => {
  it("deletes an unheld role without taking it from anyone", async () => {
    getRoleImpact.mockResolvedValue([]);
    deleteRole.mockResolvedValue(undefined);
    const onDeleted = renderDialog();

    await userEvent.click(await screen.findByRole("button", { name: "Delete role" }));

    await waitFor(() => expect(deleteRole).toHaveBeenCalledWith("rol_dispatch", { force: false }));
    expect(onDeleted).toHaveBeenCalledTimes(1);
  });

  it("says who holds the role and takes it from them when confirmed", async () => {
    getRoleImpact.mockResolvedValue([
      { userId: "usr_a", userName: "Ada", email: "ada@example.com" },
      { userId: "usr_b", userName: "Bo", email: "bo@example.com" },
    ]);
    deleteRole.mockResolvedValue(undefined);
    renderDialog();

    expect(await screen.findByText(/2 people hold this role/)).toBeTruthy();
    await userEvent.click(screen.getByRole("button", { name: "Delete and unassign" }));

    await waitFor(() => expect(deleteRole).toHaveBeenCalledWith("rol_dispatch", { force: true }));
  });

  it("keeps the dialog open when the server refuses", async () => {
    getRoleImpact.mockResolvedValue([]);
    deleteRole.mockRejectedValue(new Error("forbidden"));
    const onDeleted = renderDialog();

    await userEvent.click(await screen.findByRole("button", { name: "Delete role" }));

    await waitFor(() => expect(deleteRole).toHaveBeenCalledTimes(1));
    expect(onDeleted).not.toHaveBeenCalled();
  });
});
