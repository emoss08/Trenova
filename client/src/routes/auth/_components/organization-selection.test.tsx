import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { OrganizationSelection } from "./organization-selection";

const mocks = vi.hoisted(() => ({
  navigate: vi.fn(),
  switchOrganization: vi.fn(),
  currentUser: vi.fn(),
  setUser: vi.fn(),
  fetchManifest: vi.fn(),
  clearPermissions: vi.fn(),
}));

vi.mock("react-router", async (importActual) => {
  const actual = await importActual<typeof import("react-router")>();
  return {
    ...actual,
    useNavigate: () => mocks.navigate,
  };
});

vi.mock("@/services/api", () => ({
  apiService: {
    userService: {
      switchOrganization: mocks.switchOrganization,
      currentUser: mocks.currentUser,
    },
  },
}));

vi.mock("@/stores/auth-store", () => ({
  useAuthStore: (selector: (state: { setUser: typeof mocks.setUser }) => unknown) =>
    selector({ setUser: mocks.setUser }),
}));

vi.mock("@/stores/permission-store", () => ({
  usePermissionStore: (
    selector: (state: {
      fetchManifest: typeof mocks.fetchManifest;
      clearPermissions: typeof mocks.clearPermissions;
    }) => unknown,
  ) =>
    selector({
      fetchManifest: mocks.fetchManifest,
      clearPermissions: mocks.clearPermissions,
    }),
}));

function renderOrganizationSelection() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <OrganizationSelection
          organizations={[
            {
              id: "org_1",
              name: "Alpha Logistics",
              city: "Austin",
              state: "TX",
              logoUrl: null,
              isDefault: true,
              isCurrent: true,
            },
            {
              id: "org_2",
              name: "Bravo Freight",
              city: "Denver",
              state: "CO",
              logoUrl: null,
              isDefault: false,
              isCurrent: false,
            },
          ]}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("OrganizationSelection", () => {
  afterEach(() => {
    cleanup();
  });

  beforeEach(() => {
    Object.values(mocks).forEach((mock) => mock.mockClear());
    mocks.switchOrganization.mockResolvedValue({
      id: "usr_1",
      currentOrganizationId: "org_2",
    });
    mocks.fetchManifest.mockResolvedValue(undefined);
  });

  it("switches the selected organization and fetches permissions", async () => {
    const user = userEvent.setup();
    renderOrganizationSelection();

    expect(screen.getByText("Alpha Logistics")).toBeInTheDocument();
    expect(screen.getByText("Bravo Freight")).toBeInTheDocument();
    expect(screen.getByText("Current")).toBeInTheDocument();

    const bravoButton = screen.getByRole("button", { name: /bravo freight/i });
    await user.click(bravoButton);
    expect(bravoButton).toHaveAttribute("aria-pressed", "true");
    expect(mocks.switchOrganization).not.toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: "Continue" }));

    await waitFor(() =>
      expect(mocks.switchOrganization).toHaveBeenCalledWith({ organizationId: "org_2" }),
    );
    expect(mocks.setUser).toHaveBeenCalledWith(
      expect.objectContaining({ currentOrganizationId: "org_2" }),
    );
    expect(mocks.clearPermissions).toHaveBeenCalled();
    expect(mocks.fetchManifest).toHaveBeenCalledTimes(1);
    expect(mocks.navigate).toHaveBeenCalledWith("/", { replace: true });
  });
});
