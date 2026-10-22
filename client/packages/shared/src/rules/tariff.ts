// tariff.ts wraps the rules module's `tariff.*` operations (docs/engineering/formula-language.md,
// "Module operations"). The module is the language: it parses, checks, formats, highlights,
// evaluates and runs tests, and the browser only draws what it answers. Every call is synchronous
// on a loaded runtime (loadRules(), rulesSync(), useRules()), every position in and out is a
// UTF-16 code unit (what CodeMirror counts), and every answer is validated, so a module that
// answers with a shape this client does not expect fails loudly at the boundary instead of
// drawing nonsense.
//
// The catalog is static per module, so it is read once per loaded runtime.

import {
  tariffCheckResultSchema,
  tariffDiagnosticsSchema,
  tariffEvalErrorSchema,
  tariffMessageSchema,
  tariffRangeSchema,
  tariffTestRunSchema,
  tariffTypeSchema,
  tariffWireValueSchema,
  type TariffCheckResult,
  type TariffContext,
  type TariffFileKind,
  type TariffInputValues,
  type TariffTestRun,
  type TariffTraceEvent,
  type TariffWireValue,
} from "@trenova/shared/types/tariff";
import { z } from "zod";
import type { RulesRuntime } from "./runtime";

export const TARIFF_CHECK_OP = "tariff.check";
export const TARIFF_EVALUATE_OP = "tariff.evaluate";
export const TARIFF_TEST_OP = "tariff.test";
export const TARIFF_FORMAT_OP = "tariff.format";
export const TARIFF_TOKENS_OP = "tariff.tokens";
export const TARIFF_COMPLETE_OP = "tariff.complete";
export const TARIFF_HOVER_OP = "tariff.hover";
export const TARIFF_SIGNATURE_OP = "tariff.signature";
export const TARIFF_DEFINITION_OP = "tariff.definition";
export const TARIFF_RENAME_OP = "tariff.rename";
export const TARIFF_INLAYS_OP = "tariff.inlays";
export const TARIFF_CATALOG_OP = "tariff.catalog";
export const TARIFF_SCAFFOLD_TEST_OP = "tariff.scaffoldTest";

/** Every `tariff.*` operation the module answers. */
export const TARIFF_OPERATIONS = [
  TARIFF_CHECK_OP,
  TARIFF_EVALUATE_OP,
  TARIFF_TEST_OP,
  TARIFF_FORMAT_OP,
  TARIFF_TOKENS_OP,
  TARIFF_COMPLETE_OP,
  TARIFF_HOVER_OP,
  TARIFF_SIGNATURE_OP,
  TARIFF_DEFINITION_OP,
  TARIFF_RENAME_OP,
  TARIFF_INLAYS_OP,
  TARIFF_CATALOG_OP,
  TARIFF_SCAFFOLD_TEST_OP,
] as const;

/** A JSON value the module reads: `undefined` fields are left out, as JSON.stringify does. */
function wireContext(context: TariffContext) {
  return {
    libraries: context.libraries,
    tables: context.tables,
    ...(context.codes ? { codes: context.codes } : {}),
    settings: context.settings,
  };
}

export type TariffCheckRequest = {
  source: string;
  kind?: TariffFileKind;
  context: TariffContext;
  /** A contract's input values, checked against the declarations (`input-value` findings). */
  values?: { rule?: TariffInputValues; agreement?: TariffInputValues };
};

/** Checks one file with the libraries it imports: findings with fixes, inputs, outline. */
export function tariffCheck(rules: RulesRuntime, request: TariffCheckRequest): TariffCheckResult {
  return tariffCheckResultSchema.parse(
    rules.call(TARIFF_CHECK_OP, {
      source: request.source,
      ...(request.kind ? { kind: request.kind } : {}),
      context: wireContext(request.context),
      ...(request.values ? { values: request.values } : {}),
    }),
  );
}

export type TariffTestRequest = {
  source: string;
  context: TariffContext;
  /** Run only these tests. */
  names?: readonly string[];
};

/** Runs a file's `test` blocks against the sample record. */
export function tariffTest(rules: RulesRuntime, request: TariffTestRequest): TariffTestRun {
  return tariffTestRunSchema.parse(
    rules.call(TARIFF_TEST_OP, {
      source: request.source,
      context: wireContext(request.context),
      ...(request.names ? { names: request.names } : {}),
    }),
  );
}

