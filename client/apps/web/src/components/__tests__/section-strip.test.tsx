import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { CalendarIcon, ClockIcon, File06Icon } from "@trenova/shared/components/icons";
import { useState } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { SectionStrip, type SectionStripItem } from "../section-strip";

type View = "hours-of-service" | "schedule" | "time-off";

const ITEMS: SectionStripItem<View>[] = [
  {
    value: "hours-of-service",
    label: "Hours of service",
    icon: ClockIcon,
    state: "Driving · 6h 12m drive left",
  },
  { value: "schedule", label: "Schedule", icon: CalendarIcon, state: "Not on a shift" },
  {
    value: "time-off",
    label: "Time off",
    icon: File06Icon,
    state: "9.5 vacation days available",
    tone: "warning",
  },
];

const CELL_WIDTH = 120;

function cellIndex(element: HTMLElement): number {
  const cells = Array.from(element.parentElement?.querySelectorAll('[role="tab"]') ?? []);
  return cells.indexOf(element);
}

function Strip({ initial = "hours-of-service" as View }) {
  const [value, setValue] = useState<View>(initial);
  return (
    <SectionStrip items={ITEMS} value={value} onValueChange={setValue} aria-label="Time views" />
  );
}

function plate(): HTMLElement {
  const indicator = document.querySelector<HTMLElement>('[data-slot="tab-indicator"]');
  if (!indicator) throw new Error("no selection plate");
  return indicator;
}

describe("SectionStrip", () => {
  beforeEach(() => {
    // happy-dom lays nothing out, so each cell is given the box a browser would: 120px wide,
    // side by side.
    vi.spyOn(HTMLElement.prototype, "offsetWidth", "get").mockImplementation(function (
      this: HTMLElement,
    ) {
      return this.getAttribute("role") === "tab" ? CELL_WIDTH : 0;
    });
    vi.spyOn(HTMLElement.prototype, "offsetHeight", "get").mockImplementation(function (
      this: HTMLElement,
    ) {
      return this.getAttribute("role") === "tab" ? 48 : 0;
    });
    vi.spyOn(HTMLElement.prototype, "offsetLeft", "get").mockImplementation(function (
      this: HTMLElement,
    ) {
      return this.getAttribute("role") === "tab" ? cellIndex(this) * CELL_WIDTH : 0;
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("draws each view as a tab with its name and the line on how it stands", () => {
    render(<Strip />);

    const list = screen.getByRole("tablist", { name: "Time views" });
    const tabs = screen.getAllByRole("tab");
    expect(list).toContainElement(tabs[0]);
    expect(tabs.map((tab) => tab.textContent)).toEqual([
      "Hours of serviceDriving · 6h 12m drive left",
      "ScheduleNot on a shift",
      "Time off9.5 vacation days available",
    ]);
    expect(tabs[0]).toHaveAttribute("aria-selected", "true");
  });

  it("colours a state line by its tone and leaves an untoned one muted", () => {
    render(<Strip />);

    expect(screen.getByText("9.5 vacation days available")).toHaveAttribute("data-tone", "warning");
    expect(screen.getByText("9.5 vacation days available")).toHaveClass("text-warning-foreground");
    expect(screen.getByText("Not on a shift")).toHaveAttribute("data-tone", "none");
    expect(screen.getByText("Not on a shift")).toHaveClass("text-muted-foreground");
  });

  it("moves the selection plate onto the cell that is chosen", async () => {
    const user = userEvent.setup();
    render(<Strip />);

    await waitFor(() => expect(plate().style.getPropertyValue("--active-tab-left")).toBe("0px"));
    expect(plate().style.getPropertyValue("--active-tab-width")).toBe("120px");

    await user.click(screen.getByRole("tab", { name: /Time off/ }));

    expect(screen.getByRole("tab", { name: /Time off/ })).toHaveAttribute("aria-selected", "true");
    await waitFor(() => expect(plate().style.getPropertyValue("--active-tab-left")).toBe("240px"));
  });

  it("follows the arrow keys, Home and End as a tab list does", async () => {
    const user = userEvent.setup();
    render(<Strip />);
    const [hours, schedule, timeOff] = screen.getAllByRole("tab");

    hours.focus();
    await user.keyboard("{ArrowRight}");
    expect(schedule).toHaveFocus();
    expect(schedule).toHaveAttribute("aria-selected", "true");

    await user.keyboard("{End}");
    expect(timeOff).toHaveFocus();
    expect(timeOff).toHaveAttribute("aria-selected", "true");

    await user.keyboard("{Home}");
    expect(hours).toHaveFocus();
    expect(hours).toHaveAttribute("aria-selected", "true");

    await user.keyboard("{ArrowLeft}");
    expect(timeOff).toHaveAttribute("aria-selected", "true");
  });

  it("draws a shimmer in place of each state line while they are read", () => {
    render(
      <SectionStrip
        items={ITEMS}
        value="schedule"
        onValueChange={() => undefined}
        aria-label="Time views"
        loading
      />,
    );

    expect(screen.queryByText("Not on a shift")).not.toBeInTheDocument();
    expect(document.querySelectorAll(".ui-shimmer")).toHaveLength(3);
  });
});
