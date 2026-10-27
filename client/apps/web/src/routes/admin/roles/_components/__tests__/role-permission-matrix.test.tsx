import type { RoleResourceGuide } from "@/lib/graphql/page-access";
import { stubLayout } from "@/test/layout";
import type { BundleCatalog, ResourceCategory } from "@/lib/role-api";
import type { AddPermission } from "@trenova/shared/types/role";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { RolePermissionMatrix } from "../role-permission-matrix";

// Written from GET /permissions/resources: two categories, a resource whose
// update implies its read, a child resource whose parent's read reaches it,
// and a read-only resource elsewhere.
const categories: ResourceCategory[] = [
  {
    category: "Billing",
    resources: [
      {
        resource: "customer",
        displayName: "Customer",
        description: "Customer management",
        category: "Billing",
        defaultSensitivity: "internal",
        operations: [
          { operation: "read", displayName: "View", description: "View customers", implies: [] },
          {
            operation: "update",
            displayName: "Update",
            description: "Modify customers",
            implies: ["read"],
          },
          {
            operation: "delete",
            displayName: "Delete",
            description: "Delete customers",
            implies: ["read"],
          },
        ],
      },
      {
        resource: "customer_contact",
        displayName: "Customer Contact",
        description: "Contacts of a customer",
        category: "Billing",
        parentResource: "customer",
        defaultSensitivity: "internal",
        operations: [
          { operation: "read", displayName: "View", description: "View contacts", implies: [] },
        ],
      },
    ],
  },
  {
    category: "Dispatch",
    resources: [
      {
        resource: "location_category",
        displayName: "Location Category",
        description: "Categories for locations",
        category: "Dispatch",
        defaultSensitivity: "internal",
        operations: [
          { operation: "read", displayName: "View", description: "View categories", implies: [] },
        ],
      },
    ],
  },
];

let catalog = categories;

const guide = (resource: string): RoleResourceGuide => ({
  resource,
  name: resource,
  granted: `granted:${resource}`,
  withheld: `withheld:${resource}`,
  ceiling: null,
  screens: [],
});

const fetchRoleResourceGuides =
  vi.fn<(resources: readonly string[], maxSensitivity: string) => Promise<RoleResourceGuide[]>>();

vi.mock("@/lib/role-api", () => ({
  getAvailableResources: () => Promise.resolve(catalog),
  getPermissionBundles: () => Promise.resolve({ bundles: [], templates: [] } as BundleCatalog),
}));

vi.mock("@/lib/graphql/page-access", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/page-access")>()),
  fetchRoleResourceGuides: (resources: readonly string[], maxSensitivity: string) =>
    fetchRoleResourceGuides(resources, maxSensitivity),
}));

let restoreLayout = () => {};

beforeEach(() => {
  restoreLayout = stubLayout(320);
});

afterEach(() => {
  restoreLayout();
  cleanup();
  catalog = categories;
  fetchRoleResourceGuides.mockReset();
});

async function renderAdvanced(permissions: AddPermission[]) {
  fetchRoleResourceGuides.mockImplementation(async (resources) => resources.map(guide));
  const onPermissionsChange = vi.fn<(permissions: AddPermission[]) => void>();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <RolePermissionMatrix
          permissions={permissions}
          onPermissionsChange={onPermissionsChange}
          maxSensitivity="restricted"
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  await userEvent.click(await screen.findByRole("tab", { name: "Advanced" }));
  await screen.findByRole("tab", { name: "Advanced", selected: true });
  await screen.findByRole("navigation", { name: "Resource categories" });
  return onPermissionsChange;
}

const row = (resource: string) =>
  document.querySelector<HTMLElement>(`[data-resource="${resource}"]`);

