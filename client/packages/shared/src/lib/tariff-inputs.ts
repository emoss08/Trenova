// tariff-inputs.ts is what a contract form knows about a tariff's inputs besides drawing them:
// what each is called, which a form may leave unset, which a whole agreement can set for every
// lane, and whether the values typed so far fit their declared types. The module and the server
// stay the authority (`tariff.check` validates every saved value); this lets a form say what is
// wrong at the field before it asks.

import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import {
  tariffConstraintMessage,
  tariffConstraintViolations,
} from "@trenova/shared/lib/tariff-constraint";
import { tariffValueProblems, type TariffValueProblem } from "@trenova/shared/rules/tariff-wire";
import type {
  TariffInputDeclaration,
  TariffInputValues,
  TariffType,
  TariffWireValue,
} from "@trenova/shared/types/tariff";
import { z } from "zod";

/** `extraDropFee` as a person reads it: `Extra drop fee`. */
export function humanizeTariffName(name: string): string {
  const words = name
    .replace(/([a-z0-9])([A-Z])/g, "$1 $2")
    .replace(/([A-Z]+)([A-Z][a-z])/g, "$1 $2")
    .replace(/_/g, " ")
    .trim()
    .split(/\s+/)
    .filter(Boolean);
  if (words.length === 0) return name;
  return words
    .map((word, index) => {
      if (/^[A-Z0-9]+$/.test(word) && word.length > 1) return word;
      const lower = word.toLowerCase();
      return index === 0 ? lower.charAt(0).toUpperCase() + lower.slice(1) : lower;
    })
    .join(" ");
}

/**
 * The label and description an input's doc comment gives it: the doc's first sentence labels
 * the field and the rest describes it. An input with no doc is labelled by its name. Doc
 * comments are the tariff author's words, shown as written.
 */
export function tariffInputText(declaration: { name: string; doc: string | null }): {
  label: string;
  description: string | null;
} {
  const doc = declaration.doc?.trim() ?? "";
  if (!doc) return { label: humanizeTariffName(declaration.name), description: null };
  const match = /^(.+?[.!?])(\s+|$)/s.exec(doc);
  const first = (match?.[1] ?? doc).replace(/\s+/g, " ").trim();
  const rest = match ? doc.slice(match[0].length).replace(/\s+/g, " ").trim() : "";
  return {
    label: first.replace(/[.]$/, ""),
    description: rest || null,
  };
}

function sameType(a: TariffType, b: TariffType): boolean {
  return JSON.stringify(a) === JSON.stringify(b);
}

/**
 * The inputs every one of the given tariffs declares with the same type: the ones an agreement
 * can set once for all its lanes. A value set on the agreement is checked against every lane's
 * tariff, so an input some lane's tariff lacks (or types differently) is not one the agreement
 * may set. Declarations keep the first tariff's order; a default is kept only when every tariff
 * declares the same one.
 */
export function sharedTariffInputs(
  sets: readonly (readonly TariffInputDeclaration[])[],
): TariffInputDeclaration[] {
  const [first, ...others] = sets;
  if (!first) return [];
  return first.flatMap((declaration) => {
    const matches = others.map((set) => set.find((other) => other.name === declaration.name));
    if (matches.some((match) => !match || !sameType(match.descriptor, declaration.descriptor))) {
      return [];
    }
    const sameDefault = matches.every(
      (match) => JSON.stringify(match?.default) === JSON.stringify(declaration.default),
    );
    return [
      {
        ...declaration,
        required: declaration.required || matches.some((match) => match?.required),
        default: sameDefault ? declaration.default : undefined,
      },
    ];
  });
}

/** The value an unset input takes: the inherited one (the agreement's, under a rule), else its default. */
export function inheritedTariffValue(
  declaration: TariffInputDeclaration,
  inherited: TariffInputValues | undefined,
): TariffWireValue | undefined {
  return inherited?.[declaration.name] ?? declaration.default;
}

function problemMessage(problem: TariffValueProblem, t: TranslateFn): string {
  switch (problem.problem) {
    case "missing":
      return t("Required");
    case "not-bool":
      return t("Must be yes or no");
    case "not-text":
      return t("Must be text");
    case "not-number":
      return t("Must be a number");
    case "not-int":
      return t("Must be a whole number");
    case "not-money":
      return t("Must be an amount of money");
    case "wrong-unit":
      return t("Must be a quantity in a unit of {0}", problem.detail ?? "");
    case "not-percent":
      return t("Must be a percent");
    case "not-date":
      return t("Must be a date");
    case "not-date-time":
      return t("Must be a date and time");
    case "not-time":
      return t("Must be a time of day");
    case "not-variant":
      return t("Must be one of {0}", problem.detail ?? "");
    case "not-list":
      return t("Must be a list");
    case "not-record":
      return t("Must have every field filled in");
    case "unknown-field":
      return t("{0} is not a field of this input", problem.detail ?? "");
  }
}

/** `[2].rate` as a form path's segments: `[2, "rate"]`. */
function pathSegments(path: string): (string | number)[] {
  const segments: (string | number)[] = [];
  for (const match of path.matchAll(/\[(\d+)\]|\.?([^.[\]]+)/g)) {
    if (match[1] !== undefined) segments.push(Number(match[1]));
    else if (match[2] !== undefined) segments.push(match[2]);
  }
  return segments;
}

