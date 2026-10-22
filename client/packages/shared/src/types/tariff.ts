// tariff.ts types what crosses the boundary between Tariff (docs/engineering/formula-language.md),
// the rules module that implements it, the server and the browser: wire values, the input type
// descriptors `tariff.check` derives from `input` declarations, the bill, refusals, evaluation
// errors, diagnostics, test results and the trace. The server stores and returns these same
// shapes (a template version's `inputs`, a preview's `bill`), so one schema parses both the
// module's answers and the API's.
//
// Positions are UTF-16 code units, half-open, as CodeMirror counts them. Numbers are decimal
// strings and are never read through a float.

import type { CatalogMessage, CatalogMessageArg } from "@trenova/shared/i18n/message";
import { z } from "zod";

/** User-facing text the module wrote: an English ICU template (the catalog key) and its args. */
export type TariffMessage = CatalogMessage;

const tariffMessageArgSchema: z.ZodType<CatalogMessageArg> = z.lazy(() =>
  z.union([z.string(), z.number(), tariffMessageSchema]),
);

export const tariffMessageSchema: z.ZodType<CatalogMessage> = z.object({
  message: z.string(),
  args: z.array(tariffMessageArgSchema).optional(),
});

/** A range of a source file, in UTF-16 code units. */
export const tariffRangeSchema = z.object({
  from: z.number().int().nonnegative(),
  to: z.number().int().nonnegative(),
});
export type TariffRange = z.output<typeof tariffRangeSchema>;

/** The three kinds of Tariff file, named on the file's first line. */
export const tariffFileKindSchema = z.enum(["tariff", "library", "condition"]);
export type TariffFileKind = z.output<typeof tariffFileKindSchema>;

/** `Int` and `Number`: `{"$n": "1200"}`. */
export type TariffWireNumber = { $n: string };
/** Money, rates, quantities and percents: `{"$q": "2.35", "u": "$/mi"}`. */
export type TariffWireQuantity = { $q: string; u: string };
/** `Date` and `DateTime`: `{"$date": "2026-10-22"}`, `{"$date": "2026-10-22T08:00:00-05:00"}`. */
export type TariffWireDate = { $date: string };
/** A record: field names never start with `$`. */
export type TariffWireRecord = { [field: string]: TariffWireValue };

/**
 * A value as Tariff sends it (the reference's "Wire values"): `Bool` and `Text` as JSON, an enum
 * variant as its name, `none` as `null`, lists and records as arrays and objects, and every
 * number tagged with its exact decimal text.
 */
export type TariffWireValue =
  | boolean
  | string
  | null
  | TariffWireNumber
  | TariffWireQuantity
  | TariffWireDate
  | TariffWireValue[]
  | TariffWireRecord;

/** Exact decimal text: an optional minus, digits, and an optional fraction. */
export const TARIFF_DECIMAL_PATTERN = /^-?\d+(\.\d+)?$/;

const decimalTextSchema = z.string().regex(TARIFF_DECIMAL_PATTERN);

export const tariffWireNumberSchema = z.strictObject({ $n: decimalTextSchema });
export const tariffWireQuantitySchema = z.strictObject({
  $q: decimalTextSchema,
  u: z.string().min(1),
});
export const tariffWireDateSchema = z.strictObject({ $date: z.string().min(10) });

const tariffWireRecordSchema: z.ZodType<TariffWireRecord> = z.lazy(() =>
  z
    .record(z.string(), tariffWireValueSchema)
    .refine((record) => Object.keys(record).every((key) => !key.startsWith("$")), {
      error: "A record field name cannot start with $",
    }),
);

export const tariffWireValueSchema: z.ZodType<TariffWireValue> = z.lazy(() =>
  z.union([
    z.boolean(),
    z.string(),
    z.null(),
    tariffWireNumberSchema,
    tariffWireQuantitySchema,
    tariffWireDateSchema,
    z.array(tariffWireValueSchema),
    tariffWireRecordSchema,
  ]),
);

