import type { BundleCatalog, ResourceCategory } from "@/lib/role-api";
import type { AddPermission } from "@trenova/shared/types/role";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { RoleBundlePicker } from "../role-bundle-picker";

// Written from GET /permissions/bundles: two modules, one holding a bundle
// that sits inside another.
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
        {
          resource: "customer",
          displayName: "Customer",
          operations: ["read"],
          dataScope: "organization",
        },
        {
          resource: "document",
          displayName: "Document",
          operations: ["read"],
          dataScope: "organization",
        },
      ],
    },
    {
      id: "dispatch.board",
      module: "Dispatch",
      label: "Dispatch board",
      description: "Work the board.",
      grants: [
        {
          resource: "assignment",
          displayName: "Assignment",
          operations: ["read", "assign"],
          dataScope: "organization",
        },
      ],
    },
    {
      id: "dispatch.watch",
      module: "Dispatch",
      label: "Watch the board",
      description: "See the board.",
      grants: [
        {
          resource: "assignment",
          displayName: "Assignment",
          operations: ["read"],
          dataScope: "organization",
        },
      ],
    },
  ],
  templates: [],
};

// Written from GET /permissions/resources: each resource names its own
// operations, so an update can read as "Work".
const resources: ResourceCategory[] = [
  {
    category: "Billing",
    resources: [
      {
        resource: "billing_queue",
        displayName: "Billing Queue",
        description: "The billing queue",
        category: "Billing",
        defaultSensitivity: "internal",
        operations: [
          { operation: "read", displayName: "View", description: "View", implies: [] },
          { operation: "update", displayName: "Work", description: "Work", implies: ["read"] },
        ],
      },
      {
        resource: "invoice",
        displayName: "Invoice",
        description: "Invoices",
        category: "Billing",
        defaultSensitivity: "restricted",
        operations: [
          { operation: "read", displayName: "View", description: "View", implies: [] },
          { operation: "create", displayName: "Draft", description: "Draft", implies: ["read"] },
        ],
      },
    ],
  },
];

const getPermissionBundles = vi.fn<() => Promise<BundleCatalog>>();
const getAvailableResources = vi.fn<() => Promise<ResourceCategory[]>>();

vi.mock("@/lib/role-api", () => ({
  getPermissionBundles: () => getPermissionBundles(),
  getAvailableResources: () => getAvailableResources(),
}));

afterEach(() => {
  cleanup();
  getPermissionBundles.mockReset();
  getAvailableResources.mockReset();
});

function renderPicker(permissions: AddPermission[]) {
  getPermissionBundles.mockResolvedValue(catalog);
  getAvailableResources.mockResolvedValue(resources);
  const onPermissionsChange = vi.fn<(permissions: AddPermission[]) => void>();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <RoleBundlePicker permissions={permissions} onPermissionsChange={onPermissionsChange} />
    </QueryClientProvider>,
  );
  return onPermissionsChange;
}

const row = (id: string) => document.querySelector<HTMLElement>(`[data-bundle="${id}"]`);

const billingGrants = [
  { resource: "billing_queue", operations: ["read", "update"], dataScope: "organization" },
  { resource: "invoice", operations: ["read", "create"], dataScope: "organization" },
  { resource: "customer", operations: ["read"], dataScope: "organization" },
  { resource: "document", operations: ["read"], dataScope: "organization" },
] satisfies AddPermission[];

