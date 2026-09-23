import { setUserDatePreferences } from "@trenova/shared/lib/date";
import { afterEach, describe, expect, it } from "vitest";
import {
  classifyValues,
  displayLabel,
  formatDisplayValue,
  formatMetric,
  humanizeCode,
  humanizeKey,
  isHiddenKey,
  isRecordId,
  projectColumns,
  projectRecord,
  projectValue,
  readDisplayColumns,
  recordLabel,
  statusTone,
  type DisplayType,
} from "../readable-values";

const t = (message: string | null | undefined) => message ?? "";

afterEach(() => setUserDatePreferences({}));

/*
 * These are the server's rules (assistantservice/artifact_display.go), held
 * here for artifacts stored before the server applied them and for the step
 * details, which read the model's own copy. The cases below are the server's
 * cases, so a rule that drifts on one side fails here.
 */
describe("isRecordId", () => {
  it("knows a PULID by its shape and nothing else by it", () => {
    expect(isRecordId("inst_01M37R101VKZTB7TSKR30FJ0AT")).toBe(true);
    expect(isRecordId(" shp_01M37R101VKZTB7TSKR30FJ0AT ")).toBe(true);

    for (const value of [
      "shp_1",
      "PRO-778",
      "01M37R101VKZTB7TSKR30FJ0AT",
      "Inst_01M37R101VKZTB7TSKR30FJ0AT",
      "inst_01m37r101vkztb7tskr30fj0at",
      "inst_01M37R101VKZTB7TSKR30FJ0AI",
      "a_very_long_prefix_01M37R101VKZTB7TSKR30FJ0AT",
      "Three workers' medical cards expire",
      42,
      null,
    ]) {
      expect(isRecordId(value), String(value)).toBe(false);
    }
  });
});

describe("isHiddenKey", () => {
  it("hides ids under any name, the tenant and bookkeeping", () => {
    for (const key of [
      "id",
      "customerId",
      "shipmentID",
      "workerIds",
      "business_unit_id",
      "organizationId",
      "version",
      "direction",
    ]) {
      expect(isHiddenKey(key), key).toBe(true);
    }
  });

  it("does not take a word that ends in the same letters for an id", () => {
    for (const key of ["paid", "valid", "fluid", "name", "status"]) {
      expect(isHiddenKey(key), key).toBe(false);
    }
  });
});

describe("classifyValues", () => {
  const cases: [string, string, unknown[], DisplayType | null][] = [
    ["a flag that never holds is noise", "stale", [false, false], null],
    ["a flag that holds is said", "stale", [false, true], "flag"],
    ["a yes or no", "hazardous", [true, false], "boolean"],
    ["a count named like a date stays a count", "daysUntilExpiry", [21], "number"],
    ["an unset date is nothing", "medicalCardExpiry", [0], null],
    ["the phrase for an unset date is words", "medicalCardExpiry", ["none on file"], "text"],
    ["a phrase beside real dates", "medicalCardExpiry", ["none on file", 1_790_187_600], "date"],
    ["a date already written", "windowStart", ["2026-08-24 11:20 PDT (30 days ago)"], "datetime"],
    ["an amount due is not a date", "amountDue", ["12.00"], "text"],
    ["reason is not a date", "reason", ["Driver out sick"], "text"],
    ["a total written as a decimal is money", "total", ["12.00"], "money"],
    ["a total counted is a number", "total", [4], "number"],
    ["a percentage", "onTimePercent", ["92.5"], "percent"],
    ["a year is not a quantity", "year", [2019], "text"],
    ["a state is two letters, not a status", "state", ["CA"], "text"],
    ["a status with spaces is words", "assignmentBlocked", ["Medical card lapsed"], "text"],
    ["a type is a category", "driverType", ["OTR", "Local"], "enum"],
    ["prose by its name", "recommendation", ["Call them."], "longText"],
    ["prose by its length", "note", ["x".repeat(121)], "longText"],
    ["words in a list", "tags", [["hazmat", "team"]], "text"],
    ["records nested in a row", "stops", [[{ city: "Reno" }]], null],
    ["a nested record by its name", "customer", [{ id: "cus_1", name: "Acme" }], "text"],
    ["a nested record with no name", "rating", [{ score: 3 }], null],
    ["a column of nothing but ids", "assignedTo", ["wrk_01M37R101VKZTB7TSKR30FJ0AT"], null],
    ["an empty column", "dismissReason", [undefined, ""], null],
    ["which way is worse", "direction", ["HigherIsWorse"], null],
    [
      "measurements",
      "metrics",
      [[{ label: "Workers affected", value: "3", unit: "Count", direction: "HigherIsWorse" }]],
      "metrics",
    ],
    ["links", "links", [[{ label: "Workers", path: "/hr/workers", count: 3 }]], "links"],
  ];

  it.each(cases)("%s", (_name, key, values, want) => {
    expect(classifyValues(key, values)).toBe(want);
  });

  // A bare amount beside a method may be a percentage of the linehaul.
  it("reads an amount beside a method as a figure, not money", () => {
    expect(classifyValues("amount", ["12.50"], false)).not.toBe("money");
    expect(classifyValues("amount", ["12.50"], true)).toBe("money");
  });
});

