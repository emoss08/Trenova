import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { setUserDatePreferences } from "@trenova/shared/lib/date";
import { useForm } from "react-hook-form";
import { afterEach, describe, expect, it } from "vitest";
import { TimeOfDayField } from "../time-of-day-field";

afterEach(() => {
  cleanup();
  setUserDatePreferences({});
});

type Values = { submissionTime: string };

let latest: Values | undefined;

function Harness({ value }: { value: string }) {
  const form = useForm<Values>({ defaultValues: { submissionTime: value } });
  latest = form.watch();
  return (
    <TimeOfDayField
      control={form.control}
      name="submissionTime"
      label="Daily schedule time"
      description="When the day's schedule is proposed."
    />
  );
}

const hour = () => screen.getByRole("textbox", { name: "Hour" }) as HTMLInputElement;
const minute = () => screen.getByRole("textbox", { name: "Minute" }) as HTMLInputElement;
const period = () => screen.getByRole("textbox", { name: "AM or PM" }) as HTMLInputElement;

describe("TimeOfDayField", () => {
  it("splits the stored 24-hour value into hour, minute and period on a 12-hour clock", () => {
    setUserDatePreferences({ timeFormat: "12-hour" });
    render(<Harness value="14:30" />);

    expect(hour().value).toBe("02");
    expect(minute().value).toBe("30");
    expect(period().value).toBe("PM");
  });

  it("shows empty segments when no time is stored", () => {
    setUserDatePreferences({ timeFormat: "12-hour" });
    render(<Harness value="" />);

    expect(hour().value).toBe("--");
    expect(minute().value).toBe("--");
    expect(period().value).toBe("--");
  });

  it("takes digits segment by segment and moves on once a segment is full", () => {
    setUserDatePreferences({ timeFormat: "12-hour" });
    render(<Harness value="09:00" />);

    hour().focus();
    fireEvent.keyDown(hour(), { key: "1" });
    fireEvent.keyDown(hour(), { key: "0" });
    expect(document.activeElement).toBe(minute());
    fireEvent.keyDown(minute(), { key: "4" });
    fireEvent.keyDown(minute(), { key: "5" });

    expect(latest?.submissionTime).toBe("10:45");
  });

  it("ignores letters in the hour and sets the period from A or P", () => {
    setUserDatePreferences({ timeFormat: "12-hour" });
    render(<Harness value="09:15" />);

    fireEvent.keyDown(hour(), { key: "x" });
    expect(latest?.submissionTime).toBe("09:15");

    fireEvent.keyDown(period(), { key: "p" });
    expect(latest?.submissionTime).toBe("21:15");
    fireEvent.keyDown(period(), { key: "a" });
    expect(latest?.submissionTime).toBe("09:15");
  });

  it("drops the period on a 24-hour clock and steps the hour with the arrows", () => {
    setUserDatePreferences({ timeFormat: "24-hour" });
    render(<Harness value="09:00" />);

    expect(screen.queryByRole("textbox", { name: "AM or PM" })).toBeNull();
    fireEvent.keyDown(hour(), { key: "ArrowUp" });

    expect(latest?.submissionTime).toBe("10:00");
  });

  it("offers the clock picker beside the typed time", () => {
    render(<Harness value="09:00" />);

    expect(screen.getByRole("button", { name: "Open time picker" })).toBeTruthy();
  });
});
