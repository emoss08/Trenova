import type {
  ReportCatalog,
  ReportCatalogEntity,
  ReportCatalogField,
  ReportIrInput,
} from "@/lib/graphql/reports";
import {
  PIVOT_OTHER_VALUE,
  REPORT_IR_VERSION,
  bandIsSet,
  computedOperand,
  computedOperandIsTarget,
  reportOutputColumnIds,
  type ReportBandSpec,
  type ReportComputedOperand,
  type ReportColumnSpec,
  type ReportDisplaySpec,
  type ReportFieldRef,
  type ReportFilterGroup,
  type ReportIR,
  type ReportParameterDef,
  type ReportTransformSpec,
} from "@/types/report";

export const MAX_PATH_DEPTH = 3;

export type CatalogIndex = {
  catalog: ReportCatalog;
  entities: Map<string, ReportCatalogEntity>;
};

export function buildCatalogIndex(catalog: ReportCatalog): CatalogIndex {
  return {
    catalog,
    entities: new Map(catalog.entities.map((entity) => [entity.key, entity])),
  };
}

export function resolvePathEntity(
  index: CatalogIndex,
  entityKey: string,
  path: string[] | undefined,
): ReportCatalogEntity | undefined {
  let current = index.entities.get(entityKey);
  for (const edgeName of path ?? []) {
    if (!current) return undefined;
    const edge = current.edges.find((e) => e.name === edgeName);
    if (!edge || !edge.traversable) return undefined;
    current = index.entities.get(edge.target);
  }
  return current;
}

export function resolveField(
  index: CatalogIndex,
  entityKey: string,
  ref: ReportFieldRef,
): ReportCatalogField | undefined {
  const entity = resolvePathEntity(index, entityKey, ref.path);
  return entity?.fields.find((field) => field.key === ref.field);
}

/**
 * The catalog entity a reference field points at. Ref fields carry no target of
 * their own, but the relationship beside them does — `serviceTypeId` sits next
 * to the `serviceType` edge — which is what lets a filter on an id offer real
 * records instead of asking for a raw identifier.
 */
export function refTargetEntity(
  index: CatalogIndex,
  entityKey: string,
  ref: ReportFieldRef,
): string | undefined {
  const owner = resolvePathEntity(index, entityKey, ref.path);
  if (!owner) return undefined;

  const base = ref.field.endsWith("Id") ? ref.field.slice(0, -2) : ref.field;
  return owner.edges.find((edge) => edge.name === base)?.target;
}

/**
 * Prefers the foreign key beside a relationship when offering something to
 * filter on. A report that groups by "Service Type › Code" is best narrowed by
 * picking actual service types, not by typing a code — so the grouping ref is
 * swapped for the `serviceTypeId` reference on the owning entity.
 */
export function preferReferenceField(
  index: CatalogIndex,
  entityKey: string,
  ref: ReportFieldRef,
): ReportFieldRef {
  const path = ref.path ?? [];
  if (path.length === 0) return ref;

  // Only identity-ish text collapses to the relationship that names it.
  // A number, date, or enum on a related record — a credit limit, a status —
  // means itself, so filtering it must stay on that field.
  if (resolveField(index, entityKey, ref)?.type !== "string") return ref;

  const parentPath = path.slice(0, -1);
  const lastEdge = path[path.length - 1];
  const owner = resolvePathEntity(index, entityKey, parentPath);

  const edge = owner?.edges.find((candidate) => candidate.name === lastEdge);
  if (!owner || !edge || edge.cardinality !== "one") return ref;

  const foreignKey = `${lastEdge}Id`;
  const field = owner.fields.find(
    (candidate) =>
      candidate.key === foreignKey &&
      candidate.type === "ref" &&
      candidate.filterable &&
      candidate.accessible,
  );

  return field ? { path: parentPath.length > 0 ? parentPath : undefined, field: foreignKey } : ref;
}

