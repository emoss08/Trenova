import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import DataRetentionForm from "../_components/data-retention-form";
import { DataRetentionSkeleton } from "../skeleton";

/*
A record whose switched purges are all off (0 keeps everything on the EDI and
evaluation-case periods), the state the outline draws.
*/
const allKept = {
  id: "dr_1",
  organizationId: "org_1",
  businessUnitId: "bu_1",
  auditAccessRetentionDays: 90,
  auditOperationalRetentionDays: 730,
  auditFinancialRetentionDays: 2555,
  auditSecurityRetentionDays: 730,
  ediInboundFileRetentionPeriod: 0,
  ediMessageRetentionPeriod: 0,
  driverQualificationRetentionPeriod: 1095,
  agentEvalCaseRetentionPeriod: 0,
  aiFeedbackRetentionPeriod: 730,
  aiAuditRetentionPeriod: 2555,
  aiCorrectionRetentionPeriod: 730,
  version: 1,
  createdAt: 1_780_000_000,
  updatedAt: 1_780_000_000,
};

vi.mock("@/lib/queries/data-retention", () => ({
  dataRetention: {
    get: () => ({ queryKey: ["data-retention"], queryFn: async () => allKept }),
  },
}));

vi.mock("@/hooks/use-optimistic-mutation", () => ({
  useOptimisticMutation: () => ({ mutateAsync: vi.fn() }),
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

type Outline = { tabs: number; sections: number; rows: number };

function skeletonOutline(): Outline {
  const { container } = render(
    <TestRouter>
      <DataRetentionSkeleton />
    </TestRouter>,
  );
  return {
    tabs: container.querySelectorAll('[data-slot="tab-skeleton"]').length,
    sections: container.querySelectorAll('[data-slot="settings-section-skeleton"]').length,
    rows: container.querySelectorAll('[data-slot="field-skeleton"]').length,
  };
}

async function formOutline(): Promise<Outline> {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const { container } = render(
    <TestRouter>
      <QueryClientProvider client={queryClient}>
        <DataRetentionForm now={1_793_016_000} />
      </QueryClientProvider>
    </TestRouter>,
  );
  await screen.findByText("Audit trail");
  return {
    tabs: container.querySelectorAll('[role="tab"]').length,
    sections: container.querySelectorAll('[data-slot="settings-section"]').length,
    rows: container.querySelectorAll('[data-slot="setting-row"]').length,
  };
}

describe("DataRetentionSkeleton", () => {
  afterEach(() => {
    cleanup();
  });

  // The fallback has to be the page that lands, or the layout jumps: four sections of four,
  // two, one and four rows with the switched purges off, and no tab row for the one tab.
  it("draws the form's four sections and their rows, with no tab row", async () => {
    const outline = skeletonOutline();
    expect(outline).toEqual({ tabs: 0, sections: 4, rows: 11 });
    cleanup();

    expect(await formOutline()).toEqual(outline);
  });

  it("is announced once as loading", () => {
    render(
      <TestRouter>
        <DataRetentionSkeleton />
      </TestRouter>,
    );

    const root = screen.getByLabelText("Loading settings");
    expect(root).toHaveAttribute("aria-busy", "true");
    expect(root.firstElementChild).toHaveAttribute("aria-hidden", "true");
  });
});
