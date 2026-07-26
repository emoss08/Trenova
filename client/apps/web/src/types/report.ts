import type { ReportCatalogField } from "@/lib/graphql/reports";
import type {
  ReportBoolStyle,
  ReportDateStyle,
  ReportDisplayStyle,
  ReportDurationStyle,
  ReportDurationUnit,
  ReportNegativeStyle,
  ReportNotation,
} from "@trenova/shared/lib/report-format";

export const REPORT_IR_VERSION = 1;

export type ReportColumnKind = "dimension" | "measure" | "computed";

export type ReportAggregation = "count" | "count_distinct" | "sum" | "avg" | "min" | "max";

export type ReportDateBucket = "day" | "week" | "month" | "quarter" | "year";

export type ReportComputedOp = "add" | "subtract" | "multiply" | "divide";

export type ReportBoolOp = "and" | "or";

export type ReportFieldRef = {
  path?: string[];
  field: string;
};

// A calculation operand is either another measure in the report or a constant
// target. leftValue/rightValue carry the target for that side, and the two are
// mutually exclusive — a goal has no column to point at.
export type ReportComputedSpec = {
  op: ReportComputedOp;
  leftId?: string;
  rightId?: string;
  leftValue?: number;
  rightValue?: number;
  format?: string;
};

export type ReportComputedSide = "left" | "right";

export type ReportComputedOperand = {
  columnId?: string;
  value?: number;
};

export function computedOperand(
  computed: ReportComputedSpec,
  side: ReportComputedSide,
): ReportComputedOperand {
  return side === "left"
    ? { columnId: computed.leftId, value: computed.leftValue }
    : { columnId: computed.rightId, value: computed.rightValue };
}

export function computedOperandIsTarget(operand: ReportComputedOperand): boolean {
  return operand.value !== undefined;
}

// withComputedOperand keeps the exclusivity invariant in one place: setting one
// form of an operand always clears the other, so a spec can never reach the
// server carrying both a column and a target.
export function withComputedOperand(
  computed: ReportComputedSpec,
  side: ReportComputedSide,
  operand: ReportComputedOperand,
): ReportComputedSpec {
  return side === "left"
    ? { ...computed, leftId: operand.columnId, leftValue: operand.value }
    : { ...computed, rightId: operand.columnId, rightValue: operand.value };
}

export type ReportTransformOp =
  | "round"
  | "ceil"
  | "floor"
  | "truncate"
  | "abs"
  | "scale"
  | "upper"
  | "lower"
  | "title"
  | "trim";

export type ReportTransformSpec = {
  op: ReportTransformOp;
  precision?: number;
  factor?: number;
};

export type ReportDisplayTone = "positive" | "warning" | "negative" | "neutral";

export type ReportDisplayRuleOp = "gt" | "gte" | "lt" | "lte" | "eq" | "between";

export type ReportDisplayRuleSpec = {
  op: ReportDisplayRuleOp;
  value: number;
  upper?: number;
  tone: ReportDisplayTone;
};

export const MAX_DISPLAY_RULES = 8;

export const REPORT_RULE_OP_CHOICES: Choice<ReportDisplayRuleOp>[] = [
  { value: "gt", label: "Greater than" },
  { value: "gte", label: "At least" },
  { value: "lt", label: "Less than" },
  { value: "lte", label: "At most" },
  { value: "eq", label: "Equals" },
  { value: "between", label: "Between" },
];

export const REPORT_TONE_CHOICES: Choice<ReportDisplayTone>[] = [
  { value: "positive", label: "Good" },
  { value: "warning", label: "Watch" },
  { value: "negative", label: "Bad" },
  { value: "neutral", label: "Emphasis" },
];

export function ruleNeedsUpper(op: ReportDisplayRuleOp): boolean {
  return op === "between";
}

export type ReportDisplaySpec = {
  style?: ReportDisplayStyle;
  decimals?: number;
  grouping?: boolean;
  currency?: string;
  negative?: ReportNegativeStyle;
  notation?: ReportNotation;
  prefix?: string;
  suffix?: string;
  dateStyle?: ReportDateStyle;
  boolStyle?: ReportBoolStyle;
  durationUnit?: ReportDurationUnit;
  durationStyle?: ReportDurationStyle;
  nullText?: string;
  rules?: ReportDisplayRuleSpec[];
};