/**
 * A record field of a record type descriptor. Its doc is a catalog message when the standard library wrote it (shown with
 * `translateMessage`) and plain text, shown as written, when an organisation library did.
 */
export type TariffTypeField = {
  name: string;
  doc: string | TariffMessage | null;
  type: TariffType;
};

/**
 * A type as `tariff.check` describes an input's, for building its form: the scalar kinds,
 * `money`, a `quantity` of a dimension shown in a unit (`Money/Distance` in `$/mi`, `Weight` in
 * `lb`), a `list`, an `optional`, a `record` with its fields, an `enum` with its variants, and
 * `unknown` for a type the checker could not read.
 */
export type TariffType =
  | {
      kind:
        | "bool"
        | "int"
        | "number"
        | "percent"
        | "text"
        | "date"
        | "dateTime"
        | "money"
        | "unknown";
    }
  | { kind: "quantity"; dimension: string; unit: string }
  | { kind: "list"; item: TariffType }
  | { kind: "optional"; inner: TariffType }
  | { kind: "record"; name: string | null; fields: TariffTypeField[] }
  | { kind: "enum"; name: string; variants: string[] };

export const tariffTypeSchema: z.ZodType<TariffType> = z.lazy(() =>
  z.discriminatedUnion("kind", [
    z.object({
      kind: z.enum([
        "bool",
        "int",
        "number",
        "percent",
        "text",
        "date",
        "dateTime",
        "money",
        "unknown",
      ]),
    }),
    z.object({ kind: z.literal("quantity"), dimension: z.string(), unit: z.string() }),
    z.object({ kind: z.literal("list"), item: tariffTypeSchema }),
    z.object({ kind: z.literal("optional"), inner: tariffTypeSchema }),
    z.object({
      kind: z.literal("record"),
      name: z
        .string()
        .nullish()
        .transform((value) => value ?? null),
      fields: z.array(
        z.object({
          name: z.string(),
          doc: z
            .union([z.string(), tariffMessageSchema])
            .nullish()
            .transform((value) => value ?? null),
          type: tariffTypeSchema,
        }),
      ),
    }),
    z.object({ kind: z.literal("enum"), name: z.string(), variants: z.array(z.string()) }),
  ]),
);

/**
 * One `input` of a tariff or condition, as `tariff.check` derives it and a template version
 * stores it: the name, the doc comment (the form's label), the type as written (`Money/mi`) and
 * as a descriptor, whether nothing supplies a default, and the default as a wire value.
 */
export const tariffInputDeclarationSchema = z.object({
  name: z.string(),
  doc: z
    .string()
    .nullish()
    .transform((value) => value ?? null),
  type: z.string(),
  descriptor: tariffTypeSchema,
  required: z.boolean(),
  default: tariffWireValueSchema.optional(),
  range: tariffRangeSchema.nullish(),
});
export type TariffInputDeclaration = z.output<typeof tariffInputDeclarationSchema>;

export const tariffInputDeclarationsSchema = z
  .array(tariffInputDeclarationSchema)
  .nullish()
  .transform((value) => value ?? []);

/** Input values a contract, a shipment or a preview sets, by input name, in wire form. */
export const tariffInputValuesSchema = z.record(z.string(), tariffWireValueSchema);
export type TariffInputValues = z.output<typeof tariffInputValuesSchema>;

/** A library a template version imports, pinned to the library version it was approved with. */
export const tariffImportSchema = z.object({
  path: z.string(),
  version: z
    .string()
    .nullish()
    .transform((value) => value ?? null),
});
export type TariffImport = z.output<typeof tariffImportSchema>;

/** What a bill line is: a charge the tariff billed, or an adjustment a total clause made. */
export const tariffLineKindSchema = z.enum(["charge", "minimum", "maximum", "rounding"]);
export type TariffLineKind = z.output<typeof tariffLineKindSchema>;

