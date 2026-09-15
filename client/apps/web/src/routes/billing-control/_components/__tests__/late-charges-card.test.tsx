import type { BillingControl } from "@/types/billing-control";
import { cleanup, render, screen } from "@testing-library/react";
import { FormProvider, useForm } from "react-hook-form";
import { afterEach, describe, expect, it } from "vitest";
import { LateChargesCard } from "../billing-control-form";

afterEach(cleanup);

function Harness({ mode }: { mode: BillingControl["lateChargeAssessmentMode"] }) {
  const form = useForm<BillingControl>({
    defaultValues: {
      lateChargeAssessmentMode: mode,
      lateChargeMinimumAmount: 5,
    } as BillingControl,
  });
  return (
    <FormProvider {...form}>
      <LateChargesCard />
    </FormProvider>
  );
}

describe("LateChargesCard", () => {
  it("offers the assessment mode and the minimum charge", () => {
    render(<Harness mode="Automatic" />);

    expect(screen.getByRole("button", { name: /Automatic/ })).toBeInTheDocument();
    expect(screen.getByLabelText("Minimum late charge")).toHaveValue("5.00");
  });

  it("warns that nothing is raised while the mode is Preview", () => {
    render(<Harness mode="Preview" />);

    expect(screen.getByText(/computes what it would raise and writes nothing/)).toBeInTheDocument();
  });
});
