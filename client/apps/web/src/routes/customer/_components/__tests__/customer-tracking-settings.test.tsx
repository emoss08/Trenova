import type { Customer } from "@trenova/shared/types/customer";
import {
  defaultCustomerTrackingProfile,
  type CustomerTrackingProfile,
} from "@trenova/shared/types/customer-tracking";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { FormProvider, useForm, type UseFormReturn } from "react-hook-form";
import { afterEach, describe, expect, it } from "vitest";
import { CustomerTrackingSettingsForm } from "../customer-tracking-settings";

afterEach(cleanup);

let current: UseFormReturn<Customer> | null = null;

function Harness({ trackingProfile }: { trackingProfile: CustomerTrackingProfile }) {
  const form = useForm<Customer>({
    defaultValues: { name: "Halstead Foods", trackingProfile } as Customer,
  });
  // oxlint-disable-next-line react/globals -- the test drives the form from outside it
  current = form;
  return (
    <FormProvider {...form}>
      <CustomerTrackingSettingsForm />
    </FormProvider>
  );
}

function renderSettings(overrides: Partial<CustomerTrackingProfile> = {}) {
  render(<Harness trackingProfile={{ ...defaultCustomerTrackingProfile(), ...overrides }} />);
  if (!current) {
    throw new Error("form not mounted");
  }
  return current;
}

describe("CustomerTrackingSettingsForm", () => {
  it("has no save of its own: the customer form's save writes it", () => {
    renderSettings();

    expect(screen.queryByRole("button", { name: /save/i })).toBeNull();
    expect(screen.queryByRole("button", { name: /discard/i })).toBeNull();
  });

  it("writes each change into the customer form's tracking settings", async () => {
    const form = renderSettings();

    await userEvent.click(screen.getByRole("switch", { name: /^Delivered/ }));
    await userEvent.click(screen.getByRole("switch", { name: /^Allow tracking links/ }));

    expect(form.getValues("trackingProfile")).toEqual({
      linkEnabled: false,
      visibleFields: ["Status", "Windows", "Actuals", "Eta", "References"],
      linkTtlDays: 30,
      notifyPickedUp: false,
      notifyDelayed: false,
      notifyDelivered: true,
      notifyPodReady: false,
      includeLink: true,
      recipients: "",
    });
  });

  it("turns off the link in each email while links are not allowed", () => {
    renderSettings({ linkEnabled: false });

    expect(
      screen.getByRole("switch", { name: /^Include a tracking link in each email/ }),
    ).toHaveAttribute("aria-disabled", "true");
    expect(screen.getByText("Allow tracking links to include one.")).toBeInTheDocument();
  });

  it("shows the stored settings it was opened with", () => {
    renderSettings({ notifyPodReady: true, recipients: "ops@halstead.example" });

    expect(screen.getByRole("switch", { name: /^Proof of delivery ready/ })).toBeChecked();
    expect(
      screen.getByRole("switch", { name: /^Include a tracking link in each email/ }),
    ).not.toHaveAttribute("aria-disabled", "true");
    expect(screen.getByLabelText("Send to")).toHaveValue("ops@halstead.example");
  });
});
