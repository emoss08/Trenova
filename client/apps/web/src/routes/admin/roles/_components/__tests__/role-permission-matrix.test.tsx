import type { RoleResourceGuide } from "@/lib/graphql/page-access";
import type { BundleCatalog, ResourceCategory } from "@/lib/role-api";
import type { AddPermission } from "@trenova/shared/types/role";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { RolePermissionMatrix } from "../role-permission-matrix";

// Written from GET /permissions/resources: a category holding a resource
// and a child resource whose parent's read reaches it.
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
];

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
  getAvailableResources: () => Promise.resolve(categories),
  getPermissionBundles: () => Promise.resolve({ bundles: [], templates: [] } as BundleCatalog),
}));

vi.mock("@/lib/graphql/page-access", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/page-access")>()),
  fetchRoleResourceGuides: (resources: readonly string[], maxSensitivity: string) =>
    fetchRoleResourceGuides(resources, maxSensitivity),
}));

afterEach(() => {
  cleanup();
  fetchRoleResourceGuides.mockReset();
});

async function renderMatrix(permissions: AddPermission[]) {
  fetchRoleResourceGuides.mockImplementation(async (resources) => resources.map(guide));
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <RolePermissionMatrix
          permissions={permissions}
          onPermissionsChange={() => undefined}
          maxSensitivity="restricted"
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  await userEvent.click(await screen.findByRole("tab", { name: "Advanced" }));
  await userEvent.click(await screen.findByText("Billing"));
}

describe("RolePermissionMatrix sentences", () => {
  it("asks what each resource's grant means only once its category is open", async () => {
    await renderMatrix([]);

    await waitFor(() =>
      expect(fetchRoleResourceGuides).toHaveBeenCalledWith(
        ["customer", "customer_contact"],
        "restricted",
      ),
    );
    expect(await screen.findByText("withheld:customer")).toBeInTheDocument();
    expect(screen.getByText("withheld:customer_contact")).toBeInTheDocument();
  });

  it("says what the role sees with read, granted outright, implied or through the parent", async () => {
    await renderMatrix([
      { resource: "customer", operations: ["update"], dataScope: "organization" },
    ]);

    expect(await screen.findByText("granted:customer")).toBeInTheDocument();
    expect(screen.getByText("granted:customer_contact")).toBeInTheDocument();
    expect(screen.queryByText("withheld:customer")).toBeNull();
  });
});
