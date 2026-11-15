import { TestRouter } from "@trenova/shared/testing/test-router";
import { BASE_ONLY_UNAVAILABLE, seedPlan, unavailableFeatures } from "@/test/plan";
import { cleanup, render, screen } from "@testing-library/react";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CarrierTabbedForm } from "../carrier-tabbed-form";

vi.mock("../carrier-form", () => ({ CarrierForm: () => <p>identity form</p> }));
vi.mock("../carrier-compliance-form", () => ({ CarrierComplianceForm: () => null }));
vi.mock("../carrier-contacts-form", () => ({ CarrierContactsForm: () => null }));
vi.mock("../carrier-remittance-form", () => ({ CarrierRemittanceForm: () => null }));
vi.mock("../carrier-tax-form", () => ({ CarrierTaxForm: () => null }));
vi.mock("../intelligence/carrier-intelligence-tab", () => ({
  CarrierIntelligenceTab: () => <p>intelligence tab</p>,
}));
vi.mock("@/components/record-memories/record-memories", () => ({
  RecordMemories: () => null,
  useRecordMemoriesAvailable: () => false,
}));

function renderForm(searchParams = "") {
  return render(
    <TestRouter search={searchParams}>
      <CarrierTabbedForm carrierId="car_01" />
    </TestRouter>,
  );
}

afterEach(() => {
  cleanup();
  usePermissionStore.getState().clearPermissions();
});

describe("the carrier's intelligence tab", () => {
  it("is not offered on a plan without Carrier & Brokerage", () => {
    seedPlan({}, BASE_ONLY_UNAVAILABLE);

    renderForm();

    expect(screen.queryByRole("tab", { name: "Intelligence" })).not.toBeInTheDocument();
    expect(screen.queryByText(/not on your plan/)).not.toBeInTheDocument();
  });

  it("opens a saved link to it on the identity instead of on nothing", () => {
    seedPlan({}, BASE_ONLY_UNAVAILABLE);

    renderForm("?tab=intelligence");

    expect(screen.getByText("identity form")).toBeInTheDocument();
    expect(screen.queryByText("intelligence tab")).not.toBeInTheDocument();
  });

  it("is offered on a plan that includes Carrier & Brokerage", () => {
    seedPlan({}, unavailableFeatures("ai.desk"));

    renderForm("?tab=intelligence");

    expect(screen.getByRole("tab", { name: "Intelligence" })).toBeInTheDocument();
    expect(screen.getByText("intelligence tab")).toBeInTheDocument();
  });
});