export type ReportColumnSpec = {
  id: string;
  ref?: ReportFieldRef;
  kind: ReportColumnKind;
  agg?: ReportAggregation;
  bucket?: ReportDateBucket;
  /**
   * Groups a numeric dimension into ranges — the numeric counterpart of
   * `bucket`, and mutually exclusive with it.
   */
  band?: ReportBandSpec;
  label?: string;
  computed?: ReportComputedSpec;
  transform?: ReportTransformSpec;
  display?: ReportDisplaySpec;
  /** Narrows what this measure aggregates without narrowing the report. */
  filter?: ReportFilterGroup;
};

export type ReportFieldFilter = {
  ref: ReportFieldRef;
  operator: string;
  value?: unknown;
  param?: string;
  agg?: ReportAggregation;
  transform?: ReportTransformSpec;
};

export type ReportFilterGroup = {
  op: ReportBoolOp;
  filters?: ReportFieldFilter[];
  groups?: ReportFilterGroup[];
};

export type ReportSortSpec = {
  columnId: string;
  direction: "asc" | "desc";
};

export type ReportPivotSpec = {
  ref: ReportFieldRef;
  values: string[];
  labels?: string[];
  measureIds: string[];
  includeOther?: boolean;
};

export type ReportParameterDef = {
  name: string;
  label?: string;
  type: string;
  required: boolean;
  default?: unknown;
  multi?: boolean;
  allowedValues?: string[];
  refEntity?: string;
};

export type ReportChartType =
  | "bar"
  | "hbar"
  | "line"
  | "area"
  | "pie"
  | "donut"
  | "scatter"
  | "kpi"
  | "map";

export type ReportChartGoal = {
  value?: number;
  columnId?: string;
  label?: string;
};

export type ReportChartSpec = {
  id: string;
  type: ReportChartType;
  title?: string;
  xColumnId?: string;
  seriesIds?: string[];
  stacked?: boolean;
  hideLegend?: boolean;
  showValues?: boolean;
  curved?: boolean;
  limit?: number;
  compareId?: string;
  goal?: ReportChartGoal;
  /** A map positions each row by coordinate; the pair is required together. */
  latColumnId?: string;
  lngColumnId?: string;
  labelColumnId?: string;
};

export type ReportIR = {
  irVersion?: number;
  entity: string;
  columns: ReportColumnSpec[];
  filters?: ReportFilterGroup | null;
  having?: ReportFilterGroup | null;
  sort?: ReportSortSpec[];
  limit?: number;
  pivot?: ReportPivotSpec | null;
  parameters?: ReportParameterDef[];
  totals?: boolean;
  charts?: ReportChartSpec[];
};

export type ReportOperatorChoice = {
  value: string;
  label: string;
  requiresValue: boolean;
  multiValue?: boolean;
};

const OPERATORS_EQUALITY: ReportOperatorChoice[] = [
  { value: "eq", label: "Equals", requiresValue: true },
  { value: "ne", label: "Not equals", requiresValue: true },
];

const OPERATORS_ORDERING: ReportOperatorChoice[] = [
  { value: "gt", label: "Greater than", requiresValue: true },
  { value: "gte", label: "Greater than or equal", requiresValue: true },
  { value: "lt", label: "Less than", requiresValue: true },
  { value: "lte", label: "Less than or equal", requiresValue: true },
];

const OPERATORS_TEXT: ReportOperatorChoice[] = [
  { value: "contains", label: "Contains", requiresValue: true },
  { value: "startswith", label: "Starts with", requiresValue: true },
  { value: "endswith", label: "Ends with", requiresValue: true },
];

const OPERATORS_SET: ReportOperatorChoice[] = [
  { value: "in", label: "Is any of", requiresValue: true, multiValue: true },
  { value: "notin", label: "Is none of", requiresValue: true, multiValue: true },
];

const OPERATORS_NULLNESS: ReportOperatorChoice[] = [
  { value: "isnull", label: "Is empty", requiresValue: false },
  { value: "isnotnull", label: "Is not empty", requiresValue: false },
];