export function pathCrossesToMany(
  index: CatalogIndex,
  entityKey: string,
  path: string[] | undefined,
): boolean {
  let current = index.entities.get(entityKey);
  for (const edgeName of path ?? []) {
    if (!current) return false;
    const edge = current.edges.find((e) => e.name === edgeName);
    if (!edge) return false;
    if (edge.cardinality !== "one") return true;
    current = index.entities.get(edge.target);
  }
  return false;
}

export function refLabel(index: CatalogIndex, entityKey: string, ref: ReportFieldRef): string {
  const segments: string[] = [];
  let current = index.entities.get(entityKey);
  for (const edgeName of ref.path ?? []) {
    const edge = current?.edges.find((e) => e.name === edgeName);
    segments.push(edge?.label ?? edgeName);
    current = edge ? index.entities.get(edge.target) : undefined;
  }
  const field = current?.fields.find((f) => f.key === ref.field);
  segments.push(field?.label ?? ref.field);
  return segments.join(" › ");
}

export function emptyIR(entity: string): ReportIR {
  return {
    irVersion: REPORT_IR_VERSION,
    entity,
    columns: [],
    filters: { op: "and", filters: [] },
    sort: [],
    parameters: [],
  };
}

export function uniqueColumnId(ir: ReportIR, base: string): string {
  const normalized = base.replace(/[^a-zA-Z0-9_]/g, "_").toLowerCase() || "column";
  if (!ir.columns.some((column) => column.id === normalized)) return normalized;
  let suffix = 2;
  while (ir.columns.some((column) => column.id === `${normalized}_${suffix}`)) {
    suffix += 1;
  }
  return `${normalized}_${suffix}`;
}

export type MeasureColumnSpec = ReportColumnSpec & { ref: ReportFieldRef };

export function measureColumns(ir: ReportIR): MeasureColumnSpec[] {
  return ir.columns.filter(
    (column): column is MeasureColumnSpec => column.kind === "measure" && !!column.ref,
  );
}

// Aggregate-level columns (measures and calculations) — everything that can
// be pivoted or sorted as a value column.
export function aggregateColumns(ir: ReportIR): ReportColumnSpec[] {
  return ir.columns.filter((column) => column.kind !== "dimension");
}

export function columnDisplayLabel(
  index: CatalogIndex,
  ir: ReportIR,
  column: ReportColumnSpec,
): string {
  if (column.label) return column.label;
  return defaultColumnLabel(index, ir, column);
}

// absorbs mirrors the server's label composition: "Fleet Code" already conveys
// "Code", so the pair renders as "Fleet Code", not "Fleet Code Code".
function absorbs(outer: string, inner: string): boolean {
  if (!outer || !inner) return false;
  const a = outer.toLowerCase();
  const b = inner.toLowerCase();
  return a === b || a.endsWith(` ${b}`) || a.startsWith(`${b} `);
}

function joinLabelSegments(segments: string[]): string {
  const parts: string[] = [];
  for (const segment of segments) {
    if (!segment) continue;
    const last = parts[parts.length - 1];
    if (last && absorbs(last, segment)) continue;
    if (last && absorbs(segment, last)) {
      parts[parts.length - 1] = segment;
      continue;
    }
    parts.push(segment);
  }
  return parts.join(" ");
}

function qualifiedFieldLabel(index: CatalogIndex, entityKey: string, ref: ReportFieldRef): string {
  const path = ref.path ?? [];
  let current = index.entities.get(entityKey);
  let lastEdgeLabel = "";

  for (const edgeName of path) {
    const edge = current?.edges.find((e) => e.name === edgeName);
    lastEdgeLabel = edge?.label ?? edgeName;
    current = edge ? index.entities.get(edge.target) : undefined;
  }

  const field = current?.fields.find((f) => f.key === ref.field);
  return joinLabelSegments([lastEdgeLabel, field?.label ?? ref.field]);
}

