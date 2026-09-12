import { cleanup, render, screen } from "@testing-library/react";
import type { Shipment } from "@trenova/shared/types/shipment";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import type { ReactNode } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ShipmentForm } from "../shipment-form";

vi.mock("../shipment-service-details", () => ({ default: () => null }));
vi.mock("../shipment-billing-details", () => ({ default: () => <p>billing details</p> }));
vi.mock("../additional-charges/shipment-additional-charges", () => ({ default: () => null }));
vi.mock("../shipment-general-information", () => ({ default: () => null }));
vi.mock("../shipment-commodities", () => ({ default: () => null }));
vi.mock("../load-envelope-panel", () => ({ default: () => null }));
vi.mock("../move/shipment-move-details", () => ({ default: () => null }));
vi.mock("../trailer-loading/trailer-loading-drawer", () => ({ default: () => null }));
vi.mock("../recurring-shipment-suggestion", () => ({ RecurringShipmentSuggestion: () => null }));

afterEach(cleanup);

function Harness({ values, children }: { values: Partial<Shipment>; children: ReactNode }) {
  const form = useForm<Shipment>({ defaultValues: values as Shipment });
  return (
    <NuqsTestingAdapter>
      <FormProvider {...form}>{children}</FormProvider>
    </NuqsTestingAdapter>
  );
}

function renderForm(values: Partial<Shipment>) {
  return render(
    <Harness values={values}>
      <ShipmentForm />
    </Harness>,
  );
}

describe("ShipmentForm billing lock", () => {
  it.each(["Approved", "Posted"] as const)(
    "locks the charges of a shipment whose invoice is %s",
    async (billingTransferStatus) => {
      renderForm({ status: "Invoiced", billingTransferStatus });

      expect(await screen.findByText("billing details")).toBeInTheDocument();
      expect(screen.getByText("Locked — shipment has been invoiced")).toBeInTheDocument();
      expect(screen.queryByText("Under Billing Review")).not.toBeInTheDocument();
    },
  );

  it("leaves a shipment billing sent back to operations editable", async () => {
    renderForm({ status: "ReadyToInvoice", billingTransferStatus: "SentBackToOps" });

    expect(await screen.findByText("billing details")).toBeInTheDocument();
    expect(screen.queryByText("Locked — shipment has been invoiced")).not.toBeInTheDocument();
    expect(screen.queryByText("Under Billing Review")).not.toBeInTheDocument();
  });
});
