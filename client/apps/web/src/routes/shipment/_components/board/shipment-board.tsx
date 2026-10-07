import { DataTable } from "@/components/data-table/data-table";
import { useUserTimezone } from "@/hooks/use-user-timezone";
import { shipmentBoardTableGraphQLConfig } from "@/lib/graphql/shipment";
import { queries } from "@/lib/queries";
import { useShipmentCapabilities } from "@/lib/shipment-board/capabilities";
import { STAGE_GROUP_FIELD, stageGroups, stageRankLookup } from "@/lib/shipment-board/stage";
import { Keyboard01Icon, Truck01Icon, User01Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatShortcut } from "@trenova/shared/lib/shortcuts";
import type {
  DataTableAlternateView,
  DataTableExpansion,
  DataTableGroupKey,
  DataTableGrouping,
  DataTableKeyboard,
  DataTableToolbarSlots,
  DockAction,
} from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import type { Shipment } from "@trenova/shared/types/shipment";
import { useAppDialogsStore } from "@/stores/app-dialogs-store";
import { useQuery } from "@tanstack/react-query";
import { toast } from "sonner";
import { lazy, Suspense, useCallback, useMemo, useState } from "react";
import { ShipmentExpandedRow } from "./expanded-row/shipment-expanded-row";
import { useShipmentRecordActions } from "./record-actions";
import { getColumns, SHIPMENT_HIDDEN_COLUMNS } from "./shipment-columns";
import { GroupToggle, PanelToggle, TOOLBAR_RESPONSIVE, ViewSwitch } from "./toolbar/board-controls";
import { useBoardActions } from "./use-board-actions";
import { useBoardScope } from "./use-board-scope";
import { useQuickFilterSearch } from "./use-quick-filter-search";
import { useShipmentBoardUrl } from "./url-state";

const ShipmentTimeline = lazy(() => import("./views/shipment-timeline"));
const ShipmentMapView = lazy(() => import("./views/shipment-map-view"));

const PAGE_SIZE_OPTIONS = [25, 50, 100] as const;
const PINNED_COLUMNS = { left: ["select", "lane"], right: [] };

type ShipmentBoardProps = {
  panelOpen: boolean;
  onPanelOpenChange: (open: boolean) => void;
};