/**
 * Money per pallet, piece or stop is money (a count folds into the value), and the module
 * writes and accepts it in that unit (`{"$q": "5", "u": "$/pallet"}`); the shared value check
 * knows only `$`, so its complaint about such a value is dropped here.
 */
function isFoldedMoney(problem: TariffValueProblem, value: TariffWireValue): boolean {
  if (problem.problem !== "not-money") return false;
  let at: unknown = value;
  for (const segment of pathSegments(problem.path)) {
    at = (at as Record<string | number, unknown> | null | undefined)?.[segment];
  }
  return (
    typeof at === "object" &&
    at !== null &&
    "u" in at &&
    /^\$\/(pallet|piece|stop)$/.test(String((at as { u: unknown }).u))
  );
}

export type TariffInputIssue = { path: (string | number)[]; message: string };

export type TariffInputCheckOptions = {
  /** Values an unset input inherits (the agreement's, for a rule). */
  inherited?: TariffInputValues;
  /**
   * Whether an input with no default that nothing sets is an error here. An agreement's
   * inputs are never required: its lanes may set them instead.
   */
  requireUnset?: boolean;
  /**
   * Whose values these are. An agreement's are checked against the inputs its lanes share,
   * so a name outside them is one some lane's tariff lacks rather than an unknown input.
   */
  scope?: "agreement";
  /** The rating currency's ISO code, for the money a constraint's message names. */
  currency?: string | null;
};

/**
 * What is wrong with a form's input values against the tariff's declarations: values that do
 * not fit their type, values outside an input's `where` bounds (each item of a list input),
 * values for inputs the tariff does not declare, and (when asked) inputs no one sets that have
 * no default. Paths are relative to the values record.
 */
export function tariffInputIssues(
  declarations: readonly TariffInputDeclaration[],
  values: Readonly<Record<string, unknown>> | null | undefined,
  t: TranslateFn,
  options: TariffInputCheckOptions = {},
): TariffInputIssue[] {
  const issues: TariffInputIssue[] = [];
  const set = values ?? {};
  const known = new Set(declarations.map((declaration) => declaration.name));
  for (const declaration of declarations) {
    const value = set[declaration.name] as TariffWireValue | undefined;
    if (value === undefined) {
      const unset =
        declaration.required && inheritedTariffValue(declaration, options.inherited) === undefined;
      if (options.requireUnset && unset) {
        issues.push({ path: [declaration.name], message: t("Required: nothing else sets it") });
      }
      continue;
    }
    const problems = tariffValueProblems(declaration.descriptor, value).filter(
      (found) => !isFoldedMoney(found, value),
    );
    for (const problem of problems) {
      issues.push({
        path: [declaration.name, ...pathSegments(problem.path)],
        message: problemMessage(problem, t),
      });
    }
    if (problems.length > 0 || !declaration.constraint) continue;
    const message = tariffConstraintMessage(declaration.constraint, t, {
      currency: options.currency,
    });
    for (const index of tariffConstraintViolations(
      declaration.constraint,
      declaration.descriptor,
      value,
    )) {
      issues.push({ path: index < 0 ? [declaration.name] : [declaration.name, index], message });
    }
  }
  for (const name of Object.keys(set)) {
    if (!known.has(name) && set[name] !== undefined) {
      issues.push({
        path: [name],
        message:
          options.scope === "agreement"
            ? t("Not every lane's tariff has an input named {0} of this type", name)
            : t("This tariff has no input named {0}", name),
      });
    }
  }
  return issues;
}

/**
 * A zod schema for one tariff's input values, built from its declarations: every value must
 * fit its declared type, and with `requireUnset` every input without a default must be set
 * here or inherited.
 */
export function tariffInputValuesSchemaFor(
  declarations: readonly TariffInputDeclaration[],
  t: TranslateFn,
  options: TariffInputCheckOptions = {},
): z.ZodType<Record<string, unknown>> {
  return z.record(z.string(), z.unknown()).superRefine((values, ctx) => {
    for (const issue of tariffInputIssues(declarations, values, t, options)) {
      ctx.addIssue({ code: "custom", path: issue.path, message: issue.message });
    }
  });
}

/** Adds the issues of one input values record to a parent schema's refinement, under `at`. */
export function addTariffInputIssues(
  ctx: z.RefinementCtx,
  at: (string | number)[],
  declarations: readonly TariffInputDeclaration[],
  values: Readonly<Record<string, unknown>> | null | undefined,
  t: TranslateFn,
  options: TariffInputCheckOptions = {},
): void {
  for (const issue of tariffInputIssues(declarations, values, t, options)) {
    ctx.addIssue({ code: "custom", path: [...at, ...issue.path], message: issue.message });
  }
}

/**
 * The refinement option that checks input values even while other fields of the form fail, so
 * a person sees every problem at once; it needs only an object to read.
 */
export const TARIFF_INPUTS_REFINEMENT = {
  when: (payload: { value: unknown }) =>
    typeof payload.value === "object" && payload.value !== null,
};
