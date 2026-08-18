import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { OrganizationCapabilities } from "@trenova/shared/types/organization-capability";
import type { Assignment, CarrierAssignment } from "@trenova/shared/types/shipment";
import type { User } from "@trenova/shared/types/user";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AssignmentDialog } from "../assignment-dialog";

vi.mock("../assignment-hos-feasibility", () => ({
  AssignmentHosFeasibility: () => null,
}));

const hybrid: OrganizationCapabilities = {
  brokerageEnabled: true,
  assetOperationsEnabled: true,
};
const brokerageOnly: OrganizationCapabilities = {
  brokerageEnabled: true,
  assetOperationsEnabled: false,
};

function signIn(capabilities: OrganizationCapabilities) {
  useAuthStore.setState({
    user: {
      currentOrganizationId: "org_01",
      memberships: [
        {
          userId: "usr_01",
          organizationId: "org_01",
          isDefault: true,
          organization: { id: "org_01", name: "Brokerage Co", ...capabilities },
        },
      ],
    } as unknown as User,
    isAuthenticated: true,
  });
}

const existingAssignment = {
  id: "asn_1",
  status: "New",
  shipmentMoveId: "mov_1",
  primaryWorkerId: "wrk_1",
  tractorId: "trc_1",
  trailerId: null,
  secondaryWorkerId: null,
} as unknown as Assignment;

function renderDialog(options: { existingAssignment?: Assignment | null } = {}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <AssignmentDialog
          open
          onOpenChange={vi.fn()}
          moveId="mov_1"
          existingAssignment={options.existingAssignment ?? null}
          existingCarrierAssignment={null as CarrierAssignment | null}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("AssignmentDialog coverage modes", () => {
  afterEach(() => {
    cleanup();
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  it("offers both coverage modes to an organization that runs drivers and brokers", () => {
    signIn(hybrid);

    renderDialog();

    expect(screen.getByRole("tab", { name: "Driver" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Carrier" })).toBeInTheDocument();
  });

  it("drops the driver tab for an organization that employs no drivers", () => {
    signIn(brokerageOnly);

    renderDialog();

    expect(screen.queryByRole("tab", { name: "Driver" })).toBeNull();
    // One remaining mode is not a choice, so the whole control goes with it.
    expect(screen.queryByRole("tablist", { name: "Coverage type" })).toBeNull();
  });

  it("opens straight into carrier coverage when the driver mode is unavailable", () => {
    signIn(brokerageOnly);

    renderDialog();

    expect(screen.getByText("Assign Move to Carrier")).toBeInTheDocument();
  });

  it("still opens in driver mode for an organization that employs drivers", () => {
    signIn(hybrid);

    renderDialog();

    expect(screen.getByText("Assign Move")).toBeInTheDocument();
  });

  /**
   * The server refuses to create or change a driver assignment for an
   * organization without asset operations. Reaching that refusal through a
   * submit would surface a raw API error, so the dialog says it up front in the
   * same panel shape it already uses for the driver/carrier cross-refusals.
   */
  it("pre-empts the server refusal instead of offering a reassignment that will fail", () => {
    signIn(brokerageOnly);

    renderDialog({ existingAssignment });

    expect(screen.getByText("Driver assignment is not enabled")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Reassign" })).toBeNull();
    expect(screen.queryByLabelText("Tractor")).toBeNull();
  });

  it("says how the move can still be freed up", () => {
    signIn(brokerageOnly);

    renderDialog({ existingAssignment });

    expect(screen.getByText(/Unassign the driver/)).toBeInTheDocument();
  });

  it("leaves reassignment alone for an organization that employs drivers", () => {
    signIn(hybrid);

    renderDialog({ existingAssignment });

    expect(screen.queryByText("Driver assignment is not enabled")).toBeNull();
    expect(screen.getByText("Reassign Move")).toBeInTheDocument();
  });
});
