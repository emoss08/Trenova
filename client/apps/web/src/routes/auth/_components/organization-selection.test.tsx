import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { OrganizationSelection } from "./organization-selection";

const mocks = vi.hoisted(() => ({
  switchOrganization: vi.fn(),
  currentUser: vi.fn(),
  setUser: vi.fn(),
  clearPermissions: vi.fn(),
  onSelected: vi.fn(),
  onBack: vi.fn(),
}));

vi.mock("@/services/api", () => ({
  apiService: {
    userService: {
      switchOrganization: mocks.switchOrganization,
      currentUser: mocks.currentUser,
    },
  },
}));

vi.mock("@trenova/shared/stores/auth-store", () => ({
  useAuthStore: (selector: (state: { setUser: typeof mocks.setUser }) => unknown) =>
    selector({ setUser: mocks.setUser }),
}));

vi.mock("@trenova/shared/stores/permission-store", () => ({
  usePermissionStore: (
    selector: (state: { clearPermissions: typeof mocks.clearPermissions }) => unknown,
  ) => selector({ clearPermissions: mocks.clearPermissions }),
}));

const organizations = [
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
];

function renderOrganizationSelection() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <OrganizationSelection
          organizations={organizations}
          stepLabel="02 / 03"
          onBack={mocks.onBack}
          onSelected={mocks.onSelected}
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
    mocks.currentUser.mockResolvedValue({ id: "usr_1", currentOrganizationId: "org_1" });
    mocks.onSelected.mockResolvedValue(undefined);
  });

  it("switches the selected organization and hands control back to the flow", async () => {
    const user = userEvent.setup();
    renderOrganizationSelection();

    expect(screen.getByText("02 / 03")).toBeInTheDocument();
    expect(screen.getByText("2 available")).toBeInTheDocument();
    expect(screen.getByText("Current")).toBeInTheDocument();

    const bravo = screen.getByRole("radio", { name: /bravo freight/i });
    await user.click(bravo);
    expect(bravo).toHaveAttribute("aria-checked", "true");
    expect(mocks.switchOrganization).not.toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: "Continue" }));

    await waitFor(() =>
      expect(mocks.switchOrganization).toHaveBeenCalledWith({ organizationId: "org_2" }),
    );
    expect(mocks.setUser).toHaveBeenCalledWith(
      expect.objectContaining({ currentOrganizationId: "org_2" }),
    );
    expect(mocks.clearPermissions).toHaveBeenCalled();
    expect(mocks.onSelected).toHaveBeenCalledWith(
      expect.objectContaining({ id: "org_2", name: "Bravo Freight" }),
    );
  });

  it("refreshes the user rather than switching when the current organization is kept", async () => {
    const user = userEvent.setup();
    renderOrganizationSelection();

    await user.click(screen.getByRole("button", { name: "Continue" }));

    await waitFor(() => expect(mocks.currentUser).toHaveBeenCalledTimes(1));
    expect(mocks.switchOrganization).not.toHaveBeenCalled();
    expect(mocks.onSelected).toHaveBeenCalledWith(expect.objectContaining({ id: "org_1" }));
  });

  it("shows a three-letter code for each organization", () => {
    renderOrganizationSelection();

    expect(screen.getByText("ALL")).toBeInTheDocument();
    expect(screen.getByText("BRF")).toBeInTheDocument();
  });
});
