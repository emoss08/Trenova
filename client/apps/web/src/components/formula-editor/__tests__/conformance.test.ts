import { describe, expect, it } from "vitest";
import corpus from "../../../../../../../services/tms/internal/modules/rating/app/formula/testdata/conformance.json";
import { checkExpression, engineErrorRange } from "../expr-check";
import { buildKnownIdentifiers } from "../known-identifiers";
import { describedShipment } from "./described-shipment";
import { variableDefinitionSchema } from "@trenova/shared/types/formula-template";

type ConformanceCase = {
  id: string;
  expression: string;
  accepted: boolean;
  serverError?: string;
  errorFrom?: number;
};

const cases: ConformanceCase[] = corpus.cases;

const known = buildKnownIdentifiers(
  describedShipment,
  corpus.inputs.map((input) => variableDefinitionSchema.parse(input)),
);

describe("the editor agrees with the engine on what saves", () => {
  it.each(cases.map((tc) => [tc.id, tc] as const))("%s", (_id, tc) => {
    const errors = checkExpression(tc.expression, known).diagnostics.filter(
      (diagnostic) => diagnostic.severity === "error",
    );
    const messages = errors.map((error) => `${error.code}: ${error.message}`);

    if (tc.accepted) {
      expect(messages, `the engine saves ${tc.expression}`).toEqual([]);
    } else {
      expect(messages.length, `the engine refuses ${tc.expression}`).toBeGreaterThan(0);
    }
  });
});

describe("the editor points where the engine's error does", () => {
  const positioned = cases.filter((tc) => tc.serverError !== undefined);

  it.each(positioned.map((tc) => [tc.id, tc] as const))("%s", (_id, tc) => {
    const message = `schema error for '${tc.expression}' during compile: ${tc.serverError}`;
    expect(engineErrorRange(message, tc.expression)?.from).toBe(tc.errorFrom);
  });
});