const formatResultSchema = z.object({
  formatted: z.string().nullable(),
  diagnostics: tariffDiagnosticsSchema,
});
export type TariffFormatResult = z.output<typeof formatResultSchema>;

/** The file in the one canonical style, or null with the syntax errors that stop it. */
export function tariffFormat(rules: RulesRuntime, source: string): TariffFormatResult {
  return formatResultSchema.parse(rules.call(TARIFF_FORMAT_OP, { source }));
}

/** The classes the module highlights a token as. */
export const tariffTokenKindSchema = z.enum([
  "keyword",
  "name",
  "type",
  "code",
  "number",
  "unit",
  "money",
  "percent",
  "date",
  "text",
  "escape",
  "interpolation",
  "punct",
  "comment",
  "doc",
  "error",
]);
export type TariffTokenKind = z.output<typeof tariffTokenKindSchema>;

const tokenSchema = z.object({
  from: z.number().int(),
  to: z.number().int(),
  kind: tariffTokenKindSchema,
});
export type TariffToken = z.output<typeof tokenSchema>;
const tokensSchema = z.object({ tokens: z.array(tokenSchema) });

/** Every token of the file but newlines, in source order, classed for highlighting. */
export function tariffTokens(rules: RulesRuntime, source: string): TariffToken[] {
  return tokensSchema.parse(rules.call(TARIFF_TOKENS_OP, { source })).tokens;
}

/** A doc the catalog carries: the crate's English text, as a message or as plain text. */
const docSchema = z
  .union([z.string(), tariffMessageSchema])
  .nullish()
  .transform((value) => value ?? null);
export type TariffDoc = z.output<typeof docSchema>;

type CatalogField = {
  name: string;
  type: string;
  descriptor?: z.output<typeof tariffTypeSchema>;
  words?: z.output<typeof tariffMessageSchema> | null;
  doc: TariffDoc;
  fields: CatalogField[];
};

const catalogFieldSchema: z.ZodType<CatalogField, unknown> = z.lazy(() =>
  z.object({
    name: z.string(),
    type: z.string(),
    descriptor: tariffTypeSchema.optional(),
    words: tariffMessageSchema.nullish(),
    doc: docSchema,
    fields: z
      .array(catalogFieldSchema)
      .nullish()
      .transform((value) => value ?? []),
  }),
);
export type TariffCatalogField = CatalogField;

const catalogSchema = z.object({
  modules: z.array(
    z.object({
      path: z.string(),
      doc: docSchema,
      declarations: z.array(
        z.object({
          kind: z.enum(["fn", "const", "type", "enum"]),
          name: z.string(),
          signature: z.string(),
          doc: docSchema,
        }),
      ),
    }),
  ),
  types: z.array(z.object({ name: z.string(), doc: docSchema })),
  methods: z.array(
    z.object({
      receiver: z.string(),
      name: z.string(),
      kind: z.enum(["property", "method"]),
      signature: z.string(),
      returns: z.string(),
      doc: docSchema,
    }),
  ),
  tableMethods: z
    .array(z.object({ name: z.string(), doc: docSchema }))
    .nullish()
    .transform((value) => value ?? []),
  units: z.array(z.object({ unit: z.string(), dimension: z.string() })),
  keywords: z.array(z.string()),
  contextual: z
    .array(z.string())
    .nullish()
    .transform((value) => value ?? []),
  schema: z.object({
    roots: z.array(
      z.object({
        name: z.string(),
        doc: docSchema,
        type: z.string(),
        fields: z.array(catalogFieldSchema),
      }),
    ),
    records: z.array(
      z.object({
        name: z
          .string()
          .nullish()
          .transform((value) => value ?? ""),
        fields: z.array(catalogFieldSchema),
      }),
    ),
    enums: z.array(z.object({ name: z.string(), variants: z.array(z.string()) })),
  }),
});
export type TariffCatalog = z.output<typeof catalogSchema>;

const catalogs = new WeakMap<RulesRuntime, TariffCatalog>();

/** The standard library, built-in types, methods, units, keywords and schema, with docs. */
export function tariffCatalog(rules: RulesRuntime): TariffCatalog {
  const cached = catalogs.get(rules);
  if (cached) return cached;
  const catalog = catalogSchema.parse(rules.call(TARIFF_CATALOG_OP, {}));
  catalogs.set(rules, catalog);
  return catalog;
}

