import { describe, expect, it } from "vitest";
import { arrivalVerdict } from "../arrival-window-cell";

const START = 1_800_000_000;
const END = START + 2 * 3600;

describe("arrivalVerdict", () => {
  it("is on time anywhere inside the window", () => {
    expect(arrivalVerdict(START + 3600, START, END)).toEqual({ minutesOff: 0, tone: "on-time" });
  });

  it("gives fifteen minutes' grace either side before calling it early or late", () => {
    expect(arrivalVerdict(END + 15 * 60, START, END).tone).toBe("on-time");
    expect(arrivalVerdict(END + 16 * 60, START, END)).toEqual({ minutesOff: 16, tone: "late" });
    expect(arrivalVerdict(START - 15 * 60, START, END).tone).toBe("on-time");
    expect(arrivalVerdict(START - 45 * 60, START, END)).toEqual({ minutesOff: -45, tone: "early" });
  });

  it("treats an appointment with no end as a single moment", () => {
    expect(arrivalVerdict(START + 20 * 60, START, null)).toEqual({ minutesOff: 20, tone: "late" });
  });
});