function prefixed(prefix: string, base: string): string {
  return base.toLowerCase().startsWith(prefix.toLowerCase()) ? base : `${prefix} ${base}`;
}

const BUCKET_SUFFIXES: Record<NonNullable<ReportColumnSpec["bucket"]>, string> = {
  day: "Day",
  week: "Week",
  month: "Month",
  quarter: "Quarter",
  year: "Year",
};

/**
 * The header the server will generate when the column carries no explicit
 * label — shown as the placeholder so renaming starts from what you'd get.
 */
export function defaultColumnLabel(
  index: CatalogIndex,
  ir: ReportIR,
  column: ReportColumnSpec,
): string {
  if (!column.ref) return "Calculation";

  const entity = resolvePathEntity(index, ir.entity, column.ref.path);
  const field = resolveField(index, ir.entity, column.ref);
  const base = qualifiedFieldLabel(index, ir.entity, column.ref);

  if (column.kind === "measure") {
    const identity = column.ref.field === "id";
    switch (column.agg) {
      case "sum":
        return prefixed("Total", base);
      case "avg":
        return prefixed("Average", base);
      case "min":
        return prefixed(field?.type === "epoch" ? "Earliest" : "Minimum", base);
      case "max":
        return prefixed(field?.type === "epoch" ? "Latest" : "Maximum", base);
      case "count":
        return identity ? `${entity?.pluralLabel ?? base} Count` : `${base} Count`;
      case "count_distinct":
        return `Distinct ${identity ? (entity?.pluralLabel ?? base) : base}`;
      default:
        return base;
    }
  }

  if (column.kind === "dimension" && column.bucket) {
    return `${base} (${BUCKET_SUFFIXES[column.bucket]})`;
  }
  if (column.kind === "dimension" && bandIsSet(column.band)) {
    return `${base} (Range)`;
  }

  return base;
}

// What the server names a pivot column when the author gave it no override.
export function pivotValueLabel(field: ReportCatalogField | undefined, value: string): string {
  const enumValue = field?.enumValues.find((candidate) => candidate.value === value);
  if (enumValue) return enumValue.label;
  if (field?.type === "bool") return value === "true" ? "Yes" : "No";
  return value;
}

export type OutputColumnChoice = {
  id: string;
  label: string;
  isDim: boolean;
};

/**
 * The columns the report will actually return, labelled the way the header
 * will read — what charts, sorting, and KPI tiles pick from.
 */
export function outputColumnChoices(index: CatalogIndex, ir: ReportIR): OutputColumnChoice[] {
  const pivotField = ir.pivot ? resolveField(index, ir.entity, ir.pivot.ref) : undefined;
  const byId = new Map(ir.columns.map((column) => [column.id, column]));

  return reportOutputColumnIds(ir).map((output) => {
    const [baseId, pivotValue] = splitOutputId(output.id);
    const column = byId.get(baseId);
    const base = column ? columnDisplayLabel(index, ir, column) : baseId;

    if (pivotValue === undefined) {
      return { id: output.id, label: base, isDim: output.isDim };
    }

    const valueIndex = ir.pivot?.values.indexOf(pivotValue) ?? -1;
    const override = valueIndex >= 0 ? ir.pivot?.labels?.[valueIndex] : undefined;
    const suffix =
      override ||
      (pivotValue === PIVOT_OTHER_VALUE ? "Other" : pivotValueLabel(pivotField, pivotValue));

    return { id: output.id, label: `${base} (${suffix})`, isDim: output.isDim };
  });
}

export function splitOutputId(outputId: string): [string, string | undefined] {
  const separator = outputId.indexOf(":");
  if (separator < 0) return [outputId, undefined];
  return [outputId.slice(0, separator), outputId.slice(separator + 1)];
}

function sameRef(a: ReportFieldRef | undefined, b: ReportFieldRef): boolean {
  return !!a && a.field === b.field && (a.path ?? []).join(".") === (b.path ?? []).join(".");
}