const OPERATORS_DATE: ReportOperatorChoice[] = [
  { value: "daterange", label: "Between dates", requiresValue: true, multiValue: true },
  { value: "lastndays", label: "In the last N days", requiresValue: true },
  { value: "nextndays", label: "In the next N days", requiresValue: true },
  { value: "today", label: "Today", requiresValue: false },
  { value: "yesterday", label: "Yesterday", requiresValue: false },
  { value: "tomorrow", label: "Tomorrow", requiresValue: false },
  { value: "thisweek", label: "This week", requiresValue: false },
  { value: "lastweek", label: "Last week", requiresValue: false },
  { value: "thismonth", label: "This month", requiresValue: false },
  { value: "lastmonth", label: "Last month", requiresValue: false },
  { value: "thisquarter", label: "This quarter", requiresValue: false },
  { value: "lastquarter", label: "Last quarter", requiresValue: false },
  { value: "thisyear", label: "This year", requiresValue: false },
  { value: "lastyear", label: "Last year", requiresValue: false },
];

export function operatorsForFieldType(fieldType: string): ReportOperatorChoice[] {
  switch (fieldType) {
    case "string":
      return [
        ...OPERATORS_EQUALITY,
        ...OPERATORS_TEXT,
        ...OPERATORS_ORDERING,
        ...OPERATORS_SET,
        ...OPERATORS_NULLNESS,
      ];
    case "int":
      return [
        ...OPERATORS_EQUALITY,
        ...OPERATORS_ORDERING,
        ...OPERATORS_SET,
        ...OPERATORS_NULLNESS,
      ];
    case "decimal":
      return [...OPERATORS_EQUALITY, ...OPERATORS_ORDERING, ...OPERATORS_NULLNESS];
    case "epoch":
      return [
        ...OPERATORS_EQUALITY,
        ...OPERATORS_ORDERING,
        ...OPERATORS_DATE,
        ...OPERATORS_NULLNESS,
      ];
    case "enum":
    case "ref":
      return [...OPERATORS_EQUALITY, ...OPERATORS_SET, ...OPERATORS_NULLNESS];
    case "bool":
      return [...OPERATORS_EQUALITY, ...OPERATORS_NULLNESS];
    case "json":
      return [...OPERATORS_NULLNESS];
    default:
      return [...OPERATORS_EQUALITY, ...OPERATORS_NULLNESS];
  }
}

export function operatorChoice(operator: string): ReportOperatorChoice | undefined {
  return [
    ...OPERATORS_EQUALITY,
    ...OPERATORS_ORDERING,
    ...OPERATORS_TEXT,
    ...OPERATORS_SET,
    ...OPERATORS_NULLNESS,
    ...OPERATORS_DATE,
  ].find((choice) => choice.value === operator);
}

export const REPORT_AGGREGATION_LABELS: Record<ReportAggregation, string> = {
  count: "Count",
  count_distinct: "Count distinct",
  sum: "Sum",
  avg: "Average",
  min: "Minimum",
  max: "Maximum",
};

/** Mirrors `pkg/reportfmt.Band`: a fixed width or an ascending boundary list. */
export type ReportBandSpec = {
  width?: number;
  edges?: number[];
};

export type ReportBandMode = "width" | "edges";

export const MAX_BAND_EDGES = 24;

export function bandMode(band: ReportBandSpec | undefined): ReportBandMode {
  return band?.edges && band.edges.length > 0 ? "edges" : "width";
}

/** A band only groups something once it has a width or two boundaries. */
export function bandIsUsable(band: ReportBandSpec | undefined): boolean {
  if (!band) return false;
  if (band.edges && band.edges.length > 0) return band.edges.length >= 2;
  return (band.width ?? 0) > 0;
}

export function bandIsSet(band: ReportBandSpec | undefined): boolean {
  return !!band && ((band.width ?? 0) !== 0 || (band.edges?.length ?? 0) > 0);
}

/** Only numeric columns carry a magnitude that ranges can divide. */
export function bandableType(valueType: string | undefined): boolean {
  return valueType === "int" || valueType === "decimal";
}

export const REPORT_DATE_BUCKET_CHOICES: { value: ReportDateBucket; label: string }[] = [
  { value: "day", label: "Day" },
  { value: "week", label: "Week" },
  { value: "month", label: "Month" },
  { value: "quarter", label: "Quarter" },
  { value: "year", label: "Year" },
];

export const REPORT_FORMAT_CHOICES: { value: string; label: string }[] = [
  { value: "csv", label: "CSV" },
  { value: "xlsx", label: "Excel (XLSX)" },
  { value: "pdf", label: "PDF" },
  { value: "json", label: "JSON" },
];

