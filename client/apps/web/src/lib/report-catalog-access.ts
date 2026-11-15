import type {
  ReportCatalog,
  ReportCatalogEdge,
  ReportCatalogEntity,
  ReportCatalogField,
} from "@/lib/graphql/reports";
import type { ReportFieldRef, ReportFilterGroup, ReportIR } from "@/types/report";

export type ReportFieldAccess = ReportCatalogField["access"];

export type ReportPlanWithheld = ReportCatalog["withheldByPlan"][number];

/** The catalog's entities by key, as the builder indexes them. */
type CatalogIndex = {
  entities: ReadonlyMap<string, ReportCatalogEntity>;
  catalog?: Pick<ReportCatalog, "withheldByPlan">;
};

/**
 * What the organization's plan leaves out of the catalog under an entity: the
 * entity itself when name is omitted, else one of its fields or edges.
 */
export function planWithheld(
  index: CatalogIndex,
  entity: string,
  name?: string,
): ReportPlanWithheld | undefined {
  return index.catalog?.withheldByPlan.find(
    (item) => item.entity === entity && (item.name ?? undefined) === name,
  );
}

/**
 * Whether a report may use a field. The server decides it by the rule its
 * compiler applies, so an available field compiles and any other is refused.
 */
export function isFieldAvailable(field: Pick<ReportCatalogField, "access">): boolean {
  return field.access === "Available";
}

export function isEdgeAvailable(edge: Pick<ReportCatalogEdge, "access">): boolean {
  return edge.access === "Available";
}

/** The entities a report can start from: the kinds the caller reads. */
export function rootEntities(catalog: ReportCatalog): ReportCatalogEntity[] {
  return catalog.entities.filter((entity) => entity.readable);
}

/**
 * The fields of an entity the builder lists. A kind the caller reads lists
 * every field, the unavailable ones disabled with their reason. A kind
 * reached only through a join lists its reference view alone: the rest
 * needs the kind's read and is summed up in one note rather than listed.
 */
export function listedFields(entity: ReportCatalogEntity): ReportCatalogField[] {
  if (entity.readable) return entity.fields;
  return entity.fields.filter((field) => field.inReferenceView);
}

export type UnavailableRef = {
  ref: ReportFieldRef;
  /** The field's label, or its key when the catalog no longer has it. */
  label: string;
  /** The entity the field is on, or the root when the path is broken. */
  entityLabel: string;
  access: ReportFieldAccess | "Missing" | "NotOnPlan";
  /** The add-on that holds it and the feature it belongs to, for NotOnPlan. */
  addOnName?: string;
  feature?: string;
  featureName?: string;
  /** The kind whose read is missing, for NeedsKindRead. */
  kindLabel?: string;
  /** The protected group, for NeedsFieldGroup. */
  group?: string;
};

type Resolved = {
  entity?: ReportCatalogEntity;
  field?: ReportCatalogField;
  blocked?: {
    access: ReportFieldAccess | "Missing";
    kindLabel?: string;
    entity?: ReportCatalogEntity;
  };
  plan?: ReportPlanWithheld;
};

function resolve(index: CatalogIndex, root: string, ref: ReportFieldRef): Resolved {
  let current = index.entities.get(root);
  if (!current) {
    const plan = planWithheld(index, root);
    if (plan) return { plan };
  }
  for (const name of ref.path ?? []) {
    const edge = current?.edges.find((candidate) => candidate.name === name);
    if (current && !edge) {
      const plan = planWithheld(index, current.key, name);
      if (plan) return { entity: current, plan };
    }
    if (!current || !edge) return { entity: current, blocked: { access: "Missing" } };
    if (!isEdgeAvailable(edge)) {
      return {
        entity: current,
        blocked: {
          access: edge.access,
          kindLabel: edge.needsReadOf?.label,
          entity: current,
        },
      };
    }
    const next = index.entities.get(edge.target);
    if (!next) {
      return {
        entity: current,
        blocked: { access: "NeedsKindRead", kindLabel: edge.label, entity: current },
      };
    }
    current = next;
  }
  if (!current) return { blocked: { access: "Missing" } };
  const field = current.fields.find((candidate) => candidate.key === ref.field);
  if (!field) {
    const plan = planWithheld(index, current.key, ref.field);
    if (plan) return { entity: current, plan };
  }
  return { entity: current, field };
}