/**
 * One line of a bill. A charge has the author's label (the customer's text, shown as written)
 * and a code when it is an accessorial; an adjustment has neither and is labelled in the
 * reader's language. `quantity` and `rate` are there when the charge gave both.
 */
export const tariffBillLineSchema = z.object({
  kind: tariffLineKindSchema,
  label: z
    .string()
    .nullish()
    .transform((value) => value ?? null),
  code: z
    .string()
    .nullish()
    .transform((value) => value ?? null),
  freight: z.boolean(),
  amount: tariffWireQuantitySchema,
  quantity: tariffWireValueSchema.nullish().transform((value) => value ?? null),
  rate: tariffWireValueSchema.nullish().transform((value) => value ?? null),
  range: tariffRangeSchema.nullish().transform((value) => value ?? null),
});
export type TariffBillLine = z.output<typeof tariffBillLineSchema>;

export const tariffNoteSchema = z.object({
  text: z.string(),
  range: tariffRangeSchema.nullish().transform((value) => value ?? null),
});
export type TariffNote = z.output<typeof tariffNoteSchema>;

/** The bill in the billing currency, when it differs from the rating currency. */
export const tariffConvertedSchema = z.object({
  currency: z.string(),
  rate: decimalTextSchema,
  lines: z.array(tariffWireQuantitySchema),
  total: tariffWireQuantitySchema,
});
export type TariffConverted = z.output<typeof tariffConvertedSchema>;

/** A tariff's result: its lines in order, its notes and its totals. Lines add up to `total`. */
export const tariffBillSchema = z.object({
  lines: z.array(tariffBillLineSchema),
  notes: z
    .array(tariffNoteSchema)
    .nullish()
    .transform((value) => value ?? []),
  freightTotal: tariffWireQuantitySchema,
  accessorialTotal: tariffWireQuantitySchema,
  total: tariffWireQuantitySchema,
  converted: tariffConvertedSchema.nullish().transform((value) => value ?? null),
});
export type TariffBill = z.output<typeof tariffBillSchema>;

/**
 * Why a tariff refused to rate: the author's own words from `fail` or `require` (text, shown as
 * written), or a message the module wrote (an input nothing set), and where.
 */
export const tariffRefusalSchema = z.object({
  message: z.union([z.string(), tariffMessageSchema]),
  range: tariffRangeSchema.nullish().transform((value) => value ?? null),
});
export type TariffRefusal = z.output<typeof tariffRefusalSchema>;

/** Why evaluation failed (a budget, a division by zero, an input that does not fit), and where. */
export const tariffEvalErrorSchema = z.object({
  code: z.string(),
  message: tariffMessageSchema,
  range: tariffRangeSchema.nullish().transform((value) => value ?? null),
});
export type TariffEvalError = z.output<typeof tariffEvalErrorSchema>;

export const tariffSeveritySchema = z.enum(["error", "warning", "info"]);
export type TariffSeverity = z.output<typeof tariffSeveritySchema>;

/** One text change of a fix. */
export const tariffChangeSchema = z.object({ range: tariffRangeSchema, insert: z.string() });
export type TariffChange = z.output<typeof tariffChangeSchema>;

/** A fix: source edits that resolve the finding. */
export const tariffFixSchema = z.object({
  label: tariffMessageSchema,
  changes: z.array(tariffChangeSchema),
});
export type TariffFix = z.output<typeof tariffFixSchema>;

export const tariffRelatedSchema = z.object({
  range: tariffRangeSchema,
  message: tariffMessageSchema,
});
export type TariffRelated = z.output<typeof tariffRelatedSchema>;

/**
 * A finding of `tariff.check` (and of the server, which runs the same check). `module` names
 * the imported library a finding is in (its range is in that library's source); `path` names
 * the contract input value an `input-value` finding is about, which has no range.
 */