export const REPORT_PARAMETER_TYPE_CHOICES: { value: string; label: string }[] = [
  { value: "string", label: "Text" },
  { value: "int", label: "Whole number" },
  { value: "decimal", label: "Decimal" },
  { value: "bool", label: "Yes / No" },
  { value: "epoch", label: "Date" },
  { value: "ref", label: "Entity" },
];

export const REPORT_COMPUTED_OP_CHOICES: { value: ReportComputedOp; label: string }[] = [
  { value: "divide", label: "Divided by (÷)" },
  { value: "multiply", label: "Multiplied by (×)" },
  { value: "add", label: "Plus (+)" },
  { value: "subtract", label: "Minus (−)" },
];

export const REPORT_COMPUTED_FORMAT_CHOICES: { value: string; label: string }[] = [
  { value: "none", label: "Number" },
  { value: "money", label: "Money" },
  { value: "percent", label: "Percent" },
  { value: "weight", label: "Weight" },
  { value: "distance", label: "Distance" },
  { value: "duration", label: "Duration" },
  { value: "count", label: "Count" },
];

type Choice<T extends string> = { value: T; label: string };

// Which display styles can apply to a column, given the type the column emits.
// Mirrors reportfmt.styleAllowed on the server.
export function displayStylesForType(fieldType: string): Choice<ReportDisplayStyle>[] {
  switch (fieldType) {
    case "int":
    case "decimal":
      return [
        { value: "number", label: "Number" },
        { value: "currency", label: "Currency" },
        { value: "percent", label: "Percent (×100)" },
        { value: "duration", label: "Duration" },
        { value: "date", label: "Date & time" },
        { value: "text", label: "Plain text" },
      ];
    case "epoch":
      return [
        { value: "date", label: "Date & time" },
        { value: "number", label: "Raw epoch number" },
        { value: "text", label: "Plain text" },
      ];
    case "bool":
      return [
        { value: "bool", label: "Yes / No" },
        { value: "text", label: "Plain text" },
      ];
    default:
      return [{ value: "text", label: "Plain text" }];
  }
}

export const REPORT_DATE_STYLE_CHOICES: Choice<ReportDateStyle>[] = [
  { value: "dateTime", label: "2026-03-04 14:05" },
  { value: "date", label: "2026-03-04" },
  { value: "dateLong", label: "Mar 4, 2026" },
  { value: "time", label: "14:05" },
  { value: "monthYear", label: "Mar 2026" },
  { value: "quarter", label: "Q1 2026" },
  { value: "year", label: "2026" },
  { value: "weekday", label: "Wed 2026-03-04" },
  { value: "iso", label: "ISO 8601" },
];

export const REPORT_BOOL_STYLE_CHOICES: Choice<ReportBoolStyle>[] = [
  { value: "yesNo", label: "Yes / No" },
  { value: "trueFalse", label: "True / False" },
  { value: "onOff", label: "On / Off" },
  { value: "check", label: "✓ / ✗" },
  { value: "oneZero", label: "1 / 0" },
];

export const REPORT_DURATION_UNIT_CHOICES: Choice<ReportDurationUnit>[] = [
  { value: "seconds", label: "Value is in seconds" },
  { value: "minutes", label: "Value is in minutes" },
  { value: "hours", label: "Value is in hours" },
  { value: "days", label: "Value is in days" },
];

export const REPORT_DURATION_STYLE_CHOICES: Choice<ReportDurationStyle>[] = [
  { value: "clock", label: "26:10" },
  { value: "compact", label: "1d 2h 10m" },
  { value: "verbose", label: "1 day 2 hours" },
  { value: "decimalHours", label: "26.2 h" },
];

export const REPORT_NEGATIVE_CHOICES: Choice<ReportNegativeStyle>[] = [
  { value: "minus", label: "-1,234.00" },
  { value: "parentheses", label: "(1,234.00)" },
];

export const REPORT_NOTATION_CHOICES: Choice<ReportNotation>[] = [
  { value: "standard", label: "1,284,000" },
  { value: "compact", label: "1.3M" },
];

export const REPORT_CURRENCY_CHOICES: { value: string; label: string }[] = [
  { value: "USD", label: "USD ($)" },
  { value: "CAD", label: "CAD (C$)" },
  { value: "MXN", label: "MXN (MX$)" },
  { value: "EUR", label: "EUR (€)" },
  { value: "GBP", label: "GBP (£)" },
];

