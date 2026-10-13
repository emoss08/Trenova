import type { Customer } from "@trenova/shared/types/customer";
import { cleanup, render, screen } from "@testing-library/react";
import { FormProvider, useForm } from "react-hook-form";
import { afterEach, describe, expect, it } from "vitest";
import { CreditExposureSummary } from "../credit-exposure-summary";

afterEach(cleanup);

function Harness({ profile }: { profile: Partial<NonNullable<Customer["billingProfile"]>> }) {
  const form = useForm<Customer>({
    defaultValues: { billingProfile: { billingCurrency: "USD", ...profile } } as Customer,
  });
  return (
    <FormProvider {...form}>
      <CreditExposureSummary />
    </FormProvider>
  );
}

describe("CreditExposureSummary", () => {
  it("shows the open receivables and the share of the limit they use", () => {
    render(
      <Harness
        profile={{
          creditBalance: 9500,
          creditLimit: 10000,
          creditReviewedAt: 1_700_000_000,
          creditStatus: "Active",
        }}
      />,
    );

    expect(screen.getByText("$9,500.00")).toBeInTheDocument();
    expect(screen.getByText("95% of the limit")).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("explains an automatic hold and how it lifts", () => {
    render(
      <Harness
        profile={{
          creditBalance: 15000,
          creditLimit: 10000,
          creditReviewedAt: 1_700_000_000,
          creditStatus: "Hold",
          creditHoldAutomatic: true,
          creditHoldReason:
            "Open receivables of 15000.00 USD are over the credit limit of 10000.00 USD",
        }}
      />,
    );

    expect(
      screen.getByText(
        "Held automatically: Open receivables of 15000.00 USD are over the credit limit of 10000.00 USD. The hold lifts once the balance is paid back under the limit.",
      ),
    ).toBeInTheDocument();
  });

  it("says when the balance has not been computed yet", () => {
    render(<Harness profile={{ creditBalance: 0, creditLimit: null, creditReviewedAt: null }} />);

    expect(
      screen.getByText(
        "Computed from open invoices once this customer's receivables next move, or overnight.",
      ),
    ).toBeInTheDocument();
    expect(screen.queryByText(/of the limit/)).not.toBeInTheDocument();
  });
});
