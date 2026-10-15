import type { BundleCatalog } from "@/lib/role-api";
import type { AddPermission } from "@trenova/shared/types/role";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { RoleBundlePicker } from "../role-bundle-picker";

const catalog: BundleCatalog = {
  bundles: [
    {
      id: "billing.queue",
      module: "Billing",
      label: "Billing queue",
      description: "Work the billing queue.",
      grants: [
        {
          resource: "billing_queue",
          displayName: "Billing Queue",
          operations: ["read", "update"],
          dataScope: "organization",
        },
        {
          resource: "invoice",
          displayName: "Invoice",
          operations: ["read", "create"],
          dataScope: "organization",
        },
      ],
    },
  ],
  templates: [],
};

const getPermissionBundles = vi.fn<() => Promise<BundleCatalog>>();

vi.mock("@/lib/role-api", () => ({
  getPermissionBundles: () => getPermissionBundles(),
}));

afterEach(() => {
  cleanup();
  getPermissionBundles.mockReset();
});

function renderPicker(permissions: AddPermission[]) {
  const onPermissionsChange = vi.fn<(permissions: AddPermission[]) => void>();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <RoleBundlePicker permissions={permissions} onPermissionsChange={onPermissionsChange} />
    </QueryClientProvider>,
  );
  return onPermissionsChange;
}

describe("RoleBundlePicker", () => {
  it("adds every grant of a bundle the person ticks", async () => {
    getPermissionBundles.mockResolvedValue(catalog);
    const onPermissionsChange = renderPicker([]);

    await userEvent.click(await screen.findByRole("checkbox", { name: /^Billing queue/ }));

    expect(onPermissionsChange).toHaveBeenCalledWith([
      { resource: "billing_queue", operations: ["read", "update"], dataScope: "organization" },
      { resource: "invoice", operations: ["read", "create"], dataScope: "organization" },
    ]);
  });

  it("takes a fully granted bundle away again", async () => {
    getPermissionBundles.mockResolvedValue(catalog);
    const onPermissionsChange = renderPicker([
      { resource: "billing_queue", operations: ["read", "update"], dataScope: "organization" },
      { resource: "invoice", operations: ["read", "create", "export"], dataScope: "organization" },
    ]);

    await userEvent.click(await screen.findByRole("checkbox", { name: /^Billing queue/ }));

    expect(onPermissionsChange).toHaveBeenCalledWith([
      { resource: "invoice", operations: ["export"], dataScope: "organization" },
    ]);
  });

  it("says when a bundle is only partly granted", async () => {
    getPermissionBundles.mockResolvedValue(catalog);
    renderPicker([{ resource: "invoice", operations: ["read"], dataScope: "organization" }]);

    expect(await screen.findByText("Partly granted: Billing Queue, Invoice")).toBeTruthy();
  });
});