export const REPORT_NUMERIC_TRANSFORM_CHOICES: Choice<ReportTransformOp>[] = [
  { value: "round", label: "Round" },
  { value: "ceil", label: "Round up" },
  { value: "floor", label: "Round down" },
  { value: "truncate", label: "Truncate" },
  { value: "abs", label: "Absolute value" },
  { value: "scale", label: "Multiply by" },
];

export const REPORT_TEXT_TRANSFORM_CHOICES: Choice<ReportTransformOp>[] = [
  { value: "upper", label: "UPPERCASE" },
  { value: "lower", label: "lowercase" },
  { value: "title", label: "Title Case" },
  { value: "trim", label: "Trim whitespace" },
];

export const REPORT_TRANSFORM_LABELS: Record<ReportTransformOp, string> = {
  round: "Round",
  ceil: "Round up",
  floor: "Round down",
  truncate: "Truncate",
  abs: "Absolute value",
  scale: "Multiply by",
  upper: "UPPERCASE",
  lower: "lowercase",
  title: "Title Case",
  trim: "Trim whitespace",
};

export const MAX_TRANSFORM_PRECISION = 6;

export function transformChoicesForType(fieldType: string): Choice<ReportTransformOp>[] {
  switch (fieldType) {
    case "int":
    case "decimal":
      return REPORT_NUMERIC_TRANSFORM_CHOICES;
    case "string":
    case "enum":
      return REPORT_TEXT_TRANSFORM_CHOICES;
    default:
      return [];
  }
}

export function transformUsesPrecision(op: ReportTransformOp): boolean {
  return op === "round" || op === "ceil" || op === "floor" || op === "truncate";
}

// The value type a column emits, which is what display styles and transforms
// are validated against — not the raw catalog field type.
export function columnValueType(column: ReportColumnSpec, fieldType: string | undefined): string {
  if (column.kind === "computed") return "decimal";
  if (column.kind === "measure") {
    if (column.agg === "count" || column.agg === "count_distinct") return "int";
    if (column.agg === "avg") return "decimal";
    if (column.agg === "sum") return fieldType === "int" ? "int" : "decimal";
    return fieldType ?? "string";
  }
  if (column.bucket) return "epoch";
  return fieldType ?? "string";
}

export const MAX_CHARTS = 6;
export const MAX_CHART_SERIES = 12;
export const PIVOT_OTHER_VALUE = "__other__";

export const REPORT_CHART_TYPE_CHOICES: Choice<ReportChartType>[] = [
  { value: "bar", label: "Bar" },
  { value: "hbar", label: "Horizontal bar" },
  { value: "line", label: "Line" },
  { value: "area", label: "Area" },
  { value: "pie", label: "Pie" },
  { value: "donut", label: "Donut" },
  { value: "scatter", label: "Scatter" },
  { value: "kpi", label: "KPI tile" },
  { value: "map", label: "Map" },
];

export const REPORT_CHART_TYPE_LABELS: Record<ReportChartType, string> = {
  bar: "Bar",
  hbar: "Horizontal bar",
  line: "Line",
  area: "Area",
  pie: "Pie",
  donut: "Donut",
  scatter: "Scatter",
  kpi: "KPI tile",
  map: "Map",
};

// Mirrors report.ChartType.NeedsDimension: KPI tiles, scatter plots, and maps
// read their position from somewhere other than a grouping column.
export function chartNeedsDimension(type: ReportChartType): boolean {
  return type !== "kpi" && type !== "scatter" && type !== "map";
}

export function chartSingleSeries(type: ReportChartType): boolean {
  return (
    type === "pie" || type === "donut" || type === "kpi" || type === "scatter" || type === "map"
  );
}

// Mirrors report.ChartType.NeedsCoordinates.
export function chartNeedsCoordinates(type: ReportChartType): boolean {
  return type === "map";
}

export function chartSupportsStacking(type: ReportChartType): boolean {
  return type === "bar" || type === "hbar" || type === "area";
}

/**
 * The column ids the compiled query will actually return: a pivoted measure
 * becomes one addressable column per bucket, which is what charts and sorting
 * point at.
 */
