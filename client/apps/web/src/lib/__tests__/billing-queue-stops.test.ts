import { describe, expect, it } from "vitest";
import { dwellText, shipmentMiles, stopRows } from "../billing-queue-stops";

const t = ((message: string, ...args: unknown[]) =>
  args.reduce<string>(
    (text, arg, index) => text.replace(`{${index}}`, String(arg)),
    message,
  )) as never;

const stop = (overrides: Record<string, unknown>) => ({
  id: "stp",
  type: "Pickup",
  sequence: 0,
  scheduledWindowStart: 1_000,
  addressLine: "1 Main St",
  location: { name: "Dallas DC", city: "Dallas", state: { abbreviation: "TX" } },
  ...overrides,
});

describe("billing queue stops", () => {
  it("lists every stop move by move and stop by stop, with its time on site", () => {
    const rows = stopRows([
      {
        sequence: 1,
        stops: [
          stop({
            id: "stp_d",
            type: "Delivery",
            sequence: 0,
            actualArrival: 5_000,
            actualDeparture: 20_000,
          }),
        ],
      },
      {
        sequence: 0,
        stops: [
          stop({
            id: "stp_s",
            type: "SplitDelivery",
            sequence: 1,
            location: null,
            addressLine: "Jackson, MS",
          }),
          stop({ id: "stp_p", sequence: 0, actualArrival: 1_200 }),
        ],
      },
    ]);

    expect(rows.map((row) => row.key)).toEqual(["stp_p", "stp_s", "stp_d"]);
    expect(rows[0]).toMatchObject({ name: "Dallas DC", place: "Dallas, TX", onSite: null });
    expect(rows[1]).toMatchObject({ name: "Jackson, MS", place: "Jackson, MS" });
    expect(rows[2]!.onSite).toBe(15_000);
  });

  it("adds up the miles run loaded and empty", () => {
    expect(
      shipmentMiles([
        { sequence: 0, loaded: true, distance: 452.4, stops: [] },
        { sequence: 1, loaded: false, distance: 18.2, stops: [] },
        { sequence: 2, distance: null, stops: [] },
      ]),
    ).toEqual({ loaded: 452, empty: 18 });
  });

  it("says time on site in hours and minutes", () => {
    expect(dwellText(4 * 3600 + 10 * 60, t)).toBe("4h 10m");
    expect(dwellText(53 * 60, t)).toBe("53m");
  });
});