// withColumns replaces the column set and prunes everything that referenced a
// column which no longer exists (or is no longer a measure): computed columns
// whose operands vanished, sort specs, pivot measure ids, and HAVING filters.
// Without this, removing a column leaves the IR failing server validation
// with errors about phantom columns.
export function withColumns(ir: ReportIR, columns: ReportColumnSpec[]): ReportIR {
  const measures = columns.filter((column) => column.kind === "measure");
  const measureIdSet = new Set(measures.map((column) => column.id));

  // A constant operand survives on its own; only the column side can dangle.
  const operandSurvives = (operand: ReportComputedOperand) =>
    computedOperandIsTarget(operand) || measureIdSet.has(operand.columnId ?? "");

  const survivors = columns.filter(
    (column) =>
      column.kind !== "computed" ||
      (!!column.computed &&
        operandSurvives(computedOperand(column.computed, "left")) &&
        operandSurvives(computedOperand(column.computed, "right"))),
  );
  const aggregateIds = new Set(
    survivors.filter((column) => column.kind !== "dimension").map((column) => column.id),
  );

  let pivot = ir.pivot ?? null;
  if (pivot) {
    const measureIds = pivot.measureIds.filter((id) => aggregateIds.has(id));
    pivot = measureIds.length > 0 ? { ...pivot, measureIds } : null;
  }

  let having = ir.having ?? null;
  if (having) {
    const filters = (having.filters ?? []).filter((filter) =>
      measures.some((column) => sameRef(column.ref, filter.ref) && column.agg === filter.agg),
    );
    having = filters.length > 0 ? { ...having, filters } : null;
  }

  return pruneOutputReferences({ ...ir, columns: survivors, pivot, having });
}

/**
 * Sorting and charts address *output* columns, so a pivoted measure is only
 * reachable one bucket at a time. Anything pointing at a column the compiled
 * query would no longer return is dropped here rather than failing validation.
 */
export function pruneOutputReferences(ir: ReportIR): ReportIR {
  const outputs = reportOutputColumnIds(ir);
  const outputIds = new Set(outputs.map((output) => output.id));
  const dimensionIds = new Set(outputs.filter((output) => output.isDim).map((output) => output.id));

  const sort = (ir.sort ?? []).filter((spec) => outputIds.has(spec.columnId));

  const charts = (ir.charts ?? [])
    .map((chart) => ({
      ...chart,
      xColumnId: chart.xColumnId && outputIds.has(chart.xColumnId) ? chart.xColumnId : undefined,
      seriesIds: (chart.seriesIds ?? []).filter((id) => outputIds.has(id) && !dimensionIds.has(id)),
      compareId: chart.compareId && outputIds.has(chart.compareId) ? chart.compareId : undefined,
      goal: chart.goal?.columnId && !outputIds.has(chart.goal.columnId) ? undefined : chart.goal,
    }))
    .filter((chart) => chart.seriesIds.length > 0);

  const totals = ir.totals && !(ir.having?.filters?.length ?? 0) ? ir.totals : false;

  return { ...ir, sort, charts, totals };
}

function rewriteParamBindings(
  group: ReportFilterGroup | null | undefined,
  validNames: Set<string>,
  rename?: { from: string; to: string },
): ReportFilterGroup | null | undefined {
  if (!group) return group;
  return {
    ...group,
    filters: (group.filters ?? []).map((filter) => {
      if (!filter.param) return filter;
      if (rename && filter.param === rename.from) {
        return { ...filter, param: rename.to };
      }
      if (!validNames.has(filter.param)) {
        return { ...filter, param: undefined };
      }
      return filter;
    }),
    groups: (group.groups ?? []).map(
      (nested) => rewriteParamBindings(nested, validNames, rename) as ReportFilterGroup,
    ),
  };
}