describe("RoleBundlePicker", () => {
  it("adds every grant of a capability switched on", async () => {
    const onPermissionsChange = renderPicker([]);

    await userEvent.click(await screen.findByRole("switch", { name: "Billing queue" }));

    expect(onPermissionsChange).toHaveBeenCalledWith(billingGrants);
  });

  it("takes a fully granted capability away again", async () => {
    const onPermissionsChange = renderPicker([
      ...billingGrants.slice(0, 1),
      { resource: "invoice", operations: ["read", "create", "export"], dataScope: "organization" },
      ...billingGrants.slice(2),
    ]);

    const toggle = await screen.findByRole("switch", { name: "Billing queue" });
    expect(toggle).toHaveAttribute("aria-checked", "true");
    await userEvent.click(toggle);

    expect(onPermissionsChange).toHaveBeenCalledWith([
      { resource: "invoice", operations: ["export"], dataScope: "organization" },
    ]);
  });

  it("marks a partly granted capability and grants the rest when switched on", async () => {
    const onPermissionsChange = renderPicker([
      { resource: "invoice", operations: ["read"], dataScope: "organization" },
    ]);

    const toggle = await screen.findByRole("switch", { name: "Billing queue" });
    expect(toggle).toHaveAttribute("aria-checked", "false");
    expect(within(row("billing.queue")!).getByText("Partly granted")).toBeInTheDocument();
    expect(within(row("dispatch.board")!).queryByText("Partly granted")).toBeNull();

    await userEvent.click(toggle);

    expect(onPermissionsChange).toHaveBeenCalledWith([
      { resource: "invoice", operations: ["read", "create"], dataScope: "organization" },
      billingGrants[0],
      billingGrants[2],
      billingGrants[3],
    ]);
  });

  it("groups capabilities by module and counts those granted", async () => {
    renderPicker([{ resource: "assignment", operations: ["read"], dataScope: "organization" }]);

    const dispatch = await screen.findByRole("region", { name: "Dispatch" });
    expect(within(dispatch).getByText("1 of 2 granted")).toBeInTheDocument();
    expect(
      within(screen.getByRole("region", { name: "Billing" })).getByText("0 of 1 granted"),
    ).toBeInTheDocument();
    expect(screen.getByText("1 of 3 granted")).toBeInTheDocument();
  });

  it("turns a whole module off at once, a capability inside another included", async () => {
    const onPermissionsChange = renderPicker([
      { resource: "assignment", operations: ["read", "assign"], dataScope: "organization" },
    ]);

    const dispatch = await screen.findByRole("region", { name: "Dispatch" });
    await userEvent.click(within(dispatch).getByRole("button", { name: "Turn off all" }));

    expect(onPermissionsChange).toHaveBeenCalledWith([]);
  });

  it("turns a whole module on at once", async () => {
    const onPermissionsChange = renderPicker([]);

    const dispatch = await screen.findByRole("region", { name: "Dispatch" });
    await userEvent.click(within(dispatch).getByRole("button", { name: "Turn on all" }));

    expect(onPermissionsChange).toHaveBeenCalledWith([
      { resource: "assignment", operations: ["read", "assign"], dataScope: "organization" },
    ]);
  });

  it("lists what a capability grants, by each resource's own operation names, once opened", async () => {
    renderPicker([billingGrants[0]]);

    await screen.findByRole("switch", { name: "Billing queue" });
    expect(getAvailableResources).not.toHaveBeenCalled();

    const disclosure = within(row("billing.queue")!).getByRole("button", { name: /4 resources/ });
    await userEvent.click(disclosure);
    expect(disclosure).toHaveAttribute("aria-expanded", "true");

    const grants = screen.getByRole("list", { name: "What Billing queue grants" });
    const [queue, invoice] = within(grants).getAllByRole("listitem");
    expect(await within(queue).findByText("View, Work")).toBeInTheDocument();
    expect(within(queue).getByText("Granted")).toBeInTheDocument();
    expect(within(invoice).getByText("View, Draft")).toBeInTheDocument();
    expect(within(invoice).getByText("Not granted")).toBeInTheDocument();
    expect(getAvailableResources).toHaveBeenCalledTimes(1);
  });

  it("finds a capability by a resource it grants", async () => {
    renderPicker([]);

    await userEvent.type(
      await screen.findByRole("textbox", { name: "Search capabilities" }),
      "assignment",
    );

    expect(row("dispatch.board")).not.toBeNull();
    expect(row("billing.queue")).toBeNull();
  });

  it("offers to clear a search that finds nothing", async () => {
    renderPicker([]);

    const search = await screen.findByRole("textbox", { name: "Search capabilities" });
    await userEvent.type(search, "nothing like this");
    expect(screen.getByText("No capabilities found")).toBeInTheDocument();

    const empty = screen.getByText("No capabilities found").parentElement!;
    await userEvent.click(within(empty).getByRole("button", { name: "Clear search" }));
    expect(search).toHaveValue("");
    expect(row("billing.queue")).not.toBeNull();
  });
});
