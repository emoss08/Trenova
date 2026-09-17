import { groupCarrierEligibility, isOverridableIntelBlocker } from "@/lib/carrier-eligibility";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import type {
  CarrierAssignmentPayload,
  CarrierAssignmentPayloadInput,
  CarrierEligibility,
} from "@trenova/shared/types/shipment";
import { emptyCarrierAssignmentPayload } from "@trenova/shared/types/shipment";
import { useForm } from "react-hook-form";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  CarrierEligibilityAlerts,
  carrierEligibilityBlocksSubmit,
} from "./carrier-assignment-fields";

const mocks = vi.hoisted(() => ({ allowed: true }));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: mocks.allowed, isLoading: false }),
}));

vi.mock("@/lib/graphql/carrier-intelligence", () => ({
  CARRIER_INTELLIGENCE_KEY: "carrier-intelligence",
  fetchCarrierIntelligence: vi.fn(),
  grantCarrierIntelOverride: vi.fn(),
}));

afterEach(() => {
  mocks.allowed = true;
});

const ELIGIBILITY: CarrierEligibility = {
  blockers: ["Carrier status is Inactive", "Operating authority is revoked"],
  warnings: ["Carrier Cargo insurance policy P-1 expires within 30 days"],
  advisories: [
    "Carrier intelligence has not been refreshed recently",
    "Safety rating is Conditional",
  ],
  findings: [
    {
      code: "record.status",
      source: "Record",
      severity: "Blocker",
      message: "Carrier status is Inactive",
      requiresOverride: false,
    },
    {
      code: "authority.revoked",
      source: "Intelligence",
      severity: "Blocker",
      message: "Operating authority is revoked",
      requiresOverride: false,
    },
    {
      code: "record.insurance.Cargo",
      source: "Record",
      severity: "Warning",
      message: "Carrier Cargo insurance policy P-1 expires within 30 days",
      requiresOverride: true,
    },
    {
      code: "intel.snapshot_stale",
      source: "Intelligence",
      severity: "Advisory",
      message: "Carrier intelligence has not been refreshed recently",
      requiresOverride: false,
    },
    {
      code: "safety.conditional",
      source: "Intelligence",
      severity: "Advisory",
      message: "Safety rating is Conditional",
      requiresOverride: false,
    },
  ],
};

function Harness({ eligibility }: { eligibility: CarrierEligibility }) {
  const form = useForm<CarrierAssignmentPayloadInput, unknown, CarrierAssignmentPayload>({
    defaultValues: { ...emptyCarrierAssignmentPayload },
  });
  return (
    <CarrierEligibilityAlerts
      control={form.control}
      eligibility={eligibility}
      isLoading={false}
      carrierId="car_1"
    />
  );
}

function renderAlerts(eligibility: CarrierEligibility) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <Harness eligibility={eligibility} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

function group(container: HTMLElement, name: string): HTMLElement {
  const node = container.querySelector<HTMLElement>(`[data-eligibility-group="${name}"]`);
  if (!node) {
    throw new Error(`missing eligibility group ${name}`);
  }
  return node;
}

