import type { DetentionCustomer } from "@/lib/graphql/detention-intelligence";
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { CustomerCard } from "../paid-section";

function customer(index: number): DetentionCustomer {
  return {
    customerId: `cus_${index}`,
    customerName: `Customer ${String(index).padStart(2, "0")}`,
    stopCount: 10,
    detainedCount: 4,
    pastFreeMinutes: 600 + index * 60,
    billedAmount: "500",
    driverPayAmount: "200",
    netMargin: String(100 + index),
    waivedAmount: "0",
    disputeCount: 0,
    noticeDueCount: 4,
    noticeOnTimeCount: 3,
  };
}

const forty = Array.from({ length: 40 }, (_, index) => customer(index));
/** Every row of the list: the pinned header and the customers mounted around the viewport. */
const customerRows = () => screen.getAllByRole("row").length - 1;

beforeEach(() => {
  vi.useFakeTimers({
    toFake: ["setTimeout", "clearTimeout", "requestAnimationFrame", "cancelAnimationFrame"],
  });
  Element.prototype.scrollTo = vi.fn();
});
afterEach(() => {
  cleanup();
  vi.useRealTimers();
});

describe("CustomerCard", () => {
  it("shows the six customers that cost the most and offers the rest", () => {
    render(<CustomerCard customers={forty} marginsVisible />);

    expect(customerRows()).toBe(6);
    expect(screen.getByRole("button", { name: "Show 34 more customers" })).toBeInTheDocument();
  });

  it("mounts only the customers near the viewport once every one is shown", () => {
    render(<CustomerCard customers={forty} marginsVisible />);

    fireEvent.click(screen.getByRole("button", { name: "Show 34 more customers" }));

    expect(customerRows()).toBeGreaterThan(6);
    expect(customerRows()).toBeLessThan(40);
    expect(screen.getByRole("button", { name: "Show fewer" })).toHaveAttribute(
      "aria-expanded",
      "true",
    );
  });

  it("keeps the added rows while they fade on Show fewer, then collapses to six", () => {
    render(<CustomerCard customers={forty} marginsVisible />);
    fireEvent.click(screen.getByRole("button", { name: "Show 34 more customers" }));
    const expanded = customerRows();

    fireEvent.click(screen.getByRole("button", { name: "Show fewer" }));

    expect(customerRows()).toBe(expanded);
    act(() => {
      vi.runAllTimers();
    });
    expect(customerRows()).toBe(6);
    expect(screen.getByRole("button", { name: "Show 34 more customers" })).toHaveAttribute(
      "aria-expanded",
      "false",
    );
  });

  it("reopens instead of collapsing when Show more is pressed while the rows are leaving", () => {
    render(<CustomerCard customers={forty} marginsVisible />);
    fireEvent.click(screen.getByRole("button", { name: "Show 34 more customers" }));
    fireEvent.click(screen.getByRole("button", { name: "Show fewer" }));

    fireEvent.click(screen.getByRole("button", { name: "Show 34 more customers" }));
    act(() => {
      vi.runAllTimers();
    });

    expect(customerRows()).toBeGreaterThan(6);
    expect(screen.getByRole("button", { name: "Show fewer" })).toBeInTheDocument();
  });
});