export function reportOutputColumnIds(ir: ReportIR): { id: string; isDim: boolean }[] {
  const pivoted = new Set(ir.pivot?.measureIds ?? []);
  const outputs: { id: string; isDim: boolean }[] = [];

  for (const column of ir.columns) {
    if (column.kind === "dimension") {
      outputs.push({ id: column.id, isDim: true });
      continue;
    }
    if (!ir.pivot || !pivoted.has(column.id)) {
      outputs.push({ id: column.id, isDim: false });
      continue;
    }
    for (const value of ir.pivot.values) {
      outputs.push({ id: `${column.id}:${value}`, isDim: false });
    }
    if (ir.pivot.includeOther) {
      outputs.push({ id: `${column.id}:${PIVOT_OTHER_VALUE}`, isDim: false });
    }
  }

  return outputs;
}

export type ReportDashboardTileKind = "table" | "chart" | "kpi" | "text";

export type ReportDashboardTile = {
  id: string;
  kind: ReportDashboardTileKind;
  title?: string;
  definitionId?: string;
  cannedKey?: string;
  chartId?: string;
  columnId?: string;
  text?: string;
  x: number;
  y: number;
  w: number;
  h: number;
  limit?: number;
  paramBindings?: Record<string, string>;
};

/**
 * A field-level dashboard filter. Unlike a parameter, the tile's report does
 * not have to declare anything — every tile querying the same entity gets the
 * predicate folded into its own filters.
 */
export type ReportDashboardFilter = {
  id: string;
  label?: string;
  entity: string;
  ref: ReportFieldRef;
  operator: string;
  default?: unknown;
};

export type ReportDashboardLayout = {
  tiles: ReportDashboardTile[];
  parameters?: ReportParameterDef[];
  filters?: ReportDashboardFilter[];
};

export const DASHBOARD_GRID_COLUMNS = 12;
export const MAX_DASHBOARD_TILES = 24;
export const MAX_TILE_HEIGHT = 12;

export const REPORT_TILE_KIND_CHOICES: Choice<ReportDashboardTileKind>[] = [
  { value: "chart", label: "Chart" },
  { value: "kpi", label: "KPI" },
  { value: "table", label: "Table" },
  { value: "text", label: "Text" },
];

export function tileNeedsReport(kind: ReportDashboardTileKind): boolean {
  return kind !== "text";
}

export function parseDashboardLayout(layout: unknown): ReportDashboardLayout {
  if (!layout || typeof layout !== "object") return { tiles: [] };
  const parsed = layout as ReportDashboardLayout;
  return {
    tiles: Array.isArray(parsed.tiles) ? parsed.tiles : [],
    parameters: Array.isArray(parsed.parameters) ? parsed.parameters : undefined,
    filters: Array.isArray(parsed.filters) ? parsed.filters : undefined,
  };
}

export const REPORT_CATEGORY_CHOICES: { value: string; label: string }[] = [
  { value: "operations", label: "Operations" },
  { value: "billing", label: "Billing" },
  { value: "fleet", label: "Fleet" },
  { value: "compliance", label: "Compliance" },
  { value: "customers", label: "Customers" },
  { value: "custom", label: "Custom" },
];

export const REPORT_RUN_STATUS_LABELS: Record<string, string> = {
  queued: "Queued",
  running: "Running",
  succeeded: "Succeeded",
  failed: "Failed",
  canceled: "Canceled",
  expired: "Expired",
};

export const REPORT_RUN_TRIGGER_LABELS: Record<string, string> = {
  manual: "Manual",
  scheduled: "Scheduled",
  api: "API",
};

export const REPORT_DEFINITION_STATUS_LABELS: Record<string, string> = {
  draft: "Draft",
  active: "Active",
  archived: "Archived",
  needs_attention: "Needs Attention",
};

export const REPORT_VISIBILITY_LABELS: Record<string, string> = {
  private: "Private",
  shared: "Shared",
};

export function parseReportIR(definition: unknown): ReportIR | null {
  if (!definition || typeof definition !== "object") return null;
  const ir = definition as ReportIR;
  if (typeof ir.entity !== "string" || !Array.isArray(ir.columns)) return null;
  return ir;
}

export function aggregationsForField(field: ReportCatalogField): ReportAggregation[] {
  return field.aggregations.filter(
    (agg): agg is ReportAggregation => agg in REPORT_AGGREGATION_LABELS,
  );
}
