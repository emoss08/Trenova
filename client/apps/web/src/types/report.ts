import type { ReportCatalogField } from "@/lib/graphql/reports";

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

export type ReportComputedSpec = {
  op: ReportComputedOp;
  leftId: string;
  rightId: string;
  format?: string;
};

export type ReportColumnSpec = {
  id: string;
  ref?: ReportFieldRef;
  kind: ReportColumnKind;
  agg?: ReportAggregation;
  bucket?: ReportDateBucket;
  label?: string;
  computed?: ReportComputedSpec;
};

export type ReportFieldFilter = {
  ref: ReportFieldRef;
  operator: string;
  value?: unknown;
  param?: string;
  agg?: ReportAggregation;
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

export const REPORT_CATEGORY_CHOICES: { value: string; label: string }[] = [
  { value: "operations", label: "Operations" },
  { value: "billing", label: "Billing" },
  { value: "compliance", label: "Compliance" },
  { value: "fleet", label: "Fleet" },
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
