import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import { worstFirst } from "../card-due";
import {
  equipmentFacts,
  equipmentName,
  equipmentStanding,
  equipmentStatusCheck,
  leaseCheck,
  registrationCheck,
  type EquipmentRecord,
} from "../equipment";
import { dueReading, personName } from "../record-card-values";

/** Fills a message's numbered places the way the catalog would, plurals read by their count. */
const t = ((message: string, ...args: unknown[]) =>
  message
    .replace(
      /\{(\d+), plural, one \{([^}]*)\} other \{([^}]*)\}\}/gu,
      (_, index, one: string, other: string) => {
        const count = Number(args[Number(index)]);
        return (count === 1 ? one : other).replace("#", String(count));
      },
    )
    .replace(/\{(\d+)\}/gu, (_, index) => String(args[Number(index)]))) as TranslateFn;

const DAY = 86_400;
const NOW = 1_791_720_000;

function unit(overrides: Partial<EquipmentRecord> = {}): EquipmentRecord {
  return {
    code: "T-101",
    status: "Available",
    make: "Freightliner",
    model: "Cascadia",
    year: 2022,
    vin: "3AKJHHDR5NSNA1234",
    licensePlateNumber: "TX-48812",
    registrationNumber: "REG-99812",
    registrationExpiry: NOW + 200 * DAY,
    externalId: "",
    ownershipType: "CompanyOwned",
    lessorName: "",
    leaseReference: "",
    leaseEndDate: null,
    createdAt: 1_700_000_000,
    equipmentManufacturer: { name: "Freightliner" },
    ...overrides,
  };
}

describe("equipmentStanding", () => {
  it("reads each status as the card draws it", () => {
    expect(equipmentStanding("Available", t)).toEqual({
      label: "Available",
      tone: "success",
      mark: "dot",
    });
    expect(equipmentStanding("AtMaintenance", t).mark).toBe("pause");
    expect(equipmentStanding("OutOfService", t)).toMatchObject({ tone: "warn", mark: "pause" });
    expect(equipmentStanding("Sold", t).tone).toBe("muted");
  });
});
describe("equipmentName", () => {
  it("names the tractor by year, make and model", () => {
    expect(equipmentName(unit())).toBe("2022 Freightliner Cascadia");
  });

  it("leaves out what is not on file, and falls back to the code", () => {
    expect(equipmentName(unit({ year: null, model: "  " }))).toBe("Freightliner");
    expect(equipmentName(unit({ year: null, make: "", model: "" }))).toBe("T-101");
  });
});
describe("personName", () => {
  it("joins the names it has", () => {
    expect(personName({ firstName: " Rosa ", lastName: "Diaz" })).toBe("Rosa Diaz");
    expect(personName({ firstName: "", lastName: "Diaz" })).toBe("Diaz");
    expect(personName(null)).toBe("");
  });
});
describe("dueReading", () => {
  it("is overdue on and after the day, close inside the notice, and fine before", () => {
    expect(dueReading(NOW, NOW, 30).state).toBe("bad");
    expect(dueReading(NOW - DAY, NOW, 30)).toEqual({ state: "bad", days: -1 });
    expect(dueReading(NOW + 30 * DAY, NOW, 30)).toEqual({ state: "warn", days: 30 });
    expect(dueReading(NOW + 31 * DAY, NOW, 30)).toEqual({ state: "ok", days: 31 });
    expect(dueReading(NOW + 1, NOW, 30)).toEqual({ state: "warn", days: 1 });
  });
});

describe("equipment checks", () => {
  it("holds a unit out of service or at maintenance, and lets one that can run pass", () => {
    expect(equipmentStatusCheck("OutOfService", t)).toMatchObject({ key: "status", state: "bad" });
    expect(equipmentStatusCheck("AtMaintenance", t)).toMatchObject({ state: "warn" });
    expect(equipmentStatusCheck("Available", t)).toBeNull();
    expect(equipmentStatusCheck("Sold", t)).toBeNull();
  });

  it("reads the registration as missing, expired, close or good", () => {
    expect(registrationCheck(null, NOW, t)).toMatchObject({
      state: "warn",
      detail: "No expiry on file",
    });
    expect(registrationCheck(NOW - DAY, NOW, t)).toMatchObject({
      state: "bad",
      title: "Registration expired",
    });
    expect(registrationCheck(NOW + 12 * DAY, NOW, t)).toMatchObject({
      state: "warn",
      title: "Registration expires in 12 days",
    });
    expect(registrationCheck(NOW + DAY, NOW, t).title).toBe("Registration expires in 1 day");
    expect(registrationCheck(NOW + 90 * DAY, NOW, t)).toMatchObject({
      state: "ok",
      title: "Registration",
    });
  });

  it("reads a lease only on a leased unit with an end date, naming the lessor", () => {
    expect(leaseCheck(unit({ leaseEndDate: NOW + DAY }), NOW, t)).toBeNull();
    expect(leaseCheck(unit({ ownershipType: "Leased" }), NOW, t)).toBeNull();
    expect(
      leaseCheck(
        unit({ ownershipType: "Leased", leaseEndDate: NOW + 45 * DAY, lessorName: "Penske" }),
        NOW,
        t,
      ),
    ).toMatchObject({ state: "warn", title: "Lease ends in 45 days" });
    const ended = leaseCheck(
      unit({ ownershipType: "Leased", leaseEndDate: NOW - 2 * DAY, lessorName: " " }),
      NOW,
      t,
    );
    expect(ended).toMatchObject({ state: "bad", title: "Lease ended" });
    expect(ended?.detail).toMatch(/^Ended /u);
  });

  it("puts what is wrong first and keeps each state's own order", () => {
    const order = worstFirst([
      { key: "a", state: "ok", title: "", detail: "" },
      { key: "b", state: "warn", title: "", detail: "" },
      { key: "c", state: "bad", title: "", detail: "" },
      { key: "d", state: "warn", title: "", detail: "" },
    ]).map((check) => check.key);
    expect(order).toEqual(["c", "b", "d", "a"]);
  });
});

describe("equipmentFacts", () => {
  it("lists a unit's facts with the card's own after the manufacturer", () => {
    const facts = equipmentFacts(
      unit(),
      "TX",
      [{ key: "load", label: "Max load", value: "45,000 lb" }],
      t,
    );
    expect(facts.map((fact) => [fact.key, fact.value])).toEqual([
      ["plate", "TX-48812 · TX"],
      ["registration", "REG-99812"],
      ["vin", "3AKJHHDR5NSNA1234"],
      ["manufacturer", "Freightliner"],
      ["load", "45,000 lb"],
      ["ownership", "Company owned"],
      ["external", "None"],
      ["since", expect.any(String)],
    ]);
  });

  it("says what is not on file, and names a lessor on a leased unit", () => {
    const facts = equipmentFacts(
      unit({
        licensePlateNumber: "",
        vin: "",
        equipmentManufacturer: null,
        ownershipType: "Leased",
        lessorName: "Penske",
        leaseReference: "L-2231",
      }),
      null,
      [],
      t,
    );
    const value = (key: string) => facts.find((fact) => fact.key === key)?.value;
    expect(value("plate")).toBe("None on file");
    expect(value("vin")).toBe("None on file");
    expect(value("manufacturer")).toBe("None on file");
    expect(value("ownership")).toBe("Leased · Penske · L-2231");
  });
});