export const tariffDiagnosticSchema = z.object({
  range: tariffRangeSchema.nullish().transform((value) => value ?? null),
  severity: tariffSeveritySchema,
  code: z.string(),
  message: tariffMessageSchema,
  fixes: z
    .array(tariffFixSchema)
    .nullish()
    .transform((value) => value ?? []),
  related: z
    .array(tariffRelatedSchema)
    .nullish()
    .transform((value) => value ?? []),
  module: z
    .string()
    .nullish()
    .transform((value) => value ?? null),
  path: z
    .string()
    .nullish()
    .transform((value) => value ?? null),
});
export type TariffDiagnostic = z.output<typeof tariffDiagnosticSchema>;

export const tariffDiagnosticsSchema = z
  .array(tariffDiagnosticSchema)
  .nullish()
  .transform((value) => value ?? []);

/** What an outline entry is. */
export const tariffOutlineKindSchema = z.enum([
  "header",
  "input",
  "const",
  "type",
  "enum",
  "function",
  "charge",
  "total",
  "test",
]);
export type TariffOutlineKind = z.output<typeof tariffOutlineKindSchema>;

export const tariffOutlineItemSchema = z.object({
  kind: tariffOutlineKindSchema,
  name: z.string(),
  code: z
    .string()
    .nullish()
    .transform((value) => value ?? null),
  depth: z.number().int().nonnegative(),
  range: tariffRangeSchema,
  selection: tariffRangeSchema,
});
export type TariffOutlineItem = z.output<typeof tariffOutlineItemSchema>;

export const tariffFoldSchema = z.object({
  range: tariffRangeSchema,
  kind: z.enum(["block", "record", "list", "doc"]),
});
export type TariffFold = z.output<typeof tariffFoldSchema>;

/** The file's outline (inputs, functions, charges, totals, tests) and its folding ranges. */
export const tariffOutlineSchema = z.object({
  items: z.array(tariffOutlineItemSchema),
  folding: z.array(tariffFoldSchema),
});
export type TariffOutline = z.output<typeof tariffOutlineSchema>;

export const tariffChargeRefSchema = z.object({
  label: z.string(),
  code: z
    .string()
    .nullish()
    .transform((value) => value ?? null),
  range: tariffRangeSchema,
});
export type TariffChargeRef = z.output<typeof tariffChargeRefSchema>;

/** What `tariff.check` answers. `ok` is true when no finding is an error. */
export const tariffCheckResultSchema = z.object({
  ok: z.boolean(),
  header: z
    .object({ kind: tariffFileKindSchema, name: z.string() })
    .nullish()
    .transform((value) => value ?? null),
  diagnostics: tariffDiagnosticsSchema,
  inputs: tariffInputDeclarationsSchema,
  imports: z
    .array(tariffImportSchema)
    .nullish()
    .transform((value) => value ?? []),
  outline: tariffOutlineSchema.nullish().transform((value) => value ?? null),
  charges: z
    .array(tariffChargeRefSchema)
    .nullish()
    .transform((value) => value ?? []),
});
export type TariffCheckResult = z.output<typeof tariffCheckResultSchema>;

/** A value the trace or a test failure shows: its wire form and the text interpolation writes. */
export const tariffShownValueSchema = z.object({
  range: tariffRangeSchema,
  display: z.string(),
});
export type TariffShownValue = z.output<typeof tariffShownValueSchema>;

export const tariffTestFailureSchema = z.object({
  range: tariffRangeSchema,
  message: tariffMessageSchema,
  values: z
    .array(tariffShownValueSchema)
    .nullish()
    .transform((value) => value ?? []),
});
export type TariffTestFailure = z.output<typeof tariffTestFailureSchema>;

export const tariffTestStatusSchema = z.enum(["passed", "failed", "error"]);
export type TariffTestStatus = z.output<typeof tariffTestStatusSchema>;

/** One `test` block's result: the failing expectations with their values, and its bill. */
export const tariffTestResultSchema = z.object({
  name: z.string(),
  range: tariffRangeSchema,
  status: tariffTestStatusSchema,
  failures: z
    .array(tariffTestFailureSchema)
    .nullish()
    .transform((value) => value ?? []),
  bill: tariffBillSchema.nullish().transform((value) => value ?? null),
  error: tariffEvalErrorSchema.nullish().transform((value) => value ?? null),
});
export type TariffTestResult = z.output<typeof tariffTestResultSchema>;

