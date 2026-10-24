// report-definition.ts converts between the report builder's editing model
// (`ReportIR` in `@/types/report`, which the panels edit) and the keel analytics
// definition (version 2) the server stores, validates and runs. The builder edits
// a formula as an operation on two operands, a filter as a field reference and an
// operator from its own menus, and a having condition as a field and an
// aggregation; the server speaks dotted field paths, keel operators, formula text
// and having conditions on output columns. Everything the builder cannot edit
// (windows, top-N, subtotals, parameter-driven limits, free formulas, conditions on
// formulas) is carried through untouched, so opening and saving a definition the
// assistant wrote never loses part of it.
import type {
  Agg,
  Bucket,
  Chart,
  Column,
  DashboardFilter,
  Definition,
  Filter,
  Op,
  Parameter,
  Sort,
  ValueType,
} from "@trenova/shared/generated/keel-analytics";
import type {
  ReportAggregation,
  ReportChartSpec,
  ReportColumnSpec,
  ReportComputedOp,
  ReportComputedSpec,
  ReportDashboardFilter,
  ReportDateBucket,
  ReportFieldFilter,
  ReportFieldRef,
  ReportFilterGroup,
  ReportIR,
  ReportParameterDef,
} from "@/types/report";

export type { Definition as KeelDefinition } from "@trenova/shared/generated/keel-analytics";

const BUILDER_TO_KEEL_OPS: Record<string, Op> = {
  eq: "eq",
  ne: "ne",
  gt: "gt",
  gte: "gte",
  lt: "lt",
  lte: "lte",
  contains: "contains",
  startswith: "starts_with",
  endswith: "ends_with",
  like: "like",
  ilike: "ilike",
  in: "in",
  notin: "not_in",
  isnull: "is_null",
  isnotnull: "is_not_null",
  daterange: "between",
  lastndays: "last_n_days",
  nextndays: "next_n_days",
  today: "today",
  yesterday: "yesterday",
  tomorrow: "tomorrow",
  thisweek: "this_week",
  lastweek: "last_week",
  thismonth: "this_month",
  lastmonth: "last_month",
  thisquarter: "this_quarter",
  lastquarter: "last_quarter",
  thisyear: "this_year",
  lastyear: "last_year",
};

const KEEL_TO_BUILDER_OPS: Record<string, string> = Object.fromEntries(
  Object.entries(BUILDER_TO_KEEL_OPS).map(([builder, keel]) => [keel, builder]),
);

const COMPUTED_SYMBOLS: Record<ReportComputedOp, string> = {
  add: "+",
  subtract: "-",
  multiply: "*",
  divide: "/",
};

const SYMBOL_OPS: Record<string, ReportComputedOp> = {
  "+": "add",
  "-": "subtract",
  "*": "multiply",
  "/": "divide",
};

export function toKeelOp(operator: string): Op {
  return BUILDER_TO_KEEL_OPS[operator] ?? (operator as Op);
}

export function fromKeelOp(op: string | undefined): string {
  if (!op) return "eq";
  return KEEL_TO_BUILDER_OPS[op] ?? op;
}

export function refPath(ref: ReportFieldRef | undefined): string {
  if (!ref || !ref.field) return "";
  return [...(ref.path ?? []), ref.field].join(".");
}

export function pathRef(path: string | undefined): ReportFieldRef {
  if (!path) return { field: "" };
  const parts = path.split(".");
  const field = parts.pop() ?? "";
  return parts.length > 0 ? { path: parts, field } : { field };
}

function leafToKeel(filter: ReportFieldFilter): Filter {
  if (filter.raw) return filter.raw as Filter;
  const out: Filter = { field: refPath(filter.ref), op: toKeelOp(filter.operator) };
  if (filter.param) out.param = filter.param;
  else if (filter.value !== undefined && filter.value !== null) out.value = filter.value;
  if (filter.transform) out.transform = filter.transform;
  return out;
}

