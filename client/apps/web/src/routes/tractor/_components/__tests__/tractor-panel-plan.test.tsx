import { BASE_ONLY_UNAVAILABLE, seedPlan, unavailableFeatures } from "@/test/plan";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/components/autocomplete-fields", () => ({
  EquipmentManufacturerAutocompleteField: () => null,
  EquipmentTypeAutocompleteField: () => null,
  FleetCodeAutocompleteField: () => null,
  IftaFuelTypeAutocompleteField: () => null,
  UsStateAutocompleteField: () => null,
  WorkerAutocompleteField: () => null,
}));
vi.mock("@/components/custom-fields-section", () => ({ CustomFieldsSection: () => null }));
vi.mock("@/components/documents/documents-tab", () => ({ default: () => null }));
vi.mock("../tractor-inspections-tab", () => ({ default: () => <p>inspections tab</p> }));
vi.mock("@trenova/shared/lib/api", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return { ...actual, api: { post: vi.fn(), put: vi.fn(), get: vi.fn(async () => ({})) } };
});

const { TractorPanel, buildTractorDefaults } = await import("../tractor-panel");

const row = { ...buildTractorDefaults(), id: "trac_77", version: 4, code: "TRC-103" } as never;

function renderPanel(searchParams = "") {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } });
  return render(
    <NuqsTestingAdapter searchParams={searchParams}>
      <QueryClientProvider client={client}>
        <TractorPanel open mode="edit" row={row} onOpenChange={() => {}} />
      </QueryClientProvider>
    </NuqsTestingAdapter>,
  );
}

const GRANTS = { [Resource.Tractor]: Operation.Read | Operation.Update };

afterEach(() => {
  cleanup();
  usePermissionStore.getState().clearPermissions();
});

describe("the tractor's inspections tab", () => {
  it("is not offered on a plan without Samsara", () => {
    seedPlan(GRANTS, BASE_ONLY_UNAVAILABLE);

    renderPanel();

    // The tab strip measures every label it offers, so a label that is nowhere is not offered.
    expect(screen.queryByText("Inspections")).toBeNull();
    expect(screen.getAllByText("Documents").length).toBeGreaterThan(0);
  });

  it("opens a saved link to it on the details rather than on an empty panel", () => {
    seedPlan(GRANTS, BASE_ONLY_UNAVAILABLE);

    renderPanel("?tab=inspections");

    expect(screen.getByPlaceholderText("Code")).toBeInTheDocument();
    expect(screen.queryByText("inspections tab")).toBeNull();
  });

  it("is offered when the plan includes Samsara", async () => {
    seedPlan(GRANTS, unavailableFeatures("ai.desk"));

    renderPanel("?tab=inspections");

    expect(screen.getAllByText("Inspections").length).toBeGreaterThan(0);
    expect(await screen.findByText("inspections tab")).toBeInTheDocument();
  });
});