/** What `tariff.test` answers; `error` is set when the file does not load, and no test ran. */
export const tariffTestRunSchema = z.object({
  tests: z.array(tariffTestResultSchema),
  error: tariffEvalErrorSchema.nullish().transform((value) => value ?? null),
});
export type TariffTestRun = z.output<typeof tariffTestRunSchema>;

const traceValueFields = {
  value: tariffWireValueSchema.optional(),
  display: z.string(),
};

/** One thing the tariff did, in execution order, for the preview's trace. */
export const tariffTraceEventSchema = z.discriminatedUnion("kind", [
  z.object({
    kind: z.enum(["bind", "assign"]),
    name: z.string(),
    ...traceValueFields,
    range: tariffRangeSchema,
  }),
  z.object({
    kind: z.literal("branch"),
    taken: z.number().int().nullable(),
    range: tariffRangeSchema,
  }),
  z.object({ kind: z.literal("arm"), arm: z.number().int(), range: tariffRangeSchema }),
  z.object({
    kind: z.literal("loop"),
    name: z.string(),
    iterations: z.number().int(),
    range: tariffRangeSchema,
  }),
  z.object({
    kind: z.literal("iteration"),
    name: z.string(),
    index: z.number().int(),
    ...traceValueFields,
    range: tariffRangeSchema,
  }),
  z.object({
    kind: z.literal("charge"),
    label: z.string(),
    code: z.string().nullable(),
    ...traceValueFields,
    range: tariffRangeSchema,
  }),
  z.object({
    kind: z.literal("dropped"),
    label: z.string(),
    code: z.string().nullable(),
    range: tariffRangeSchema.nullable(),
  }),
  z.object({ kind: z.literal("note"), text: z.string(), range: tariffRangeSchema }),
  z.object({ kind: z.literal("refusal"), message: z.string(), range: tariffRangeSchema }),
  z.object({ kind: z.literal("lookup"), lookup: z.number().int(), range: tariffRangeSchema }),
  z.object({ kind: z.literal("truncated") }),
]);
export type TariffTraceEvent = z.output<typeof tariffTraceEventSchema>;

/** A rate table read: the keys, the cell or band it matched (or none) and the value. */
export const tariffLookupSchema = z.object({
  table: z.string(),
  op: z.string(),
  keys: z.array(tariffWireValueSchema),
  match: z
    .array(tariffWireValueSchema)
    .nullish()
    .transform((value) => value ?? null),
  adjusted: z.boolean(),
  value: tariffWireValueSchema.optional(),
  display: z.string(),
  range: tariffRangeSchema,
});
export type TariffLookup = z.output<typeof tariffLookupSchema>;

/** How a rating ended. `needs` means the host must send more rate table cells and ask again. */
export const tariffRatingStatusSchema = z.enum(["rated", "refused", "error", "needs"]);
export type TariffRatingStatus = z.output<typeof tariffRatingStatusSchema>;

/** One program's result from `tariff.evaluate`. */
export const tariffEvaluationResultSchema = z.object({
  id: z.string(),
  status: tariffRatingStatusSchema,
  bill: tariffBillSchema.nullish().transform((value) => value ?? null),
  value: z
    .boolean()
    .nullish()
    .transform((value) => value ?? null),
  refusal: tariffRefusalSchema.nullish().transform((value) => value ?? null),
  error: tariffEvalErrorSchema.nullish().transform((value) => value ?? null),
  needs: z
    .array(z.object({ table: z.string(), keys: z.array(z.string()) }))
    .nullish()
    .transform((value) => value ?? []),
  trace: z
    .array(tariffTraceEventSchema)
    .nullish()
    .transform((value) => value ?? null),
  lookups: z
    .array(tariffLookupSchema)
    .nullish()
    .transform((value) => value ?? []),
});
export type TariffEvaluationResult = z.output<typeof tariffEvaluationResultSchema>;