export function groupToKeel(group: ReportFilterGroup | null | undefined): Filter | undefined {
  if (!group) return undefined;
  const children: Filter[] = [];
  for (const leaf of group.filters ?? []) children.push(leafToKeel(leaf));
  for (const nested of group.groups ?? []) {
    const converted = groupToKeel(nested);
    if (converted) children.push(converted);
  }
  if (children.length === 0) return undefined;
  return group.op === "or" ? { or: children } : { and: children };
}

function leafFromKeel(filter: Filter): ReportFieldFilter {
  if (!filter.field) {
    return { ref: { field: "" }, operator: fromKeelOp(filter.op), raw: filter };
  }
  const leaf: ReportFieldFilter = { ref: pathRef(filter.field), operator: fromKeelOp(filter.op) };
  if (filter.param) leaf.param = filter.param;
  if (filter.value !== undefined) leaf.value = filter.value;
  if (filter.transform) leaf.transform = filter.transform as ReportFieldFilter["transform"];
  return leaf;
}

export function groupFromKeel(filter: Filter | null | undefined): ReportFilterGroup | undefined {
  if (!filter) return undefined;
  const children = filter.and ?? filter.or;
  if (!children) return { op: "and", filters: [leafFromKeel(filter)] };
  const group: ReportFilterGroup = { op: filter.or ? "or" : "and", filters: [], groups: [] };
  for (const child of children) {
    if (child.and || child.or) {
      const nested = groupFromKeel(child);
      if (nested) group.groups?.push(nested);
    } else {
      group.filters?.push(leafFromKeel(child));
    }
  }
  return group;
}

function operandText(columnId: string | undefined, value: number | undefined): string {
  if (columnId) return `[${columnId}]`;
  return String(value ?? 0);
}

export function computedToFormula(computed: ReportComputedSpec): string {
  return `${operandText(computed.leftId, computed.leftValue)} ${COMPUTED_SYMBOLS[computed.op]} ${operandText(computed.rightId, computed.rightValue)}`;
}

const OPERAND = String.raw`(?:\[(\w+)\]|(-?\d+(?:\.\d+)?))`;
const SIMPLE_FORMULA = new RegExp(String.raw`^\s*${OPERAND}\s*([-+*/])\s*${OPERAND}\s*$`);

export function formulaToComputed(text: string, format?: string): ReportComputedSpec | undefined {
  const match = SIMPLE_FORMULA.exec(text);
  if (!match) return undefined;
  const [, leftId, leftValue, symbol, rightId, rightValue] = match;
  const op = SYMBOL_OPS[symbol];
  if (!op) return undefined;
  return {
    op,
    leftId: leftId || undefined,
    leftValue: leftValue !== undefined ? Number(leftValue) : undefined,
    rightId: rightId || undefined,
    rightValue: rightValue !== undefined ? Number(rightValue) : undefined,
    format: format || undefined,
  };
}

function columnToKeel(column: ReportColumnSpec): Column {
  const out: Column = { id: column.id };
  if (column.label) out.label = column.label;
  if (column.hidden) out.hidden = true;
  if (column.transform) out.transform = column.transform;
  if (column.display && Object.keys(column.display).length > 0) {
    out.display = column.display as Record<string, unknown>;
  }
  switch (column.kind) {
    case "dimension": {
      const bucket = column.bucketParam
        ? { param: column.bucketParam }
        : (column.bucket as Bucket | undefined);
      out.dimension = { field: refPath(column.ref) };
      if (bucket) out.dimension.bucket = bucket;
      if (column.band && ((column.band.width ?? 0) > 0 || (column.band.edges?.length ?? 0) > 0)) {
        out.dimension.band = column.band;
      }
      break;
    }
    case "measure": {
      out.measure = { agg: (column.agg ?? "count") as Agg };
      const field = refPath(column.ref);
      if (field) out.measure.field = field;
      const filter = groupToKeel(column.filter);
      if (filter) out.measure.filter = filter;
      break;
    }
    case "computed": {
      if (column.window) {
        out.window = column.window as Column["window"];
      } else if (column.computed) {
        out.calc = computedToFormula(column.computed);
        const format = column.computed.format;
        if (format && format !== "none") out.format = format;
      } else if (column.calc !== undefined) {
        out.calc = column.calc as Column["calc"];
        if (column.format) out.format = column.format;
      }
      break;
    }
  }
  return out;
}

