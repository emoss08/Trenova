import type { BillingControl } from "@/types/billing-control";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { FormProvider, useForm } from "react-hook-form";
import { afterEach, describe, expect, it } from "vitest";
import { MonthlyStatementsCard } from "../monthly-statements-card";

afterEach(cleanup);

function Harness({ enabled }: { enabled: boolean }) {
  const form = useForm<BillingControl>({
    defaultValues: {
      monthlyStatementsEnabled: enabled,
      monthlyStatementDay: 3,
    } as BillingControl,
  });
  return (
    <FormProvider {...form}>
      <MonthlyStatementsCard />
    </FormProvider>
  );
}

describe("MonthlyStatementsCard", () => {
  it("asks for the day only while monthly statements are on", () => {
    render(<Harness enabled={false} />);

    expect(screen.queryByText("Day of the month to send")).not.toBeInTheDocument();
  });

  it("shows the day statements go out on", () => {
    render(<Harness enabled />);

    expect(screen.getByText("Day of the month to send")).toBeInTheDocument();
    expect(screen.getByRole("textbox")).toHaveValue("3");
  });

  it("reveals the day when monthly statements are turned on", async () => {
    render(<Harness enabled={false} />);

    await userEvent.click(screen.getByRole("switch", { name: "Send monthly statements" }));

    expect(screen.getByText("Day of the month to send")).toBeInTheDocument();
  });
});