export const tariffEvaluationSchema = z.object({ results: z.array(tariffEvaluationResultSchema) });
export type TariffEvaluation = z.output<typeof tariffEvaluationSchema>;

/** An organisation library a file may import with `use org.<name>`, pinned to a version. */
export const tariffLibrarySourceSchema = z.object({
  path: z.string(),
  version: z
    .string()
    .nullish()
    .transform((value) => value ?? undefined),
  source: z.string(),
});
export type TariffLibrarySource = z.output<typeof tariffLibrarySourceSchema>;

/** One axis of a rate table: `Exact` or `Range`, what it varies by, and a range's unit. */
export const tariffTableAxisSchema = z.object({
  match: z.enum(["Exact", "Range"]),
  kind: z.string(),
  label: z
    .string()
    .nullish()
    .transform((value) => value ?? ""),
  unit: z
    .string()
    .nullish()
    .transform((value) => value ?? undefined),
});
export type TariffTableAxis = z.output<typeof tariffTableAxisSchema>;

/** A rate matrix as a tariff reads it, `tables.<code>`: its axes and the type of its cells. */
export const tariffTableShapeSchema = z.object({
  code: z.string(),
  name: z
    .string()
    .nullish()
    .transform((value) => value ?? ""),
  axes: z.array(tariffTableAxisSchema),
  valueType: z.string(),
});
export type TariffTableShape = z.output<typeof tariffTableShapeSchema>;

/** An accessorial charge code a tariff may bill with `charge … as CODE`. */
export const tariffChargeCodeSchema = z.object({
  code: z.string(),
  description: z
    .string()
    .nullish()
    .transform((value) => value ?? ""),
  active: z.boolean(),
});
export type TariffChargeCode = z.output<typeof tariffChargeCodeSchema>;

/**
 * What every `tariff.*` operation that reads a file is given besides the file, and what
 * `GET /formula-templates/context` answers: the libraries it may import, the rate tables' shapes,
 * the accessorial charge codes (null while unknown, so codes go unchecked) and the settings.
 */
export const tariffContextSchema = z.object({
  libraries: z
    .array(tariffLibrarySourceSchema)
    .nullish()
    .transform((value) => value ?? []),
  tables: z
    .array(tariffTableShapeSchema)
    .nullish()
    .transform((value) => value ?? []),
  codes: z
    .array(tariffChargeCodeSchema)
    .nullish()
    .transform((value) => value ?? null),
  settings: z
    .object({
      distanceUnit: z
        .string()
        .nullish()
        .transform((value) => value ?? undefined),
    })
    .nullish()
    .transform((value) => value ?? {}),
});
export type TariffContext = z.output<typeof tariffContextSchema>;

/** An approved organisation library the context holds, as the Studio names it. */
export const tariffStudioLibrarySchema = z.object({
  /** The module path a file imports: `org.<name>`. */
  path: z.string(),
  libraryId: z.string(),
  name: z.string(),
  description: z
    .string()
    .nullish()
    .transform((value) => value ?? ""),
  /** The approved version the context holds, the one a new template version would pin. */
  versionNumber: z.number().int(),
});
export type TariffStudioLibrary = z.output<typeof tariffStudioLibrarySchema>;

/**
 * What `GET /formula-templates/context` answers: the context every check is given, and each
 * library in it with its name, description and approved version number.
 */
export const tariffStudioContextSchema = z.object({
  context: tariffContextSchema,
  libraries: z
    .array(tariffStudioLibrarySchema)
    .nullish()
    .transform((value) => value ?? []),
});
export type TariffStudioContext = z.output<typeof tariffStudioContextSchema>;

/** The context of a file checked before the organisation's context has loaded. */
export const EMPTY_TARIFF_CONTEXT: TariffContext = {
  libraries: [],
  tables: [],
  codes: null,
  settings: {},
};