export function ShipmentBoard({ panelOpen, onPanelOpenChange }: ShipmentBoardProps) {
  const t = useT();
  const capabilities = useShipmentCapabilities();
  const [{ view, group, collapsed, expanded, qf }, setUrl] = useShipmentBoardUrl();
  const timezone = useUserTimezone();
  const scope = useBoardScope();
  const { rowActions, edit, copyLink, copyProNumber } = useShipmentRecordActions();
  const actions = useBoardActions();
  const openShortcuts = useAppDialogsStore((state) => state.openDialog);
  const [cursorRowId, setCursorRowId] = useState<string | null>(null);
  const { searchSuggestions, chips } = useQuickFilterSearch();

  const { data: summary = [] } = useQuery({
    ...queries.shipmentBoard.stageSummary(scope),
    enabled: group,
    staleTime: 15_000,
  });

  const graphql = useMemo(
    () => shipmentBoardTableGraphQLConfig({ quickFilters: qf, timezone }),
    [qf, timezone],
  );

  const toggleExpanded = useCallback(
    (rowId: string) =>
      void setUrl((current) => ({ expanded: current.expanded === rowId ? null : rowId })),
    [setUrl],
  );

  const columns = useMemo(
    () => getColumns({ t, onToggleExpanded: toggleExpanded }),
    [t, toggleExpanded],
  );

  const grouping = useMemo<DataTableGrouping<Shipment> | undefined>(() => {
    if (!group) return undefined;
    const rankOf = stageRankLookup(summary);
    return {
      field: STAGE_GROUP_FIELD,
      groups: stageGroups(summary, t),
      getGroupKey: (row) => rankOf(row.stage),
      collapsedKeys: collapsed,
      onToggleGroup: (key: DataTableGroupKey) => {
        const rank = Number(key);
        void setUrl({
          collapsed: collapsed.includes(rank)
            ? collapsed.filter((entry) => entry !== rank)
            : [...collapsed, rank],
        });
      },
    };
  }, [group, summary, t, collapsed, setUrl]);

  const tenderShipments = actions.tender.mutateAsync;
  const autoAssignMoves = actions.autoAssign.mutateAsync;
  const dockActions = useMemo<DockAction<Shipment>[]>(() => {
    const ids = (rows: Shipment[]) => rows.map((row) => row.id).filter((id): id is string => !!id);
    const list: DockAction<Shipment>[] = [];
    if (capabilities.runsBrokerage) {
      list.push({
        id: "tender",
        label: capabilities.runsAssets ? t("Tender") : t("Tender to carrier"),
        loadingLabel: t("Tendering"),
        icon: Truck01Icon,
        clearSelectionOnSuccess: true,
        onClick: async (rows) => {
          const result = await tenderShipments(ids(rows).map((shipmentId) => ({ shipmentId })));
          if (result.tendered.length > 0) {
            toast.success(t("Tendered {0} to best-match carriers", result.tendered.length));
          }
        },
      });
    }
    if (capabilities.runsAssets) {
      list.push({
        id: "assign",
        label: t("Assign"),
        loadingLabel: t("Assigning"),
        icon: User01Icon,
        clearSelectionOnSuccess: true,
        onClick: async (rows) => {
          await autoAssignMoves(
            rows.flatMap(
              (row) => row.moves?.map((move) => move.id).filter(Boolean) ?? [],
            ) as string[],
          );
        },
      });
    }
    return list;
  }, [autoAssignMoves, capabilities, t, tenderShipments]);

  const rowShortcuts = useMemo(
    () => [
      { key: "e", run: edit },
      { key: "c", alt: true, run: copyProNumber },
      { key: "l", mod: true, run: copyLink },
    ],
    [edit, copyProNumber, copyLink],
  );

  const expansion = useMemo<DataTableExpansion<Shipment>>(
    () => ({
      expandedRowId: expanded,
      onExpandedRowIdChange: (rowId) => void setUrl({ expanded: rowId }),
      renderExpandedRow: (row, { collapse }) => (
        <ShipmentExpandedRow row={row} onCollapse={collapse} />
      ),
    }),
    [expanded, setUrl],
  );

  const keyboard = useMemo<DataTableKeyboard<Shipment>>(
    () => ({
      enabled: view === "table",
      cursorRowId,
      onCursorRowIdChange: setCursorRowId,
      rowShortcuts,
    }),
    [view, cursorRowId, rowShortcuts],
  );

  const toolbar = useMemo<DataTableToolbarSlots>(
    () => ({
      searchSuggestions,
      searchShortcut: "/",
      chips,
      trailing: (
        <>
          <ViewSwitch />
          <GroupToggle />
        </>
      ),
      end: <PanelToggle open={panelOpen} onOpenChange={onPanelOpenChange} />,
      responsive: TOOLBAR_RESPONSIVE,
    }),
    [searchSuggestions, chips, panelOpen, onPanelOpenChange],
  );

  const alternateView = useMemo<DataTableAlternateView>(
    () => ({
      active: view !== "table",
      render: ({ queryOptions }) => (
        <Suspense fallback={null}>
          {view === "timeline" ? (
            <ShipmentTimeline graphql={graphql} queryOptions={queryOptions} />
          ) : (
            <ShipmentMapView />
          )}
        </Suspense>
      ),
    }),
    [view, graphql],
  );

  const footerLeading = useMemo(
    () => (
      <Button
        variant="ghost"
        size="icon-sm"
        aria-label={t("Keyboard shortcuts ({0})", formatShortcut("/"))}
        onClick={() => openShortcuts("shortcuts")}
      >
        <Keyboard01Icon className="size-4" />
      </Button>
    ),
    [t, openShortcuts],
  );

  return (
    <DataTable<Shipment>
      name="Shipment"
      resource={Resource.Shipment}
      queryKey="shipment-list"
      graphql={graphql}
      columns={columns}
      initialColumnVisibility={SHIPMENT_HIDDEN_COLUMNS}
      initialColumnPinning={PINNED_COLUMNS}
      initialDensity="compact"
      pageSizeOptions={PAGE_SIZE_OPTIONS}
      enableCreateAction={false}
      enableRowSelection
      dockActions={dockActions}
      contextMenuActions={rowActions}
      grouping={grouping}
      expansion={expansion}
      keyboard={keyboard}
      toolbar={toolbar}
      alternateView={alternateView}
      footerLeading={footerLeading}
    />
  );
}