/** Why one ref of a report cannot be used, or undefined when it can. */
export function unavailableRef(
  index: CatalogIndex,
  root: string,
  ref: ReportFieldRef,
): UnavailableRef | undefined {
  const resolved = resolve(index, root, ref);
  const entityLabel = resolved.entity?.label ?? index.entities.get(root)?.label ?? root;

  if (resolved.plan) {
    const { plan } = resolved;
    return {
      ref,
      label: plan.name ? plan.label : ref.field,
      entityLabel: plan.entityLabel,
      access: "NotOnPlan",
      addOnName: plan.denial.addOnName,
      feature: plan.denial.feature,
      featureName: plan.denial.featureName,
    };
  }
  if (resolved.blocked) {
    return {
      ref,
      label: ref.field,
      entityLabel,
      access: resolved.blocked.access,
      kindLabel: resolved.blocked.kindLabel,
    };
  }
  const { entity, field } = resolved;
  if (!entity || !field) {
    return { ref, label: ref.field, entityLabel, access: "Missing" };
  }
  if (isFieldAvailable(field)) return undefined;
  return {
    ref,
    label: field.label,
    entityLabel: entity.label,
    access: field.access,
    kindLabel: field.access === "NeedsKindRead" ? entity.pluralLabel : undefined,
    group: field.protection?.group,
  };
}

function filterRefs(group: ReportFilterGroup | null | undefined, out: ReportFieldRef[]) {
  if (!group) return;
  for (const filter of group.filters ?? []) out.push(filter.ref);
  for (const nested of group.groups ?? []) filterRefs(nested, out);
}

/** Every field ref a report reads: its columns, filters, having and pivot. */
export function reportRefs(ir: ReportIR): ReportFieldRef[] {
  const refs: ReportFieldRef[] = [];
  for (const column of ir.columns) {
    if (column.ref) refs.push(column.ref);
    filterRefs(column.filter, refs);
  }
  filterRefs(ir.filters, refs);
  filterRefs(ir.having, refs);
  if (ir.pivot) refs.push(ir.pivot.ref);
  return refs;
}

function refKey(ref: ReportFieldRef): string {
  return [...(ref.path ?? []), ref.field].join(".");
}

/**
 * The refs a report uses that the caller may not, each once. A saved report
 * can name a field its reader's role no longer reaches, or that someone with
 * a wider role added.
 */
export function unavailableRefs(index: CatalogIndex, ir: ReportIR): UnavailableRef[] {
  const seen = new Set<string>();
  const out: UnavailableRef[] = [];
  for (const ref of reportRefs(ir)) {
    const key = refKey(ref);
    if (seen.has(key)) continue;
    seen.add(key);
    const problem = unavailableRef(index, ir.entity, ref);
    if (problem) out.push(problem);
  }
  return out;
}

function pruneFilters(
  group: ReportFilterGroup | null | undefined,
  blocked: Set<string>,
): ReportFilterGroup | undefined {
  if (!group) return undefined;
  const filters = (group.filters ?? []).filter((filter) => !blocked.has(refKey(filter.ref)));
  const groups = (group.groups ?? [])
    .map((nested) => pruneFilters(nested, blocked))
    .filter((nested): nested is ReportFilterGroup => nested !== undefined);
  if (filters.length === 0 && groups.length === 0) return undefined;
  return { ...group, filters, groups };
}

/**
 * Whether the fix for a ref is to take it out of the report. A ref the plan
 * leaves out is not: it comes back when the add-on is added.
 */
export function isRemovable(problem: UnavailableRef): boolean {
  return problem.access !== "NotOnPlan";
}

export type PlanAddOn = { feature: string; featureName: string; addOnName: string };

/** The add-ons a report's plan-withheld refs need, each once. */
export function planAddOns(problems: UnavailableRef[]): PlanAddOn[] {
  const out: PlanAddOn[] = [];
  for (const problem of problems) {
    if (problem.access !== "NotOnPlan" || !problem.feature) continue;
    const addOnName = problem.addOnName ?? "";
    const featureName = problem.featureName ?? "";
    const key = addOnName || featureName;
    if (out.some((addOn) => (addOn.addOnName || addOn.featureName) === key)) continue;
    out.push({ feature: problem.feature, featureName, addOnName });
  }
  return out;
}

/**
 * The report without the refs the caller may not use: their columns,
 * filters and pivot. Sorts and charts that pointed at a removed column are
 * left for pruneOutputReferences.
 */
export function withoutRefs(ir: ReportIR, refs: UnavailableRef[]): ReportIR {
  const blocked = new Set(refs.map((problem) => refKey(problem.ref)));
  return {
    ...ir,
    columns: ir.columns
      .filter((column) => !column.ref || !blocked.has(refKey(column.ref)))
      .map((column) =>
        column.filter ? { ...column, filter: pruneFilters(column.filter, blocked) } : column,
      ),
    filters: pruneFilters(ir.filters, blocked) ?? null,
    having: pruneFilters(ir.having, blocked) ?? null,
    pivot: ir.pivot && blocked.has(refKey(ir.pivot.ref)) ? null : ir.pivot,
  };
}
