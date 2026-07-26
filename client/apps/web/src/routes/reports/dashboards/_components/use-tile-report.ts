import {
  useCannedReports,
  useReportDefinition,
  useReportDefinitionList,
} from "@/hooks/use-reports";
import {
  parseReportIR,
  type ReportDashboardTile,
  type ReportFieldRef,
  type ReportFilterGroup,
  type ReportIR,
} from "@/types/report";
import { useMemo } from "react";
import { refKey } from "./dashboard-state";

export type TileReport = {
  ir: ReportIR | null;
  name: string;
  loading: boolean;
};

/** A field a dashboard filter could sensibly target, and why it was offered. */
export type FilterCandidate = {
  key: string;
  entity: string;
  ref: ReportFieldRef;
  /** Which reports on the dashboard this field reaches. */
  reports: string[];
  source: "filter" | "dimension";
};

function collectLeafRefs(group: ReportFilterGroup | null | undefined, into: ReportFieldRef[]) {
  if (!group) return;
  for (const leaf of group.filters ?? []) {
    // Parameter-bound conditions are already driven by a dashboard parameter.
    if (!leaf.param) into.push(leaf.ref);
  }
  for (const nested of group.groups ?? []) collectLeafRefs(nested, into);
}

/**
 * The fields worth offering as dashboard filters, taken from the reports the
 * dashboard already shows: what each report filters on, plus what it groups by
 * — a report broken down by Service Type is exactly one you would want to
 * narrow by Service Type.
 */
export function useDashboardFilterCandidates(tiles: ReportDashboardTile[]): {
  entities: string[];
  candidates: FilterCandidate[];
} {
  const definitions = useReportDefinitionList("");
  const canned = useCannedReports();

  return useMemo(() => {
    const byId = new Map(
      (definitions.data ?? []).map((entry) => [
        entry.id,
        { raw: entry.definition, name: entry.name },
      ]),
    );
    const byKey = new Map(
      (canned.data ?? []).map((entry) => [entry.key, { raw: entry.definition, name: entry.name }]),
    );

    const entities = new Set<string>();
    const candidates = new Map<string, FilterCandidate>();

    for (const tile of tiles) {
      const source = tile.cannedKey
        ? byKey.get(tile.cannedKey)
        : tile.definitionId
          ? byId.get(tile.definitionId)
          : undefined;
      const ir = source ? parseReportIR(source.raw) : null;
      if (!ir) continue;

      entities.add(ir.entity);

      const add = (ref: ReportFieldRef, kind: FilterCandidate["source"]) => {
        const key = `${ir.entity}::${refKey(ref)}`;
        const existing = candidates.get(key);
        if (existing) {
          if (source && !existing.reports.includes(source.name)) {
            existing.reports.push(source.name);
          }
          return;
        }
        candidates.set(key, {
          key,
          entity: ir.entity,
          ref,
          reports: source ? [source.name] : [],
          source: kind,
        });
      };

      const refs: ReportFieldRef[] = [];
      collectLeafRefs(ir.filters, refs);
      for (const ref of refs) add(ref, "filter");

      for (const column of ir.columns) {
        if (column.kind === "dimension" && column.ref) add(column.ref, "dimension");
      }
    }

    return { entities: [...entities], candidates: [...candidates.values()] };
  }, [tiles, definitions.data, canned.data]);
}

/**
 * Resolves the report behind a tile. Tiles never carry their own definition,
 * so a dashboard always shows the same numbers the report itself exports.
 */
export function useTileReport(tile: ReportDashboardTile | null): TileReport {
  const definitionId = tile?.definitionId;
  const cannedKey = tile?.cannedKey;

  const definition = useReportDefinition(definitionId);
  const canned = useCannedReports(Boolean(cannedKey));

  return useMemo(() => {
    if (cannedKey) {
      const entry = canned.data?.find((report) => report.key === cannedKey);
      return {
        ir: entry ? parseReportIR(entry.definition) : null,
        name: entry?.name ?? cannedKey,
        loading: canned.isLoading,
      };
    }
    if (definitionId) {
      return {
        ir: definition.data ? parseReportIR(definition.data.definition) : null,
        name: definition.data?.name ?? "Report",
        loading: definition.isLoading,
      };
    }
    return { ir: null, name: "", loading: false };
  }, [
    cannedKey,
    definitionId,
    canned.data,
    canned.isLoading,
    definition.data,
    definition.isLoading,
  ]);
}