// withParameters replaces the parameter set, following renames through filter
// bindings and unbinding filters whose parameter was deleted.
export function withParameters(
  ir: ReportIR,
  parameters: ReportIR["parameters"],
  rename?: { from: string; to: string },
): ReportIR {
  const validNames = new Set((parameters ?? []).map((param) => param.name));
  return {
    ...ir,
    parameters,
    filters: rewriteParamBindings(ir.filters, validNames, rename) ?? ir.filters,
    having: rewriteParamBindings(ir.having, validNames, rename) ?? ir.having,
  };
}

function pruneFilterGroup(
  group: ReportFilterGroup | null | undefined,
): ReportFilterGroup | undefined {
  if (!group) return undefined;
  const filters = group.filters ?? [];
  const groups = (group.groups ?? [])
    .map((nested) => pruneFilterGroup(nested))
    .filter((nested): nested is ReportFilterGroup => nested !== undefined);
  if (filters.length === 0 && groups.length === 0) return undefined;
  return { op: group.op, filters, groups };
}

type FilterGroupInput = NonNullable<ReportIrInput["filters"]>;

function filterGroupToInput(
  group: ReportFilterGroup | null | undefined,
): FilterGroupInput | undefined {
  const pruned = pruneFilterGroup(group);
  if (!pruned) return undefined;
  return {
    op: pruned.op,
    filters: (pruned.filters ?? []).map((filter) => ({
      ref: { path: filter.ref.path, field: filter.ref.field },
      operator: filter.operator,
      value: filter.value ?? undefined,
      param: filter.param || undefined,
      agg: filter.agg || undefined,
      transform: transformToInput(filter.transform),
    })),
    groups: (pruned.groups ?? [])
      .map((nested) => filterGroupToInput(nested))
      .filter((nested): nested is FilterGroupInput => nested !== undefined),
  };
}

type TransformInput = NonNullable<ReportIrInput["columns"][number]["transform"]>;

function transformToInput(transform: ReportTransformSpec | undefined): TransformInput | undefined {
  if (!transform) return undefined;
  return {
    op: transform.op,
    precision: transform.precision ?? undefined,
    factor: transform.factor ?? undefined,
  };
}

type BandInput = NonNullable<ReportIrInput["columns"][number]["band"]>;

// Only the mode the author chose is sent — the server rejects a band carrying
// both a width and a boundary list.
function bandToInput(band: ReportBandSpec | undefined): BandInput | undefined {
  if (!bandIsSet(band)) return undefined;
  if (band?.edges && band.edges.length > 0) return { edges: band.edges };
  return { width: band?.width };
}

type DisplayInput = NonNullable<ReportIrInput["columns"][number]["display"]>;

// Only the keys the author actually set are sent; everything else stays at the
// server-resolved default for the column's type and format hint.
function displayToInput(display: ReportDisplaySpec | undefined): DisplayInput | undefined {
  if (!display) return undefined;

  const input: DisplayInput = {
    style: display.style || undefined,
    decimals: display.decimals ?? undefined,
    grouping: display.grouping ?? undefined,
    currency: display.currency || undefined,
    negative: display.negative || undefined,
    notation: display.notation || undefined,
    prefix: display.prefix || undefined,
    suffix: display.suffix || undefined,
    dateStyle: display.dateStyle || undefined,
    boolStyle: display.boolStyle || undefined,
    durationUnit: display.durationUnit || undefined,
    durationStyle: display.durationStyle || undefined,
    nullText: display.nullText || undefined,
    rules:
      display.rules && display.rules.length > 0
        ? display.rules.map((rule) => ({
            op: rule.op,
            value: rule.value,
            upper: rule.upper ?? undefined,
            tone: rule.tone,
          }))
        : undefined,
  };

  const hasOverride = Object.values(input).some((value) => value !== undefined);
  return hasOverride ? input : undefined;
}

