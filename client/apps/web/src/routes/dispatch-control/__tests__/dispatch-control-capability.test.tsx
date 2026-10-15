import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { OrganizationCapabilities } from "@trenova/shared/types/organization-capability";
import type { User } from "@trenova/shared/types/user";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { DispatchControl } from "@/types/dispatch-control";
import DispatchControlForm from "../_components/dispatch-control-form";

const control: DispatchControl = {
  id: "dc_1",
  version: 0,
  organizationId: "org_01",
  businessUnitId: "bu_01",
  enableAutoAssignment: true,
  autoAssignmentStrategy: "Proximity",
  scoringWeights: {},
  enforceWorkerAssign: true,
  enforceTrailerContinuity: true,
  enforceHosCompliance: true,
  enforceWorkerPtaRestrictions: true,
  enforceWorkerTractorFleetContinuity: true,
  enforceDriverQualificationCompliance: true,
  enforceMedicalCertCompliance: true,
  enforceHazmatCompliance: true,
  enforceDrugAndAlcoholCompliance: true,
  complianceEnforcementLevel: "Warning",
  recordServiceFailures: "Never",
  tenderCounterOffersEnabled: true,
  tenderCounterMarginFloorPercent: "10",
  tenderAutoAcceptCounters: false,
  tenderCounterDecisionMinutes: 60,
  tenderExhaustedAction: "Notify",
  tenderEscalationRateIncreasePercent: "0",
  tenderEscalationOfferTtlMinutes: 60,
  tenderMaxEscalations: 1,
} as unknown as DispatchControl;

const served = vi.hoisted(() => ({ control: null as unknown }));
served.control = control;

vi.mock("@/lib/queries/dispatch-control", () => ({
  dispatchControl: {
    get: () => ({ queryKey: ["dispatch-control"], queryFn: async () => served.control }),
  },
}));

vi.mock("@/hooks/use-optimistic-mutation", () => ({
  useOptimisticMutation: () => ({ mutateAsync: vi.fn() }),
}));

/** Every driver-compliance control the plan calls for hiding. */
const DRIVER_COMPLIANCE_LABELS = [
  "Enable automated assignment",
  "Assignment optimization strategy",
  "Candidate scoring weights",
  "Enable DOT compliance enforcement",
  "Medical certification validation",
  "Driver qualification verification",
  "Drug and alcohol testing compliance",
  "Require worker assignment",
  "Require trailer continuity",
  "Enforce worker PTA restrictions",
  "Enforce worker tractor fleet continuity",
];

/** Neutral settings that describe the freight, not the driver moving it. */
const NEUTRAL_LABELS = ["Record Service failures"];

const hybrid: OrganizationCapabilities = {
  brokerageEnabled: true,
  assetOperationsEnabled: true,
};
const brokerageOnly: OrganizationCapabilities = {
  brokerageEnabled: true,
  assetOperationsEnabled: false,
};
const assetOnly: OrganizationCapabilities = {
  brokerageEnabled: false,
  assetOperationsEnabled: true,
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

function renderForm() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <QueryClientProvider client={queryClient}>
      <DispatchControlForm />
    </QueryClientProvider>,
  );
}

describe("dispatch control field visibility", () => {
  afterEach(() => {
    cleanup();
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  it("shows every driver-compliance control to an organization that employs drivers", async () => {
    signIn(hybrid);

    renderForm();

    for (const label of DRIVER_COMPLIANCE_LABELS) {
      expect(await screen.findByText(label), label).toBeInTheDocument();
    }
  });

  it("hides every driver-compliance control from a brokerage", async () => {
    signIn(brokerageOnly);

    renderForm();

    await screen.findByText("Record Service failures");

    for (const label of DRIVER_COMPLIANCE_LABELS) {
      expect(screen.queryByText(label), label).toBeNull();
    }
  });

  it("keeps the page and its neutral settings reachable for a brokerage", async () => {
    signIn(brokerageOnly);

    renderForm();

    for (const label of NEUTRAL_LABELS) {
      expect(await screen.findByText(label), label).toBeInTheDocument();
    }
    expect(screen.getByText("Service failure monitoring")).toBeInTheDocument();
  });
});

describe("dispatch control carrier tender settings", () => {
  afterEach(() => {
    served.control = control;
    cleanup();
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  it("shows the counter-offer settings to a brokerage", async () => {
    signIn(brokerageOnly);

    renderForm();

    expect(await screen.findByText("Carrier tenders")).toBeInTheDocument();
    expect(screen.getByText("Take counter-offers")).toBeInTheDocument();
    expect(screen.getByText("Margin floor")).toBeInTheDocument();
    expect(screen.getByText("Dispatch has to answer a counter within")).toBeInTheDocument();
  });

  it("withholds tender settings from an organization that does not broker freight", async () => {
    signIn(assetOnly);

    renderForm();

    await screen.findByText("Record Service failures");
    expect(screen.queryByText("Carrier tenders")).toBeNull();
  });

  it("hides the counter settings once counter-offers are turned off", async () => {
    signIn(brokerageOnly);
    const user = userEvent.setup();

    renderForm();

    await user.click(await screen.findByRole("switch", { name: /Take counter-offers/ }));

    expect(screen.queryByText("Margin floor")).toBeNull();
    expect(screen.queryByText("Dispatch has to answer a counter within")).toBeNull();
  });

  it("asks for escalation terms only when an exhausted tender escalates", async () => {
    signIn(brokerageOnly);

    renderForm();

    await screen.findByText("When every carrier passes");
    expect(screen.queryByText("Raise the rates by up to")).toBeNull();
    expect(screen.queryByText("Escalated offers stay open for")).toBeNull();
  });

  it("asks how much to raise rates, for how long and how often when the tender escalates", async () => {
    served.control = { ...control, tenderExhaustedAction: "Rewaterfall" };
    signIn(brokerageOnly);

    renderForm();

    expect(await screen.findByText("Raise the rates by up to")).toBeInTheDocument();
    expect(screen.getByText("Escalated offers stay open for")).toBeInTheDocument();
    expect(screen.getAllByText("Escalate at most").length).toBeGreaterThan(0);
  });
});