describe("RolePermissionMatrix advanced access", () => {
  it("shows each resource's access level, counting an implied read", async () => {
    await renderAdvanced([
      { resource: "customer", operations: ["update"], dataScope: "organization" },
    ]);

    expect(
      within(row("customer")!).getByRole("combobox", { name: "Access to Customer" }),
    ).toHaveTextContent("Custom · 2 of 3");
    expect(
      within(row("customer_contact")!).getByRole("combobox", {
        name: "Access to Customer Contact",
      }),
    ).toHaveTextContent("No access");
  });

  it("labels full access on a read-only resource as view only", async () => {
    await renderAdvanced([
      { resource: "location_category", operations: ["read"], dataScope: "organization" },
    ]);

    expect(
      within(row("location_category")!).getByRole("combobox", {
        name: "Access to Location Category",
      }),
    ).toHaveTextContent("View only");
  });

  it("asks what a grant means only once one of the category's resources is opened", async () => {
    await renderAdvanced([]);
    expect(fetchRoleResourceGuides).not.toHaveBeenCalled();

    await userEvent.click(screen.getByRole("button", { name: /^Customer Customer management/ }));

    await waitFor(() =>
      expect(fetchRoleResourceGuides).toHaveBeenCalledWith(
        ["customer", "customer_contact"],
        "restricted",
      ),
    );
    expect(fetchRoleResourceGuides).toHaveBeenCalledTimes(1);
    expect(await screen.findByText("withheld:customer")).toBeInTheDocument();
  });

  it("says what the role sees through the parent's read", async () => {
    await renderAdvanced([
      { resource: "customer", operations: ["update"], dataScope: "organization" },
    ]);

    await userEvent.click(screen.getByRole("button", { name: /Customer Contact/ }));

    expect(await screen.findByText("granted:customer_contact")).toBeInTheDocument();
  });

  it("shows an implied operation as included and not removable on its own", async () => {
    await renderAdvanced([
      { resource: "customer", operations: ["update"], dataScope: "organization" },
    ]);

    await userEvent.click(screen.getByRole("button", { name: /^Customer Customer management/ }));

    const operations = within(row("customer")!).getByRole("group", {
      name: "Operations on Customer",
    });
    const view = within(operations).getByRole("checkbox", { name: /View/ });
    expect(view).toHaveAttribute("aria-checked", "true");
    expect(view).toHaveAttribute("data-disabled");
    expect(within(operations).getByText("Included with Update")).toBeInTheDocument();
  });

  it("grants an operation from the open resource", async () => {
    const onPermissionsChange = await renderAdvanced([
      { resource: "customer", operations: ["update"], dataScope: "own" },
    ]);

    await userEvent.click(screen.getByRole("button", { name: /^Customer Customer management/ }));
    await userEvent.click(within(row("customer")!).getByRole("checkbox", { name: /Delete/ }));

    expect(onPermissionsChange).toHaveBeenCalledWith([
      { resource: "customer", operations: ["update", "delete"], dataScope: "own" },
    ]);
  });

  it("sets a resource to view only from its access level", async () => {
    const onPermissionsChange = await renderAdvanced([
      { resource: "customer", operations: ["update", "delete"], dataScope: "own" },
    ]);

    await userEvent.click(
      within(row("customer")!).getByRole("combobox", { name: "Access to Customer" }),
    );
    await userEvent.click(await screen.findByRole("option", { name: "View only" }));

    expect(onPermissionsChange).toHaveBeenCalledWith([
      { resource: "customer", operations: ["read"], dataScope: "own" },
    ]);
  });

  it("lists only granted resources in the granted view", async () => {
    await renderAdvanced([
      { resource: "location_category", operations: ["read"], dataScope: "organization" },
    ]);

    const rail = screen.getByRole("navigation", { name: "Resource categories" });
    await userEvent.click(within(rail).getByRole("button", { name: /^Granted/ }));

    expect(row("location_category")).not.toBeNull();
    expect(row("customer")).toBeNull();
    expect(row("customer_contact")).toBeNull();
  });

  it("narrows to one category from the rail", async () => {
    await renderAdvanced([]);

    const rail = screen.getByRole("navigation", { name: "Resource categories" });
    await userEvent.click(within(rail).getByRole("button", { name: /^Dispatch/ }));

    expect(screen.getByRole("heading", { name: "Dispatch" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Billing" })).toBeNull();
  });

  it("searches resources and offers to clear a search that finds nothing", async () => {
    await renderAdvanced([]);

    const search = screen.getByRole("textbox", { name: "Search resources" });
    await userEvent.type(search, "contact");
    expect(row("customer_contact")).not.toBeNull();
    expect(row("customer")).toBeNull();

    await userEvent.clear(search);
    await userEvent.type(search, "nothing like this");
    expect(screen.getByText("No resources found")).toBeInTheDocument();
  });

  it("keeps the search and open resources across a switch to capabilities and back", async () => {
    await renderAdvanced([]);

    await userEvent.type(screen.getByRole("textbox", { name: "Search resources" }), "customer");
    await userEvent.click(screen.getByRole("button", { name: /^Customer Customer management/ }));
    await userEvent.click(screen.getByRole("tab", { name: "Capabilities" }));
    await userEvent.click(screen.getByRole("tab", { name: "Advanced" }));

    expect(await screen.findByRole("textbox", { name: "Search resources" })).toHaveValue(
      "customer",
    );
    expect(screen.getByRole("button", { name: /^Customer Customer management/ })).toHaveAttribute(
      "aria-expanded",
      "true",
    );
  });

  it("mounts only the resources near the viewport of a large catalog", async () => {
    catalog = [
      {
        category: "Human resources",
        resources: Array.from({ length: 150 }, (_, index) => ({
          resource: `worker_record_${index}`,
          displayName: `Worker Record ${index}`,
          description: "A worker record",
          category: "Human resources",
          defaultSensitivity: "internal",
          operations: [
            { operation: "read", displayName: "View", description: "View", implies: [] },
          ],
        })),
      },
    ];
    await renderAdvanced([]);

    expect(await screen.findByText("Worker Record 0")).toBeInTheDocument();
    const mounted = document.querySelectorAll("[data-resource]").length;
    expect(mounted).toBeGreaterThan(0);
    expect(mounted).toBeLessThan(30);
    expect(screen.queryByText("Worker Record 149")).toBeNull();
  });
});