function havingColumnId(agg: string, path: string, taken: Set<string>): string {
  const base = `having_${agg}_${path.replaceAll(".", "_")}`;
  let id = base;
  for (let n = 2; taken.has(id); n++) id = `${base}_${n}`;
  return id;
}

function havingToKeel(
  group: ReportFilterGroup | null | undefined,
  columns: Column[],
): Filter | undefined {
  if (!group) return undefined;
  const taken = new Set(columns.map((column) => column.id));
  const measureFor = (path: string, agg: string): string => {
    const existing = columns.find(
      (column) =>
        column.measure &&
        column.measure.agg === agg &&
        (column.measure.field ?? "") === path &&
        !column.measure.filter &&
        !column.transform,
    );
    if (existing) return existing.id;
    const id = havingColumnId(agg, path, taken);
    taken.add(id);
    columns.push({
      id,
      hidden: true,
      measure: { agg: agg as Agg, ...(path ? { field: path } : {}) },
    });
    return id;
  };
  const convert = (node: ReportFilterGroup): Filter | undefined => {
    const children: Filter[] = [];
    for (const leaf of node.filters ?? []) {
      if (leaf.raw) {
        children.push(leaf.raw as Filter);
        continue;
      }
      const condition: Filter = {
        column: measureFor(refPath(leaf.ref), leaf.agg ?? "count"),
        op: toKeelOp(leaf.operator),
      };
      if (leaf.param) condition.param = leaf.param;
      else if (leaf.value !== undefined && leaf.value !== null) condition.value = leaf.value;
      children.push(condition);
    }
    for (const nested of node.groups ?? []) {
      const converted = convert(nested);
      if (converted) children.push(converted);
    }
    if (children.length === 0) return undefined;
    return node.op === "or" ? { or: children } : { and: children };
  };
  return convert(group);
}

function havingFromKeel(
  filter: Filter | undefined,
  columns: Column[],
  generated: Set<string>,
): ReportFilterGroup | undefined {
  if (!filter) return undefined;
  const byId = new Map(columns.map((column) => [column.id, column]));
  const leaf = (node: Filter): ReportFieldFilter => {
    const column = node.column ? byId.get(node.column) : undefined;
    if (!column?.measure || column.measure.filter || column.transform) {
      return { ref: { field: "" }, operator: fromKeelOp(node.op), raw: node };
    }
    if (column.hidden) generated.add(column.id);
    const out: ReportFieldFilter = {
      ref: pathRef(column.measure.field),
      agg: column.measure.agg as ReportAggregation,
      operator: fromKeelOp(node.op),
    };
    if (node.param) out.param = node.param;
    if (node.value !== undefined) out.value = node.value;
    return out;
  };
  const convert = (node: Filter): ReportFilterGroup => {
    const children = node.and ?? node.or;
    if (!children) return { op: "and", filters: [leaf(node)] };
    const group: ReportFilterGroup = { op: node.or ? "or" : "and", filters: [], groups: [] };
    for (const child of children) {
      if (child.and || child.or) group.groups?.push(convert(child));
      else group.filters?.push(leaf(child));
    }
    return group;
  };
  return convert(filter);
}

function allowedToKeel(type: string, values: string[] | undefined): unknown[] | undefined {
  if (!values || values.length === 0) return undefined;
  return values.map((value) => {
    if ((type === "int" || type === "decimal" || type === "float") && value.trim() !== "") {
      const parsed = Number(value);
      if (Number.isFinite(parsed)) return parsed;
    }
    if (type === "bool" && (value === "true" || value === "false")) return value === "true";
    return value;
  });
}

export function parameterToKeel(param: ReportParameterDef): Parameter {
  const out: Parameter = { name: param.name, type: param.type as ValueType };
  if (param.label) out.label = param.label;
  if (param.required) out.required = true;
  if (param.multi) out.multi = true;
  if (param.default !== undefined && param.default !== null && param.default !== "") {
    out.default = param.default;
  }
  const allowed = allowedToKeel(param.type, param.allowedValues);
  if (allowed) out.allowed = allowed;
  if (param.type === "ref" && param.refEntity) out.entity = param.refEntity;
  return out;
}