describe("projectValue", () => {
  it("keeps what a measurement says and drops which way is worse", () => {
    expect(
      projectValue("metrics", [
        { direction: "HigherIsWorse", label: "Workers affected", unit: "Count", value: "3" },
        { label: "No value" },
      ]),
    ).toEqual([{ label: "Workers affected", value: "3", unit: "Count" }]);
  });

  it("keeps only the links that stay inside the app", () => {
    expect(
      projectValue("links", [
        { label: "Workers", path: "/hr/workers", count: 3 },
        { label: "Elsewhere", path: "//example.com" },
        { label: "Away", path: "https://example.com" },
      ]),
    ).toEqual([{ label: "Workers", path: "/hr/workers", count: 3 }]);
  });

  it("draws nothing for an id, an implausible instant or a flag that does not hold", () => {
    expect(projectValue("text", "shp_01M37R101VKZTB7TSKR30FJ0AT")).toBeUndefined();
    expect(projectValue("date", 21)).toBeUndefined();
    expect(projectValue("flag", false)).toBeUndefined();
    expect(projectValue("boolean", false)).toBe(false);
  });
});

describe("labels", () => {
  it("says what happened rather than when", () => {
    expect(displayLabel("detectedOn", "date")).toBe("Detected");
    expect(displayLabel("createdAt", "datetime")).toBe("Created");
    expect(displayLabel("windowStart", "datetime")).toBe("Window start");
    expect(displayLabel("lastMvrCheck", "date")).toBe("Last MVR check");
  });

  it("keeps initialisms and sentence case", () => {
    expect(humanizeKey("customerId")).toBe("Customer ID");
    expect(humanizeKey("proNumber")).toBe("Pro number");
    expect(humanizeKey("dotNumber")).toBe("DOT number");
    expect(humanizeKey("cdlClass")).toBe("CDL class");
  });

  it("reads a member of a set as words, and a code in capitals as it is", () => {
    expect(humanizeCode("ServiceQuality")).toBe("Service quality");
    expect(humanizeCode("InTransit")).toBe("In transit");
    expect(humanizeCode("IN_TRANSIT")).toBe("In transit");
    expect(humanizeCode("OTR")).toBe("OTR");
  });
});

describe("statusTone", () => {
  it("follows the phase a status sits in, and guesses nothing", () => {
    expect(statusTone("Critical")).toBe("danger");
    expect(statusTone("Warning")).toBe("warning");
    expect(statusTone("Info")).toBe("neutral");
    expect(statusTone("InTransit")).toBe("info");
    expect(statusTone("Delivered")).toBe("success");
    expect(statusTone("Something new")).toBe("neutral");
  });
});

describe("recordLabel", () => {
  it("names a record by what it is called, never by its id", () => {
    expect(recordLabel({ id: "inst_01M37R101VKZTB7TSKR30FJ0AT", headline: "3 cards expire" })).toBe(
      "3 cards expire",
    );
    expect(recordLabel({ code: "inst_01M37R101VKZTB7TSKR30FJ0AT" })).toBe("");
    expect(recordLabel({ firstName: "Maria", lastName: "Ortiz" })).toBe("Maria Ortiz");
  });
});

describe("formatting", () => {
  // The same instant is a different day for a reader in Tokyo than for one in
  // Los Angeles; a date is read where the reader is.
  it("formats a date in the reader's own timezone", () => {
    setUserDatePreferences({ timezone: "America/Los_Angeles" });
    expect(formatDisplayValue("date", 1_790_187_600, t)).toBe("Sep 23, 2026");

    setUserDatePreferences({ timezone: "Asia/Tokyo" });
    expect(formatDisplayValue("date", 1_790_187_600, t)).toBe("Sep 24, 2026");
  });

  it("leaves a date already written as it was written", () => {
    expect(formatDisplayValue("datetime", "2026-08-24 11:20 PDT (30 days ago)", t)).toBe(
      "2026-08-24 11:20 PDT (30 days ago)",
    );
  });

  it("reads measurements in their units", () => {
    expect(formatMetric({ label: "Workers affected", value: "3", unit: "Count" })).toBe("3");
    expect(formatMetric({ label: "First expiry in", value: "0", unit: "Days" })).toBe("0d");
    expect(
      formatDisplayValue(
        "metrics",
        [
          { label: "Workers affected", value: "3", unit: "Count" },
          { label: "First expiry in", value: "0", unit: "Days" },
        ],
        t,
      ),
    ).toBe("Workers affected 3 · First expiry in 0d");
  });

  it("reads money, figures and a yes or no", () => {
    expect(formatDisplayValue("money", "2840.5", t)).toBe("$2,840.50");
    expect(formatDisplayValue("number", 1234, t)).toBe("1,234");
    expect(formatDisplayValue("percent", "92.5", t)).toBe("92.5%");
    expect(formatDisplayValue("boolean", false, t)).toBe("No");
  });
});

describe("stored payloads", () => {
  it("reads projected columns and drops an id or an unknown type", () => {
    expect(
      readDisplayColumns([
        { key: "headline", label: "Headline", type: "text" },
        { key: "id", label: "ID", type: "text" },
        { key: "severity", label: "Severity", type: "colour" },
      ]),
    ).toEqual([{ key: "headline", label: "Headline", type: "text" }]);
    // A payload stored before the projection declared its columns as names.
    expect(readDisplayColumns(["proNumber", "status"])).toBeNull();
  });

  it("projects a stored table's columns with the name first", () => {
    expect(
      projectColumns(
        ["id", "status", "proNumber", "customerId"],
        [{ id: "shp_1", status: "InTransit", proNumber: "P1", customerId: "cus_1" }],
      ),
    ).toEqual([
      { key: "proNumber", label: "Pro number", type: "text" },
      { key: "status", label: "Status", type: "status" },
    ]);
  });

  it("reads a nested set of sentences field by field", () => {
    expect(
      projectRecord({
        id: "inst_01M37R101VKZTB7TSKR30FJ0AT",
        headline: "3 cards expire",
        rule: { threshold: "14 days", measures: "Days until the card lapses" },
      }).map((field) => [field.key, field.label]),
    ).toEqual([
      ["headline", "Headline"],
      ["ruleMeasures", "Rule measures"],
      ["ruleThreshold", "Rule threshold"],
    ]);
  });
});
