import { render as renderInDom, screen } from "@testing-library/react";
import { TestRouter } from "@trenova/shared/testing/test-router";
import type { ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { SupportAccessLimits } from "../support-access-limits";

function render(ui: ReactElement) {
  return renderInDom(<TestRouter>{ui}</TestRouter>);
}

const settings = {
  available: true,
  staffDomain: "trenova.app",
  durations: [1, 4, 24, 72],
  maxLiveGrants: 5,
  limits: [
    "Change users, roles or how people sign in",
    "Change where money is sent: remittance instructions, remit-to addresses, the factor an invoice or customer pays, or your company address printed on invoices",
    "Export data in bulk",
  ],
};

describe("SupportAccessLimits", () => {
  it("lists every limit the server's support policy holds, in its order", () => {
    render(<SupportAccessLimits settings={settings} />);

    const items = screen.getAllByRole("listitem").map((item) => item.textContent);
    expect(items).toEqual(settings.limits);
  });

  it("says what each kind of access allows", () => {
    render(<SupportAccessLimits settings={settings} />);

    expect(
      screen.getByText(
        "Staff can read what an administrator can, except the things Trenova support can never see, and cannot change anything.",
      ),
    ).toBeTruthy();
    expect(
      screen.getByText(
        "Staff can do what an administrator can, except the things Trenova support can never do.",
      ),
    ).toBeTruthy();
  });

  it("draws no list when the server names no limit", () => {
    render(<SupportAccessLimits settings={{ ...settings, limits: [] }} />);

    expect(screen.queryAllByRole("listitem")).toEqual([]);
  });
});
