import type {
  ReportCatalog,
  ReportCatalogEntity,
  ReportCatalogField,
  ReportIrInput,
} from "@/lib/graphql/reports";
import {
  REPORT_IR_VERSION,
  type ReportColumnSpec,
  type ReportFieldRef,
  type ReportFilterGroup,
  type ReportIR,
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
  if (column.ref) return refLabel(index, ir.entity, column.ref);
  return "Calculation";
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

  const survivors = columns.filter(
    (column) =>
      column.kind !== "computed" ||
      (!!column.computed &&
        measureIdSet.has(column.computed.leftId) &&
        measureIdSet.has(column.computed.rightId)),
  );
  const columnIds = new Set(survivors.map((column) => column.id));
  const aggregateIds = new Set(
    survivors.filter((column) => column.kind !== "dimension").map((column) => column.id),
  );

  const sort = (ir.sort ?? []).filter((spec) => columnIds.has(spec.columnId));

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

  return { ...ir, columns: survivors, sort, pivot, having };
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
    })),
    groups: (pruned.groups ?? [])
      .map((nested) => filterGroupToInput(nested))
      .filter((nested): nested is FilterGroupInput => nested !== undefined),
  };
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
      label: column.label || undefined,
      computed:
        column.kind === "computed" && column.computed
          ? {
              op: column.computed.op,
              leftId: column.computed.leftId,
              rightId: column.computed.rightId,
              format:
                column.computed.format && column.computed.format !== "none"
                  ? column.computed.format
                  : undefined,
            }
          : undefined,
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
          measureIds: ir.pivot.measureIds,
          includeOther: ir.pivot.includeOther ?? false,
        }
      : undefined,
    parameters: (ir.parameters ?? []).map((param) => ({
      name: param.name,
      label: param.label || undefined,
      type: param.type,
      required: param.required,
      default: param.default ?? undefined,
      multi: param.multi ?? false,
      allowedValues:
        param.allowedValues && param.allowedValues.length > 0 ? param.allowedValues : undefined,
      refEntity: param.type === "ref" ? param.refEntity || undefined : undefined,
    })),
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