export type TariffScaffoldRequest = {
  source: string;
  context: TariffContext;
  /** A real shipment's record, in the schema's wire shape. */
  record: Record<string, TariffWireValue> | null;
  inputs?: TariffInputValues;
  name: string;
};

const scaffoldSchema = z.object({
  test: z
    .string()
    .nullish()
    .transform((value) => value ?? null),
  error: tariffEvalErrorSchema.nullish().transform((value) => value ?? null),
  /** `needs` when the rating reads rate table cells the request did not carry. */
  status: z
    .string()
    .nullish()
    .transform((value) => value ?? null),
  needs: z
    .array(z.object({ table: z.string(), keys: z.array(z.string()) }))
    .nullish()
    .transform((value) => value ?? []),
});
export type TariffScaffold = z.output<typeof scaffoldSchema>;

/** A `test` block pinning what the tariff bills for a real shipment, for "Save as test". */
export function tariffScaffoldTest(
  rules: RulesRuntime,
  request: TariffScaffoldRequest,
): TariffScaffold {
  return scaffoldSchema.parse(
    rules.call(TARIFF_SCAFFOLD_TEST_OP, {
      source: request.source,
      context: wireContext(request.context),
      record: request.record,
      inputs: request.inputs ?? {},
      name: request.name,
    }),
  );
}

/**
 * A request about the cursor: the file, its context and a UTF-16 offset. The editor operations
 * work on a file with errors (the file being typed) and answer from what is known.
 */
export type TariffCursorRequest = {
  source: string;
  context: TariffContext;
  /** The file's kind; the module reads it from the header when it is left out. */
  kind?: TariffFileKind;
  offset: number;
};

function cursorInput(request: TariffCursorRequest) {
  return {
    source: request.source,
    context: wireContext(request.context),
    ...(request.kind ? { kind: request.kind } : {}),
    offset: request.offset,
  };
}

/** What a completion offers. */
export const tariffCompletionKindSchema = z.enum([
  "keyword",
  "name",
  "input",
  "function",
  "method",
  "property",
  "field",
  "type",
  "enum",
  "variant",
  "unit",
  "code",
  "table",
  "module",
  "snippet",
  "value",
]);
export type TariffCompletionKind = z.output<typeof tariffCompletionKindSchema>;

const nullableText = z
  .string()
  .nullish()
  .transform((value) => value ?? null);

const completionSchema = z.object({
  label: z.string(),
  kind: tariffCompletionKindSchema,
  /** The type or signature, shown beside the label. */
  detail: nullableText,
  doc: docSchema,
  /** What a pick inserts; a snippet marks its fields with `${1:name}`. */
  insert: z.string(),
  snippet: z.boolean().default(false),
  /** The module's order, as text that sorts. */
  sort: nullableText,
  /** Replaces the outer range for this item alone, such as `.` rewritten to `?.`. */
  range: tariffRangeSchema.nullish().transform((value) => value ?? null),
});
export type TariffCompletion = z.output<typeof completionSchema>;

const completionsSchema = z.object({
  range: tariffRangeSchema,
  items: z.array(completionSchema),
});
export type TariffCompletions = z.output<typeof completionsSchema>;

/** What may be written at the cursor, and the range a pick replaces. */
export function tariffComplete(
  rules: RulesRuntime,
  request: TariffCursorRequest,
): TariffCompletions {
  return completionsSchema.parse(rules.call(TARIFF_COMPLETE_OP, cursorInput(request)));
}

/** The last preview's trace, which hover and inlay hints read binding values from. */
export type TariffPreviewValues = readonly TariffTraceEvent[];

const hoverSchema = z
  .object({
    range: tariffRangeSchema,
    contents: z.object({
      /** The declaration as Tariff code. */
      signature: nullableText,
      type: nullableText,
      unit: nullableText,
      /** Markdown. */
      doc: docSchema,
      /** What the binding came to in the last preview, and how many times it was bound. */
      value: z
        .object({
          value: tariffWireValueSchema.nullish().transform((value) => value ?? null),
          display: z.string(),
          times: z
            .number()
            .int()
            .nullish()
            .transform((value) => value ?? 1),
        })
        .nullish()
        .transform((value) => value ?? null),
    }),
  })
  .nullable();
export type TariffHover = z.output<typeof hoverSchema>;

