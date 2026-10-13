import type { BillingControl } from "@/types/billing-control";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { FormProvider, useForm, type UseFormReturn } from "react-hook-form";
import { useEffect } from "react";
import { afterEach, describe, expect, it } from "vitest";
import { PaymentRemindersCard } from "../payment-reminders-card";

afterEach(cleanup);

let form: UseFormReturn<BillingControl> | null = null;

function Harness({ enabled }: { enabled: boolean }) {
  const current = useForm<BillingControl>({
    defaultValues: {
      paymentRemindersEnabled: enabled,
      paymentReminderDays: [1, 15, 30],
    } as BillingControl,
  });
  useEffect(() => {
    form = current;
  });
  return (
    <FormProvider {...current}>
      <PaymentRemindersCard />
    </FormProvider>
  );
}

describe("PaymentRemindersCard", () => {
  it("hides the schedule while reminders are off", () => {
    render(<Harness enabled={false} />);

    expect(screen.queryByLabelText("Remind on days past due")).not.toBeInTheDocument();
  });

  it("stores the days typed as numbers, earliest first", async () => {
    render(<Harness enabled />);

    const input = screen.getByLabelText("Remind on days past due");
    expect(input).toHaveValue("1, 15, 30");
    await userEvent.clear(input);
    await userEvent.type(input, "45, 7, 21");

    expect(form?.getValues("paymentReminderDays")).toEqual([7, 21, 45]);
  });

  it("says when the days typed are not whole numbers and keeps the last good schedule", async () => {
    render(<Harness enabled />);

    const input = screen.getByLabelText("Remind on days past due");
    await userEvent.clear(input);
    await userEvent.type(input, "1, 2.5");

    expect(
      screen.getByText("Enter whole days separated by commas, such as 1, 15, 30"),
    ).toBeInTheDocument();
    expect(form?.getValues("paymentReminderDays")).toEqual([1, 2]);
  });
});
