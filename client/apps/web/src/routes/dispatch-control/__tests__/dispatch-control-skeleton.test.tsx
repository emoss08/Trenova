import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { OrganizationCapabilities } from "@trenova/shared/types/organization-capability";
import type { User } from "@trenova/shared/types/user";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { DispatchControl } from "@/types/dispatch-control";
import DispatchControlForm from "../_components/dispatch-control-form";
import { DispatchControlSkeleton } from "../skeleton";

const control = {
  id: "dc_1",
  version: 0,
  organizationId: "org_01",
  businessUnitId: "bu_01",
  enableAutoAssignment: false,
  autoAssignmentStrategy: "Proximity",
  scoringWeights: {},
  enforceHosCompliance: false,
  recordServiceFailures: "Never",
  enableCheckCallMonitoring: false,
} as unknown as DispatchControl;

vi.mock("@/lib/queries/dispatch-control", () => ({
  dispatchControl: {
    get: () => ({ queryKey: ["dispatch-control"], queryFn: async () => control }),
  },
}));

vi.mock("@/hooks/use-optimistic-mutation", () => ({
  useOptimisticMutation: () => ({ mutateAsync: vi.fn() }),
}));

function signIn(capabilities: OrganizationCapabilities) {
  useAuthStore.setState({
    user: {
      currentOrganizationId: "org_01",
      memberships: [
        {
          userId: "usr_01",
          organizationId: "org_01",
          isDefault: true,
          organization: { id: "org_01", name: "Carrier Co", ...capabilities },
        },
      ],
    } as unknown as User,
    isAuthenticated: true,
  });
}

function cardCount(root: ParentNode) {
  return root.querySelectorAll('[data-slot="card"]').length;
}

async function loadedFormCardCount() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const { container } = render(
    <QueryClientProvider client={queryClient}>
      <DispatchControlForm />
    </QueryClientProvider>,
  );
  await screen.findByText("Service failure monitoring");
  return cardCount(container);
}

describe("DispatchControlSkeleton", () => {
  afterEach(() => {
    cleanup();
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  // The fallback has to be the page the organization will see, or the layout jumps when the
  // form lands: assignment, scoring and DOT compliance are only there for a fleet, and carrier
  // tenders only for a brokerage.
  it("draws every card the form shows an organization that runs a fleet and brokers", async () => {
    signIn({ brokerageEnabled: true, assetOperationsEnabled: true });

    const { container: skeleton } = render(<DispatchControlSkeleton />);
    expect(cardCount(skeleton)).toBe(8);
    cleanup();

    expect(await loadedFormCardCount()).toBe(8);
  });

  it("leaves out the fleet's cards for a brokerage, as the form does", async () => {
    signIn({ brokerageEnabled: true, assetOperationsEnabled: false });

    const { container: skeleton } = render(<DispatchControlSkeleton />);
    expect(cardCount(skeleton)).toBe(5);
    cleanup();

    expect(await loadedFormCardCount()).toBe(5);
  });

  it("leaves out the carrier tenders card for a fleet that does not broker, as the form does", async () => {
    signIn({ brokerageEnabled: false, assetOperationsEnabled: true });

    const { container: skeleton } = render(<DispatchControlSkeleton />);
    expect(cardCount(skeleton)).toBe(7);
    cleanup();

    expect(await loadedFormCardCount()).toBe(7);
  });

  it("is announced once as loading", () => {
    signIn({ brokerageEnabled: true, assetOperationsEnabled: true });

    render(<DispatchControlSkeleton />);

    const root = screen.getByLabelText("Loading settings");
    expect(root).toHaveAttribute("aria-busy", "true");
    expect(root.firstElementChild).toHaveAttribute("aria-hidden", "true");
  });
});