describe("groupCarrierEligibility", () => {
  it("separates record blockers, intelligence blockers, warnings and advisories", () => {
    const grouped = groupCarrierEligibility(ELIGIBILITY);

    expect(grouped.recordBlockers.map((item) => item.code)).toEqual(["record.status"]);
    expect(grouped.intelBlockers.map((item) => item.code)).toEqual(["authority.revoked"]);
    expect(grouped.warnings.map((item) => item.code)).toEqual(["record.insurance.Cargo"]);
    expect(grouped.advisories.map((item) => item.code)).toEqual([
      "intel.snapshot_stale",
      "safety.conditional",
    ]);
  });

  it("falls back to the message lists when a server sends no findings", () => {
    const grouped = groupCarrierEligibility({
      blockers: ["Carrier status is Inactive"],
      warnings: ["Expiring"],
      advisories: ["Stale"],
      findings: [],
    });

    expect(grouped.recordBlockers.map((item) => item.message)).toEqual([
      "Carrier status is Inactive",
    ]);
    expect(grouped.intelBlockers).toEqual([]);
    expect(grouped.warnings.map((item) => item.message)).toEqual(["Expiring"]);
    expect(grouped.advisories.map((item) => item.message)).toEqual(["Stale"]);
  });

  it("never offers an override for the gate's own freshness and outage blockers", () => {
    const grouped = groupCarrierEligibility({
      blockers: ["stale", "revoked"],
      warnings: [],
      advisories: [],
      findings: [
        {
          code: "intel.snapshot_stale",
          source: "Intelligence",
          severity: "Blocker",
          message: "stale",
          requiresOverride: false,
        },
        {
          code: "authority.revoked",
          source: "Intelligence",
          severity: "Blocker",
          message: "revoked",
          requiresOverride: false,
        },
      ],
    });

    expect(grouped.intelBlockers.map(isOverridableIntelBlocker)).toEqual([false, true]);
  });
});

describe("carrierEligibilityBlocksSubmit", () => {
  it("blocks while an intelligence blocker is outstanding, even with warnings overridden", () => {
    expect(
      carrierEligibilityBlocksSubmit(
        {
          blockers: ["Operating authority is revoked"],
          warnings: [],
          advisories: [],
          findings: [],
        },
        true,
      ),
    ).toBe(true);
  });

  it("does not block on advisories alone", () => {
    expect(
      carrierEligibilityBlocksSubmit(
        { blockers: [], warnings: [], advisories: ["Stale"], findings: [] },
        false,
      ),
    ).toBe(false);
  });
});

describe("CarrierEligibilityAlerts", () => {
  it("renders intelligence blockers apart from record blockers and advisories as a note", () => {
    const { container } = renderAlerts(ELIGIBILITY);

    const groups = [...container.querySelectorAll("[data-eligibility-group]")].map((node) =>
      node.getAttribute("data-eligibility-group"),
    );
    expect(groups).toEqual(["record-blockers", "intel-blockers", "warnings", "advisories"]);

    const intel = group(container, "intel-blockers");
    expect(within(intel).getByText("Operating authority is revoked")).toBeInTheDocument();
    expect(within(intel).queryByText("Carrier status is Inactive")).toBeNull();
    expect(within(intel).getByRole("button", { name: "Grant override" })).toBeInTheDocument();

    const advisories = screen.getByRole("note", { name: "Carrier intelligence advisories" });
    expect(within(advisories).getByText("Safety rating is Conditional")).toBeInTheDocument();
    expect(within(advisories).queryByRole("button")).toBeNull();

    const links = screen.getAllByRole("link", { name: /Open carrier intelligence/ });
    expect(links[0]).toHaveAttribute(
      "href",
      "/dispatch/carriers?panelType=edit&panelEntityId=car_1&tab=intelligence",
    );
  });

  it("hides the override action from users who cannot approve overrides", () => {
    mocks.allowed = false;
    renderAlerts(ELIGIBILITY);

    expect(screen.queryByRole("button", { name: "Grant override" })).toBeNull();
  });

  it("does not offer the insurance warning override while any blocker remains", () => {
    renderAlerts(ELIGIBILITY);

    expect(screen.queryByText("Assign anyway")).toBeNull();
  });

  it("offers the insurance warning override when only warnings and advisories remain", () => {
    renderAlerts({
      blockers: [],
      warnings: ELIGIBILITY.warnings,
      advisories: ELIGIBILITY.advisories,
      findings: ELIGIBILITY.findings.filter((finding) => finding.severity !== "Blocker"),
    });

    expect(screen.getByText("Assign anyway")).toBeInTheDocument();
    expect(screen.queryByText("Carrier intelligence blocks this assignment")).toBeNull();
  });
});