/** What the thing under the cursor is: its declaration, type, unit, doc and previewed value. */
export function tariffHover(
  rules: RulesRuntime,
  request: TariffCursorRequest & { values?: TariffPreviewValues },
): TariffHover {
  return hoverSchema.parse(
    rules.call(TARIFF_HOVER_OP, {
      ...cursorInput(request),
      ...(request.values && request.values.length > 0 ? { values: request.values } : {}),
    }),
  );
}

const signatureSchema = z
  .object({
    signatures: z.array(
      z.object({
        label: z.string(),
        doc: docSchema,
        parameters: z.array(
          z.object({
            /** Where the parameter is in the label: UTF-16 `[from, to]`. */
            label: z.tuple([z.number().int(), z.number().int()]),
            doc: docSchema,
          }),
        ),
      }),
    ),
    active: z
      .number()
      .int()
      .nullish()
      .transform((value) => value ?? null),
    activeParameter: z
      .number()
      .int()
      .nullish()
      .transform((value) => value ?? null),
  })
  .nullable();
export type TariffSignatureHelp = z.output<typeof signatureSchema>;

/** The call the cursor is in, each candidate signature and the parameter being written. */
export function tariffSignature(
  rules: RulesRuntime,
  request: TariffCursorRequest,
): TariffSignatureHelp {
  return signatureSchema.parse(rules.call(TARIFF_SIGNATURE_OP, cursorInput(request)));
}

const definitionSchema = z.object({
  targets: z.array(
    z.object({
      /** The library the declaration is in (`std.ltl`, `org.x`), or null for this file. */
      module: nullableText,
      range: tariffRangeSchema,
      selectionRange: tariffRangeSchema,
    }),
  ),
});
export type TariffDefinitionTargets = z.output<typeof definitionSchema>["targets"];

/** Where the name under the cursor is declared: in this file, or in a library it imports. */
export function tariffDefinition(
  rules: RulesRuntime,
  request: TariffCursorRequest,
): TariffDefinitionTargets {
  return definitionSchema.parse(rules.call(TARIFF_DEFINITION_OP, cursorInput(request))).targets;
}

const editSchema = z.object({ range: tariffRangeSchema, insert: z.string() });

const renameSchema = z.union([
  z.object({ edits: z.array(editSchema) }),
  z.object({ error: tariffMessageSchema }),
]);
export type TariffRename = z.output<typeof renameSchema>;

/** Every edit renaming the name under the cursor, or why it cannot be renamed. */
export function tariffRename(
  rules: RulesRuntime,
  request: TariffCursorRequest & { newName: string },
): TariffRename {
  return renameSchema.parse(
    rules.call(TARIFF_RENAME_OP, { ...cursorInput(request), newName: request.newName }),
  );
}

const prepareRenameSchema = z.union([
  z.object({ range: tariffRangeSchema, placeholder: z.string() }),
  z.object({ error: tariffMessageSchema }),
]);
export type TariffPrepareRename = z.output<typeof prepareRenameSchema>;

/** Whether the name under the cursor can be renamed: its range and current text, or why not. */
export function tariffPrepareRename(
  rules: RulesRuntime,
  request: TariffCursorRequest,
): TariffPrepareRename {
  return prepareRenameSchema.parse(rules.call(TARIFF_RENAME_OP, cursorInput(request)));
}

const inlaysSchema = z.object({
  hints: z.array(
    z.object({
      offset: z.number().int(),
      label: z.string(),
      kind: z.enum(["type", "value", "parameter"]),
    }),
  ),
});
export type TariffInlay = z.output<typeof inlaysSchema>["hints"][number];

/**
 * Inlay hints for a range: inferred types after bindings and lambda parameters, parameter names
 * before positional arguments, and with the last preview's trace each binding's value.
 */
export function tariffInlays(
  rules: RulesRuntime,
  request: {
    source: string;
    context: TariffContext;
    kind?: TariffFileKind;
    from: number;
    to: number;
    values?: TariffPreviewValues;
  },
): TariffInlay[] {
  return inlaysSchema.parse(
    rules.call(TARIFF_INLAYS_OP, {
      source: request.source,
      context: wireContext(request.context),
      ...(request.kind ? { kind: request.kind } : {}),
      range: { from: request.from, to: request.to },
      ...(request.values && request.values.length > 0 ? { values: request.values } : {}),
    }),
  ).hints;
}
