import { formatMessage } from "@trenova/shared/i18n/format-message";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import {
  formulaStage,
  formulaStatusLabel,
  totalClauseLabel,
  usageLabel,
} from "../formula-template-card";

const t: TranslateFn = (message, ...args) => formatMessage("en", message ?? "", args);

describe("formulaStage", () => {
  it("reads every status", () => {
    expect(formulaStage("Active")).toBe("active");
    expect(formulaStage("InReview")).toBe("review");
    expect(formulaStage("Inactive")).toBe("inactive");
    expect(formulaStage("Draft")).toBe("draft");
    expect(formulaStatusLabel("InReview", t)).toBe("In review");
  });
});

describe("usageLabel", () => {
  it("counts each kind of use the way a person says it", () => {
    expect(usageLabel({ type: "shipment", count: 1 }, t)).toBe("1 shipment");
    expect(usageLabel({ type: "shipment", count: 3 }, t)).toBe("3 shipments");
    expect(usageLabel({ type: "rate_agreement_rule", count: 1 }, t)).toBe("1 agreement rule");
    expect(usageLabel({ type: "rate_agreement", count: 2 }, t)).toBe("2 rate agreements");
    expect(usageLabel({ type: "billing_control_fallback", count: 1 }, t)).toBe(
      "the billing fallback",
    );
    expect(usageLabel({ type: "invoice_line", count: 4 }, t)).toBe("4 other uses");
  });
});

describe("totalClauseLabel", () => {
  it("names each total clause the outline lists", () => {
    expect(totalClauseLabel({ name: "minimum" }, t)).toBe("Minimum charge");
    expect(totalClauseLabel({ name: "maximum" }, t)).toBe("Maximum charge");
    expect(totalClauseLabel({ name: "round" }, t)).toBe("Rounding");
  });
});
