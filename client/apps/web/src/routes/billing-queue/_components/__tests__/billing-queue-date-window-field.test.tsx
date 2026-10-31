import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { DateWindow } from "@/lib/billing-queue-date-window";
import { useState } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { BillingQueueDateWindowField } from "../billing-queue-date-window-field";

afterEach(cleanup);

function Harness({ onChange }: { onChange: (value: DateWindow | null) => void }) {
  const [value, setValue] = useState<DateWindow | null>(null);

  return (
    <BillingQueueDateWindowField
      label="Delivered"
      value={value}
      onChange={(next) => {
        setValue(next);
        onChange(next);
      }}
    />
  );
}

describe("BillingQueueDateWindowField", () => {
  it("moves from a rolling window to picking dates, clearing the window it replaces", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<Harness onChange={onChange} />);

    await user.click(screen.getByRole("combobox", { name: "Delivered" }));
    await user.click(await screen.findByRole("option", { name: "Last 7 days" }));
    expect(onChange).toHaveBeenLastCalledWith({ kind: "last", days: 7 });
    expect(screen.getByRole("combobox", { name: "Delivered" })).toHaveTextContent("Last 7 days");

    await user.click(screen.getByRole("combobox", { name: "Delivered" }));
    await user.click(await screen.findByRole("option", { name: "Pick dates…" }));

    expect(onChange).toHaveBeenLastCalledWith(null);
    expect(screen.getByRole("combobox", { name: "Delivered" })).toHaveTextContent("Pick dates…");
    expect(screen.getByRole("button", { name: "Delivered dates" })).toHaveTextContent("Pick dates");
  });

  it("goes back to a rolling window from picking dates", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<Harness onChange={onChange} />);

    await user.click(screen.getByRole("combobox", { name: "Delivered" }));
    await user.click(await screen.findByRole("option", { name: "Pick dates…" }));
    await user.keyboard("{Escape}");
    await user.click(screen.getByRole("combobox", { name: "Delivered" }));
    await user.click(await screen.findByRole("option", { name: "Last 30 days" }));

    expect(onChange).toHaveBeenLastCalledWith({ kind: "last", days: 30 });
    expect(screen.queryByRole("button", { name: "Delivered dates" })).not.toBeInTheDocument();
  });
});