// Converts the persisted IR shape into the GraphQL input the save/preview
// operations expect. Empty filter groups are pruned so the server never sees
// vacuous groups.
export function irToInput(ir: ReportIR): ReportIrInput {
  return {
    entity: ir.entity,
    columns: ir.columns.map((column) => ({
      id: column.id,
      ref: column.ref ? { path: column.ref.path, field: column.ref.field } : undefined,
      kind: column.kind,
      agg: column.kind === "measure" ? column.agg : undefined,
      bucket: column.bucket || undefined,
      band: bandToInput(column.band),
      label: column.label || undefined,
      computed:
        column.kind === "computed" && column.computed
          ? {
              op: column.computed.op,
              leftId: column.computed.leftId || undefined,
              rightId: column.computed.rightId || undefined,
              leftValue: column.computed.leftValue,
              rightValue: column.computed.rightValue,
              format:
                column.computed.format && column.computed.format !== "none"
                  ? column.computed.format
                  : undefined,
            }
          : undefined,
      transform: transformToInput(column.transform),
      display: displayToInput(column.display),
      filter: column.kind === "measure" ? filterGroupToInput(column.filter) : undefined,
    })),
    filters: filterGroupToInput(ir.filters),
    having: filterGroupToInput(ir.having),
    sort: (ir.sort ?? []).map((sortSpec) => ({
      columnId: sortSpec.columnId,
      direction: sortSpec.direction,
    })),
    limit: ir.limit || undefined,
    pivot: ir.pivot
      ? {
          ref: { path: ir.pivot.ref.path, field: ir.pivot.ref.field },
          values: ir.pivot.values,
          labels: ir.pivot.labels && ir.pivot.labels.length > 0 ? ir.pivot.labels : undefined,
          measureIds: ir.pivot.measureIds,
          includeOther: ir.pivot.includeOther ?? false,
        }
      : undefined,
    parameters: (ir.parameters ?? []).map(parameterToInput),
    totals: ir.totals ?? false,
    charts: (ir.charts ?? []).map((chart) => ({
      id: chart.id,
      type: chart.type,
      title: chart.title || undefined,
      xColumnId: chart.xColumnId || undefined,
      seriesIds: chart.seriesIds ?? [],
      stacked: chart.stacked ?? false,
      hideLegend: chart.hideLegend ?? false,
      showValues: chart.showValues ?? false,
      curved: chart.curved ?? false,
      limit: chart.limit || undefined,
      compareId: chart.compareId || undefined,
      latColumnId: chart.latColumnId || undefined,
      lngColumnId: chart.lngColumnId || undefined,
      labelColumnId: chart.labelColumnId || undefined,
      goal:
        chart.goal && (chart.goal.value !== undefined || chart.goal.columnId)
          ? {
              value: chart.goal.value ?? undefined,
              columnId: chart.goal.columnId || undefined,
              label: chart.goal.label || undefined,
            }
          : undefined,
    })),
  };
}

export function parameterToInput(
  param: ReportParameterDef,
): NonNullable<ReportIrInput["parameters"]>[number] {
  return {
    name: param.name,
    label: param.label || undefined,
    type: param.type,
    required: param.required,
    default: param.default ?? undefined,
    multi: param.multi ?? false,
    allowedValues:
      param.allowedValues && param.allowedValues.length > 0 ? param.allowedValues : undefined,
    refEntity: param.type === "ref" ? param.refEntity || undefined : undefined,
  };
}

export function defaultPreviewParams(ir: ReportIR): Record<string, unknown> | undefined {
  const parameters = ir.parameters ?? [];
  if (parameters.length === 0) return undefined;
  const params: Record<string, unknown> = {};
  for (const param of parameters) {
    if (param.default !== undefined && param.default !== null) {
      params[param.name] = param.default;
    }
  }
  return params;
}

// A definition is previewable once every required parameter has a value —
// the compiler rejects unbound required params.
export function previewReady(ir: ReportIR): boolean {
  if (!ir.entity || ir.columns.length === 0) return false;
  const params = defaultPreviewParams(ir) ?? {};
  return (ir.parameters ?? []).every(
    (param) => !param.required || params[param.name] !== undefined,
  );
}
