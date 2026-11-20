import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, within } from "@testing-library/react";
import { detentionPolicySchema, type DetentionPolicy } from "@trenova/shared/types/detention";
import { TestRouter } from "@trenova/shared/testing/test-router";
import { domMax, LazyMotion, MotionGlobalConfig } from "motion/react";
import type { ReactNode } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { DetentionPolicyForm } from "../_components/detention-policy-form";

// The pickers read their options from the server; a plain input registered under the same
// name stands in, so a locked scope shows as a control inside a disabled fieldset.
vi.mock("@/components/autocomplete-fields", async () => {
  const { useController } = await import("react-hook-form");
  function PickerStandIn({
    control,
    name,
    label,
  }: {
    control: never;
    name: string;
    label: string;
  }) {
    const { field } = useController({ control, name });
    return (
      <input
        aria-label={label}
        value={Array.isArray(field.value) ? field.value.join(",") : ((field.value as string) ?? "")}
        onChange={(event) => field.onChange(event.target.value || null)}
      />
    );
  }
  return {
    AccessorialChargeAutocompleteField: PickerStandIn,
    CommodityMultiSelectField: PickerStandIn,
    CustomerAutocompleteField: PickerStandIn,
    LocationAutocompleteField: PickerStandIn,
    ServiceTypeMultiSelectField: PickerStandIn,
    ShipmentTypeMultiSelectField: PickerStandIn,
  };
});

function policy(overrides: Partial<DetentionPolicy>): DetentionPolicy {
  return detentionPolicySchema.parse({
    name: "Standard detention terms",
    code: "DET-STD",
    status: "Active",
    accessorialChargeId: "acc_01",
    tiers: [],
    ...overrides,
  });
}

// jsdom does not carry a fieldset's disabled state to its controls' :disabled, as a browser
// does, so the test reads the fieldset the control sits in.
function lockedBy(element: HTMLElement): boolean {
  return element.closest("fieldset")?.disabled === true;
}

function Harness({ values, children }: { values: DetentionPolicy; children: ReactNode }) {
  const form = useForm<DetentionPolicy>({ defaultValues: values });
  return <FormProvider {...form}>{children}</FormProvider>;
}

function renderForm(values: DetentionPolicy) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <TestRouter>
      <LazyMotion features={domMax}>
        <QueryClientProvider client={queryClient}>
          <Harness values={values}>
            <DetentionPolicyForm />
          </Harness>
        </QueryClientProvider>
      </LazyMotion>
    </TestRouter>,
  );
}

beforeAll(() => {
  MotionGlobalConfig.skipAnimations = true;
});
afterAll(() => {
  MotionGlobalConfig.skipAnimations = false;
});
afterEach(() => {
  cleanup();
});

describe("detention policy form, organization default", () => {
  it("says the policy is the default and locks its status, scope and dates", async () => {
    renderForm(policy({ isOrganizationDefault: true }));

    const notice = await screen.findByRole("alert");
    expect(within(notice).getByText("The organization's default detention policy")).toBeTruthy();
    expect(within(notice).getByText(/status, scope and dates are locked/)).toBeTruthy();

    for (const label of [
      "Customer",
      "Facility",
      "Shipment types",
      "Commodities",
      "Effective From",
      "Expires",
    ]) {
      expect(lockedBy(screen.getByLabelText(label)), label).toBe(true);
    }
    expect(lockedBy(screen.getByLabelText(/^Name/))).toBe(false);
  });

  it("leaves an ordinary policy open to edit", async () => {
    renderForm(policy({ isOrganizationDefault: false }));

    await screen.findByLabelText("Customer");
    expect(screen.queryByText("The organization's default detention policy")).toBeNull();
    expect(lockedBy(screen.getByLabelText("Customer"))).toBe(false);
    expect(lockedBy(screen.getByLabelText("Effective From"))).toBe(false);
    expect(lockedBy(screen.getByLabelText("Expires"))).toBe(false);
  });
});
