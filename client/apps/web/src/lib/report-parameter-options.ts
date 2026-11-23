import type { ReportCatalog } from "@/lib/graphql/reports";
import type {
  ReportFieldRef,
  ReportFilterGroup,
  ReportIR,
  ReportParameterDef,
} from "@/types/report";
import { codeWords } from "@trenova/shared/lib/code-words";

/** One allowed value of a parameter and the name a person reads for it. */
export type ReportParameterOption = { value: string; label: string };

/** The options of each parameter that has allowed values, by parameter name. */
export type ReportParameterOptions = Readonly<Record<string, readonly ReportParameterOption[]>>;

type ValueLabels = ReadonlyMap<string, string>;

/**
 * The options of one parameter in the order of its own allowed list, each named by `labels`
 * (the catalog's names for the enum field it filters) or, without one, by the code in words.
 */
export function parameterChoices(
  param: ReportParameterDef,
  labels?: ValueLabels,
): ReportParameterOption[] {
  return (param.allowedValues ?? []).map((value) => ({
    value,
    label: labels?.get(value) ?? codeWords(value),
  }));
}

export function hasAllowedValues(parameters: readonly ReportParameterDef[] | undefined): boolean {
  return (parameters ?? []).some((param) => (param.allowedValues?.length ?? 0) > 0);
}

function collectParamRefs(
  group: ReportFilterGroup | null | undefined,
  into: Map<string, ReportFieldRef[]>,
) {
  if (!group) return;
  for (const leaf of group.filters ?? []) {
    if (!leaf.param || !leaf.ref.field) continue;
    const refs = into.get(leaf.param);
    if (refs) refs.push(leaf.ref);
    else into.set(leaf.param, [leaf.ref]);
  }
  for (const nested of group.groups ?? []) collectParamRefs(nested, into);
}

function enumLabels(
  catalog: ReportCatalog,
  entityKey: string,
  ref: ReportFieldRef,
): ValueLabels | undefined {
  let entity = catalog.entities.find((candidate) => candidate.key === entityKey);
  for (const edgeName of ref.path ?? []) {
    const edge = entity?.edges.find((candidate) => candidate.name === edgeName);
    if (!edge) return undefined;
    entity = catalog.entities.find((candidate) => candidate.key === edge.target);
  }
  const field = entity?.fields.find((candidate) => candidate.key === ref.field);
  if (!field || field.enumValues.length === 0) return undefined;
  return new Map(field.enumValues.map((entry) => [entry.value, entry.label]));
}

/**
 * The catalog's names for the values of each parameter, by parameter name: the enum labels of
 * the field a condition bound to the parameter filters, found in the report's filters (nested
 * groups included) and its measures' own filters, with the field's path walked from the
 * report's entity across the catalog's edges. A parameter bound to no enum field is absent.
 */
export function parameterValueLabels(
  ir: ReportIR | null | undefined,
  catalog: ReportCatalog | null | undefined,
): Map<string, ValueLabels> {
  const out = new Map<string, ValueLabels>();
  if (!ir || !catalog) return out;
  const refs = new Map<string, ReportFieldRef[]>();
  collectParamRefs(ir.filters, refs);
  for (const column of ir.columns) {
    if (column.kind === "measure") collectParamRefs(column.filter, refs);
  }
  for (const [name, candidates] of refs) {
    for (const ref of candidates) {
      const labels = enumLabels(catalog, ir.entity, ref);
      if (labels) {
        out.set(name, labels);
        break;
      }
    }
  }
  return out;
}

/** The options of every parameter of a report that has allowed values, by parameter name. */
export function reportParameterOptions(
  ir: ReportIR | null | undefined,
  catalog: ReportCatalog | null | undefined,
): ReportParameterOptions {
  const labels = parameterValueLabels(ir, catalog);
  const out: Record<string, ReportParameterOption[]> = {};
  for (const param of ir?.parameters ?? []) {
    if ((param.allowedValues?.length ?? 0) === 0) continue;
    out[param.name] = parameterChoices(param, labels.get(param.name));
  }
  return out;
}

/** A report a dashboard tile shows, and how the tile links its parameters to the dashboard's. */
export type DashboardTileParameters = {
  ir: ReportIR | null;
  /** Report parameter name to dashboard parameter name; a name left out links to its namesake. */
  bindings?: Record<string, string>;
};

/**
 * The options of a dashboard's parameters. A dashboard parameter carries no field of its own,
 * so its values are named through the tiles: the first tile whose report links one of its
 * parameters to it (by binding, or by the same name) and resolves that parameter to an enum
 * field lends that field's labels. A parameter no tile resolves reads its codes in words.
 */
export function dashboardParameterOptions(
  parameters: readonly ReportParameterDef[],
  tiles: readonly DashboardTileParameters[],
  catalog: ReportCatalog | null | undefined,
): ReportParameterOptions {
  const resolved = tiles.map((tile) => ({
    tile,
    labels: parameterValueLabels(tile.ir, catalog),
  }));
  const out: Record<string, ReportParameterOption[]> = {};
  for (const param of parameters) {
    if ((param.allowedValues?.length ?? 0) === 0) continue;
    let labels: ValueLabels | undefined;
    for (const { tile, labels: byParam } of resolved) {
      for (const reportParam of tile.ir?.parameters ?? []) {
        const linked = tile.bindings?.[reportParam.name] || reportParam.name;
        if (linked !== param.name) continue;
        labels = byParam.get(reportParam.name);
        if (labels) break;
      }
      if (labels) break;
    }
    out[param.name] = parameterChoices(param, labels);
  }
  return out;
}
