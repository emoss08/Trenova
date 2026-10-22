import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { StudioRailTabs } from "../studio-rail-tabs";
import type { TariffTestsState } from "../use-tariff-tests";

type Tab = "tests" | "reference" | "history" | "nova";

const TABS: { value: Exclude<Tab, "nova">; label: string }[] = [
  { value: "tests", label: "Tests" },
  { value: "reference", label: "Reference" },
  { value: "history", label: "History" },
];

const ran = (passed: number, failed: number): TariffTestsState => ({
  status: "ran",
  run: { tests: [], error: null },
  passed,
  failed,
  stale: false,
});

function strip(tests: TariffTestsState, overrides: { tab?: Tab; novaBusy?: boolean } = {}) {
  const onTab = vi.fn();
  const onClose = vi.fn();
  render(
    <StudioRailTabs<Tab>
      label="Library panes"
      tabs={TABS}
      tab={overrides.tab ?? "tests"}
      onTab={onTab}
      tests={tests}
      novaBusy={overrides.novaBusy ?? false}
      onClose={onClose}
    />,
  );
  return { onTab, onClose };
}

describe("StudioRailTabs", () => {
  it("lists the Studio's panes in order, then Nova, and marks the open one", () => {
    strip({ status: "empty" }, { tab: "reference" });
    const tabs = screen.getAllByRole("tab");
    expect(tabs.map((tab) => tab.textContent)).toEqual(["Tests", "Reference", "History", "Nova"]);
    expect(tabs.map((tab) => tab.getAttribute("aria-selected"))).toEqual([
      "false",
      "true",
      "false",
      "false",
    ]);
    expect(screen.getByRole("tablist", { name: "Library panes" })).toBeTruthy();
  });

  it("counts the passing tests against the total on the tests tab", () => {
    strip(ran(1, 2));
    expect(screen.getByRole("tab", { name: "Tests 1/3" })).toBeTruthy();
  });

  it("shows no count before a run or when the file has no tests", () => {
    strip({ status: "blocked", errors: 2 });
    expect(screen.getByRole("tab", { name: "Tests" })).toBeTruthy();
  });

  it("opens the pane a tab names, Nova included, and closes the panel", async () => {
    const user = userEvent.setup();
    const { onTab, onClose } = strip(ran(2, 0));
    await user.click(screen.getByRole("tab", { name: "History" }));
    await user.click(screen.getByRole("tab", { name: "Nova" }));
    await user.click(screen.getByRole("button", { name: "Close panel" }));
    expect(onTab.mock.calls).toEqual([["history"], ["nova"]]);
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
