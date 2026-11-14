import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
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
  autoProposeCoverage: false,
  enforceHosCompliance: false,
  recordServiceFailures: "Never",
  enableCheckCallMonitoring: false,
} as unknown as DispatchControl;

vi.mock("@/lib/queries/dispatch-control", () => ({
  dispatchControl: {
    get: () => ({ queryKey: ["dispatch-control"], queryFn: async () => control }),
  },
}));

vi.mock("@/lib/queries/dispatch-style", () => ({
  dispatchStyle: {
    get: () => ({
      queryKey: ["dispatch-style"],
      queryFn: async () => ({
        enabled: false,
        decisions: 0,
        approvals: 0,
        overrides: 0,
        declines: 0,
        learningSince: null,
        updatedAt: null,
        weights: [],
      }),
    }),
  },
}));

vi.mock("@/lib/queries/dispatch-console", () => ({
  dispatchConsoleQueries: {
    forecastQuality: () => ({
      queryKey: ["dispatch-forecast-quality"],
      queryFn: async () => ({
        status: "InsufficientHistory",
        historyWeeks: 3,
        historyLoads: 41,
        weeksTested: 0,
        loadsTested: 0,
        intervalCoverage: 0,
        meanAbsoluteError: 0,
        naiveMeanAbsoluteError: 0,
        skill: 0,
        fittedAt: 1_793_577_600,
        refitting: false,
      }),
    }),
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

type Outline = { tabs: number; sections: number };

function skeletonOutline(searchParams = ""): Outline {
  const { container } = render(
    <NuqsTestingAdapter searchParams={searchParams}>
      <DispatchControlSkeleton />
    </NuqsTestingAdapter>,
  );
  return {
    tabs: container.querySelectorAll('[data-slot="tab-skeleton"]').length,
    sections: container.querySelectorAll('[data-slot="settings-section-skeleton"]').length,
  };
}

async function formOutline(searchParams = ""): Promise<Outline> {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const { container } = render(
    <NuqsTestingAdapter searchParams={searchParams}>
      <QueryClientProvider client={queryClient}>
        <DispatchControlForm />
      </QueryClientProvider>
    </NuqsTestingAdapter>,
  );
  await screen.findByText("Service failure monitoring");
  return {
    tabs: container.querySelectorAll('[role="tab"]').length,
    sections: container.querySelectorAll(
      '[data-slot="tabs-content"]:not([hidden]) [data-slot="settings-section"]',
    ).length,
  };
}

describe("DispatchControlSkeleton", () => {
  afterEach(() => {
    cleanup();
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  // The fallback has to be the page the organization will see, or the layout jumps when the
  // form lands: planning, its costs, what it learned and DOT compliance are only there for a
  // fleet, and carrier tenders only for a brokerage.
  it("draws every tab, and the open tab's sections, for an organization that runs a fleet and brokers", async () => {
    signIn({ brokerageEnabled: true, assetOperationsEnabled: true });

    const outline = skeletonOutline();
    expect(outline).toEqual({ tabs: 7, sections: 1 });
    cleanup();

    expect(await formOutline()).toEqual(outline);
  });

  it("leaves out the fleet's tabs for a brokerage, as the form does", async () => {
    signIn({ brokerageEnabled: true, assetOperationsEnabled: false });

    const outline = skeletonOutline();
    expect(outline).toEqual({ tabs: 3, sections: 3 });
    cleanup();

    expect(await formOutline()).toEqual(outline);
  });

  it("leaves out the carrier tenders tab for a fleet that does not broker, as the form does", async () => {
    signIn({ brokerageEnabled: false, assetOperationsEnabled: true });

    const outline = skeletonOutline();
    expect(outline).toEqual({ tabs: 6, sections: 1 });
    cleanup();

    expect(await formOutline()).toEqual(outline);
  });

  it("draws the tab the address opens, as the form does", async () => {
    signIn({ brokerageEnabled: true, assetOperationsEnabled: true });

    const outline = skeletonOutline("?tab=exceptions");
    expect(outline).toEqual({ tabs: 7, sections: 4 });
    cleanup();

    expect(await formOutline("?tab=exceptions")).toEqual(outline);
  });

  it("is announced once as loading", () => {
    signIn({ brokerageEnabled: true, assetOperationsEnabled: true });

    render(
      <NuqsTestingAdapter>
        <DispatchControlSkeleton />
      </NuqsTestingAdapter>,
    );

    const root = screen.getByLabelText("Loading settings");
    expect(root).toHaveAttribute("aria-busy", "true");
    expect(root.firstElementChild).toHaveAttribute("aria-hidden", "true");
  });
});
