import { useUserTimezone } from "@/hooks/use-user-timezone";
import { queries } from "@/lib/queries";
import {
  boardGroupHeaders,
  boardGroupKey,
  collapsedScopeFor,
  groupSort,
  isServerGrouping,
  parseCollapsedKeys,
  SERVER_GROUPING,
} from "@/lib/shipment-board/grouping";
import { stageGroups, stageRankLookup } from "@/lib/shipment-board/stage";
import { useT } from "@trenova/shared/i18n/use-t";
import type { DataTableGroupKey, DataTableGrouping } from "@trenova/shared/types/data-table";
import type { Shipment } from "@trenova/shared/types/shipment";
import { useQuery } from "@tanstack/react-query";
import { useCallback, useMemo } from "react";
import { useBoardScope } from "./use-board-scope";
import { useShipmentBoardUrl } from "./url-state";

const SUMMARY_STALE_MS = 15_000;

/**
 * The board's grouping as the table takes it: the field the server sorts by,
 * how it drops collapsed groups, and headers with whole-group totals. Stage
 * keeps its own summary; every other grouping is counted by the server in
 * the same order it sorts the rows.
 */
export function useBoardGrouping(): DataTableGrouping<Shipment> | undefined {
  const t = useT();
  const [{ group, collapsed }, setUrl] = useShipmentBoardUrl();
  const scope = useBoardScope();
  const timezone = useUserTimezone();
  const serverGrouping = isServerGrouping(group) ? group : null;

  const { data: summary = [] } = useQuery({
    ...queries.shipmentBoard.stageSummary(scope),
    enabled: group === "stage",
    staleTime: SUMMARY_STALE_MS,
  });
  const { data: serverGroups = [] } = useQuery({
    ...queries.shipmentBoard.groups(scope, SERVER_GROUPING[serverGrouping ?? "customer"]),
    enabled: serverGrouping !== null,
    staleTime: SUMMARY_STALE_MS,
  });

  const collapsedKeys = useMemo(() => parseCollapsedKeys(group, collapsed), [group, collapsed]);
  const collapsedScope = useMemo(
    () => (group === "none" ? undefined : collapsedScopeFor(group, timezone)),
    [group, timezone],
  );
  const onToggleGroup = useCallback(
    (key: DataTableGroupKey) => {
      const value = String(key);
      void setUrl((current) => ({
        collapsed: current.collapsed.includes(value)
          ? current.collapsed.filter((entry) => entry !== value)
          : [...current.collapsed, value],
      }));
    },
    [setUrl],
  );

  return useMemo(() => {
    if (group === "none") return undefined;
    const { field, tieBreakers } = groupSort(group);
    const rankOf = stageRankLookup(summary);
    return {
      field,
      tieBreakers,
      collapsedScope,
      groups: serverGrouping
        ? boardGroupHeaders({ grouping: serverGrouping, groups: serverGroups, t, timezone })
        : stageGroups(summary, t),
      getGroupKey: (row: Shipment) => boardGroupKey(group, row, timezone, rankOf),
      collapsedKeys,
      onToggleGroup,
    };
  }, [
    group,
    serverGrouping,
    summary,
    serverGroups,
    t,
    timezone,
    collapsedKeys,
    collapsedScope,
    onToggleGroup,
  ]);
}