export function parameterFromKeel(param: Parameter): ReportParameterDef {
  return {
    name: param.name,
    label: param.label,
    type: param.type,
    required: param.required ?? false,
    default: param.default,
    multi: param.multi,
    allowedValues: param.allowed?.map((value) => String(value)),
    refEntity: param.entity,
  };
}

function chartToKeel(chart: ReportChartSpec): Chart {
  const out: Chart = { id: chart.id, type: chart.type };
  if (chart.title) out.title = chart.title;
  if (chart.xColumnId) out.x = chart.xColumnId;
  if (chart.seriesIds && chart.seriesIds.length > 0) out.series = chart.seriesIds;
  if (chart.stacked) out.stacked = true;
  if (chart.hideLegend) out.hideLegend = true;
  if (chart.showValues) out.showValues = true;
  if (chart.curved) out.curved = true;
  if (chart.limit) out.limit = chart.limit;
  if (chart.compareId) out.compare = chart.compareId;
  if (chart.latColumnId) out.lat = chart.latColumnId;
  if (chart.lngColumnId) out.lng = chart.lngColumnId;
  if (chart.labelColumnId) out.pointLabel = chart.labelColumnId;
  if (chart.goal && (chart.goal.value !== undefined || chart.goal.columnId)) {
    out.goal = {
      ...(chart.goal.value !== undefined ? { value: chart.goal.value } : {}),
      ...(chart.goal.columnId ? { column: chart.goal.columnId } : {}),
      ...(chart.goal.label ? { label: chart.goal.label } : {}),
    };
  }
  return out;
}

function chartFromKeel(chart: Chart): ReportChartSpec {
  return {
    id: chart.id,
    type: chart.type,
    title: chart.title,
    xColumnId: chart.x,
    seriesIds: chart.series,
    stacked: chart.stacked,
    hideLegend: chart.hideLegend,
    showValues: chart.showValues,
    curved: chart.curved,
    limit: chart.limit,
    compareId: chart.compare,
    latColumnId: chart.lat,
    lngColumnId: chart.lng,
    labelColumnId: chart.pointLabel,
    goal: chart.goal
      ? { value: chart.goal.value, columnId: chart.goal.column, label: chart.goal.label }
      : undefined,
  };
}

export function toKeelDefinition(ir: ReportIR): Definition {
  const columns = ir.columns.map(columnToKeel);
  const definition: Definition = { version: 2, entity: ir.entity, columns };
  const filters = groupToKeel(ir.filters);
  if (filters) definition.filters = filters;
  const having = havingToKeel(ir.having, columns);
  if (having) definition.having = having;
  const extensions = ir.extensions ?? {};
  if (ir.totals || (extensions.subtotals?.length ?? 0) > 0) {
    definition.groupings = {
      ...(ir.totals ? { total: true } : {}),
      ...(extensions.subtotals?.length ? { subtotals: extensions.subtotals } : {}),
    };
  }
  if (extensions.top) definition.top = extensions.top as Definition["top"];
  const sort: Sort[] = (ir.sort ?? []).map((spec) =>
    spec.direction === "desc" ? { column: spec.columnId, desc: true } : { column: spec.columnId },
  );
  if (sort.length > 0) definition.sort = sort;
  if (extensions.limitParam) definition.limit = { param: extensions.limitParam };
  else if (ir.limit) definition.limit = ir.limit;
  if (ir.pivot) {
    definition.pivot = {
      field: refPath(ir.pivot.ref),
      values: extensions.pivotValuesParam
        ? { param: extensions.pivotValuesParam }
        : ir.pivot.values,
      columns: ir.pivot.measureIds,
      ...(ir.pivot.labels && ir.pivot.labels.length > 0 ? { labels: ir.pivot.labels } : {}),
      ...(ir.pivot.includeOther ? { other: true } : {}),
      ...(extensions.pivotTop
        ? { top: extensions.pivotTop as NonNullable<Definition["pivot"]>["top"] }
        : {}),
    };
  }
  const parameters = (ir.parameters ?? []).map(parameterToKeel);
  if (parameters.length > 0) definition.parameters = parameters;
  const charts = (ir.charts ?? []).map(chartToKeel);
  if (charts.length > 0) definition.charts = charts;
  return definition;
}

function columnFromKeel(column: Column): ReportColumnSpec {
  const base: ReportColumnSpec = {
    id: column.id,
    kind: "computed",
    label: column.label,
    transform: column.transform as ReportColumnSpec["transform"],
    display: column.display as ReportColumnSpec["display"],
    hidden: column.hidden || undefined,
  };
  if (column.dimension) {
    const bucket = column.dimension.bucket;
    return {
      ...base,
      kind: "dimension",
      ref: pathRef(column.dimension.field),
      bucket: typeof bucket === "string" ? (bucket as ReportDateBucket) : undefined,
      bucketParam: bucket && typeof bucket === "object" ? bucket.param : undefined,
      band: column.dimension.band,
    };
  }
  if (column.measure) {
    return {
      ...base,
      kind: "measure",
      ref: column.measure.field ? pathRef(column.measure.field) : undefined,
      agg: column.measure.agg as ReportAggregation,
      filter: groupFromKeel(column.measure.filter),
    };
  }
  if (column.window) return { ...base, window: column.window };
  const text = typeof column.calc === "string" ? column.calc : undefined;
  const computed = text !== undefined ? formulaToComputed(text, column.format) : undefined;
  if (computed) return { ...base, computed };
  return { ...base, calc: column.calc, format: column.format };
}

export function fromKeelDefinition(raw: unknown): ReportIR | null {
  if (!raw || typeof raw !== "object") return null;
  const definition = raw as Definition;
  if (typeof definition.entity !== "string" || !Array.isArray(definition.columns)) return null;
  const generated = new Set<string>();
  const having = havingFromKeel(definition.having, definition.columns, generated);
  const columns = definition.columns
    .filter((column) => !generated.has(column.id) || !column.id.startsWith("having_"))
    .map(columnFromKeel);
  const limit = definition.limit;
  const pivot = definition.pivot;
  const extensions: NonNullable<ReportIR["extensions"]> = {};
  if (definition.groupings?.subtotals?.length)
    extensions.subtotals = definition.groupings.subtotals;
  if (definition.top) extensions.top = definition.top;
  if (limit && typeof limit === "object") extensions.limitParam = limit.param;
  if (pivot?.values && !Array.isArray(pivot.values))
    extensions.pivotValuesParam = pivot.values.param;
  if (pivot?.top) extensions.pivotTop = pivot.top;
  return {
    entity: definition.entity,
    columns,
    filters: groupFromKeel(definition.filters) ?? null,
    having: having ?? null,
    sort: (definition.sort ?? []).map((spec) => ({
      columnId: spec.column,
      direction: spec.desc ? "desc" : "asc",
    })),
    limit: typeof limit === "number" ? limit : undefined,
    pivot: pivot
      ? {
          ref: pathRef(pivot.field),
          values: Array.isArray(pivot.values) ? pivot.values : [],
          labels: pivot.labels,
          measureIds: pivot.columns,
          includeOther: pivot.other,
        }
      : null,
    parameters: (definition.parameters ?? []).map(parameterFromKeel),
    totals: definition.groupings?.total ?? false,
    charts: (definition.charts ?? []).map(chartFromKeel),
    extensions: Object.keys(extensions).length > 0 ? extensions : undefined,
  };
}

export function dashboardFilterToKeel(filter: ReportDashboardFilter): DashboardFilter {
  return {
    id: filter.id,
    ...(filter.label ? { label: filter.label } : {}),
    entity: filter.entity,
    field: refPath(filter.ref),
    op: toKeelOp(filter.operator),
    ...(filter.default !== undefined && filter.default !== null ? { default: filter.default } : {}),
  };
}

export function dashboardFilterFromKeel(filter: DashboardFilter): ReportDashboardFilter {
  return {
    id: filter.id,
    label: filter.label,
    entity: filter.entity,
    ref: pathRef(filter.field),
    operator: fromKeelOp(filter.op),
    default: filter.default,
  };
}
