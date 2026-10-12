import { useT } from "@trenova/shared/i18n/use-t";
import type { RowData } from "@tanstack/react-table";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuGroup,
  ContextMenuItem,
  ContextMenuLabel,
  ContextMenuSeparator,
  ContextMenuShortcut,
  ContextMenuTrigger,
} from "@trenova/shared/components/ui/context-menu";
import { useDataTableRowActions } from "@/contexts/data-table-row-context";
import { useOptionalDataTable } from "@/contexts/data-table-context";
import { formatShortcut } from "@trenova/shared/lib/shortcuts";
import type { RowAction, Row } from "@trenova/shared/types/data-table";
import {
  ArrowNarrowDownIcon,
  ClipboardIcon,
  Columns01Icon,
  Copy01Icon,
  Edit02Icon,
  EyeIcon,
  Pin01Icon,
  PinOffIcon,
  Rows01Icon,
} from "@trenova/shared/components/icons";
import {
  buildClipboardGrid,
  clipboardColumns,
  writeClipboardGrid,
  type ClipboardGrid,
} from "@/lib/data-table-clipboard";
import { toast } from "sonner";
import { useState } from "react";
import { useTableAtom } from "@trenova/shared/hooks/use-table-atom";

/** The most rows a person can keep pinned at the top of one table. */
export const MAX_PINNED_ROWS = 25;
import type { ReactNode } from "react";

interface DataTableContextMenuProps<TData extends RowData> {
  children: ReactNode;
  row: Row<TData>;
  openPanelEdit: (row: Row<TData>) => void;
  hasPanel: boolean;
  canOpenPanel: boolean;
  canUpdate: boolean;
  canPin?: boolean;
}

type ActionGroup<TData extends RowData> = {
  id: string;
  label?: string;
  actions: RowAction<TData>[];
};

function groupActions<TData extends RowData>(actions: RowAction<TData>[]): ActionGroup<TData>[] {
  const groups: ActionGroup<TData>[] = [];
  const groupMap = new Map<string, ActionGroup<TData>>();

  for (const action of actions) {
    const groupDef = action.group;
    const key =
      groupDef == null ? "__default__" : typeof groupDef === "string" ? groupDef : groupDef.id;
    const label = groupDef != null && typeof groupDef === "object" ? groupDef.label : undefined;

    let group = groupMap.get(key);
    if (!group) {
      group = { id: key, label, actions: [] };
      groupMap.set(key, group);
      groups.push(group);
    }
    group.actions.push(action);
  }

  return groups;
}

export function DataTableContextMenu<TData extends RowData>({
  children,
  row,
  openPanelEdit,
  hasPanel,
  canOpenPanel,
  canUpdate,
  canPin = false,
}: DataTableContextMenuProps<TData>) {
  const t = useT();
  const actions = useDataTableRowActions<TData>();
  const dataTable = useOptionalDataTable<TData, unknown>();
  // The row object outlives a pin, so whether it is pinned is read from the state.
  const pinned = useTableAtom(row.table.atoms.rowPinning, (pinning) =>
    pinning.top.includes(row.id),
  );
  const pinnedCount = useTableAtom(
    row.table.atoms.rowPinning,
    (pinning) => pinning.top.length,
  );
  // The cell the menu was opened on, so "Copy cell" and "Copy column" know which.
  const [columnId, setColumnId] = useState<string | null>(null);

  const copy = (grid: ClipboardGrid) => {
    writeClipboardGrid(grid).then(
      () => toast.success(t("Copied")),
      (error: unknown) =>
        toast.error(t("Nothing was copied"), {
          description: error instanceof Error ? error.message : undefined,
        }),
    );
  };

  const allActions: RowAction<TData>[] = [];

  if (hasPanel && canOpenPanel) {
    allActions.push({
      id: "edit",
      label: canUpdate ? "Edit" : "View",
      icon: canUpdate ? Edit02Icon : EyeIcon,
      onClick: openPanelEdit,
    });
  }

  allActions.push(...actions);

  const textColumns = clipboardColumns(row.table.getAllLeafColumns());
  const visibleColumnIds = row.table.getVisibleLeafColumns().map((column) => column.id);
  const copyableColumn = columnId && textColumns.has(columnId) ? columnId : null;
  if (copyableColumn) {
    allActions.push({
      id: "copy-cell",
      label: "Copy cell",
      icon: Copy01Icon,
      group: "__copy__",
      onClick: (target) =>
        copy(buildClipboardGrid([target.original], [copyableColumn], textColumns)),
    });
  }
  allActions.push({
    id: "copy-row",
    label: "Copy row",
    icon: Rows01Icon,
    group: "__copy__",
    onClick: (target) =>
      copy(
        buildClipboardGrid([target.original], visibleColumnIds, textColumns, {
          includeHeader: true,
        }),
      ),
  });
  if (copyableColumn) {
    allActions.push({
      id: "copy-column",
      label: "Copy column",
      icon: Columns01Icon,
      group: "__copy__",
      onClick: (target) => {
        const rows = [...target.table.getTopRows(), ...target.table.getCenterRows()];
        copy(
          buildClipboardGrid(
            rows.map((entry) => entry.original),
            [copyableColumn],
            textColumns,
            { includeHeader: true },
          ),
        );
      },
    });
  }

  const editableCell =
    canUpdate && columnId
      ? row.getVisibleCells().find((cell) => cell.column.id === columnId && cell.getCanEdit())
      : undefined;
  if (dataTable && editableCell) {
    allActions.push({
      id: "paste",
      label: "Paste",
      icon: ClipboardIcon,
      shortcut: formatShortcut("V"),
      group: "__edit_cells__",
      onClick: (target) => {
        if (!editableCell.getIsSelected()) target.table.setFocusedCell(target.id, editableCell.column.id);
        dataTable.pasteFromClipboard();
      },
    });
    allActions.push({
      id: "fill-down",
      label: "Fill down",
      icon: ArrowNarrowDownIcon,
      shortcut: formatShortcut("D"),
      group: "__edit_cells__",
      disabled: () => row.table.getSelectedCellCount() < 2,
      onClick: () => dataTable.fillDown(),
    });
  }

  if (canPin) {
    allActions.push({
      id: "pin",
      label: pinned ? "Unpin" : "Pin to top",
      icon: pinned ? PinOffIcon : Pin01Icon,
      shortcut: "P",
      group: "__pin__",
      disabled: () => !pinned && pinnedCount >= MAX_PINNED_ROWS,
      onClick: (target) => target.pin(pinned ? false : "top"),
    });
  }

  const visibleActions = allActions.filter((action) => !action.hidden?.(row));

  if (visibleActions.length === 0) {
    return <>{children}</>;
  }

  const standardActions = visibleActions.filter((a) => a.variant !== "destructive");
  const destructiveActions = visibleActions.filter((a) => a.variant === "destructive");

  const standardGroups = groupActions(standardActions);

  return (
    <ContextMenu>
      <ContextMenuTrigger
        render={children as React.ReactElement}
        onContextMenu={(event) => {
          const cell = (event.target as HTMLElement).closest<HTMLElement>("td[data-column-id]");
          setColumnId(cell?.dataset.columnId ?? null);
        }}
      />
      <ContextMenuContent className="w-auto min-w-[160px]">
        {standardGroups.map((group, groupIndex) => (
          <ContextMenuGroup key={group.id}>
            {groupIndex > 0 && <ContextMenuSeparator />}
            {group.label && <ContextMenuLabel>{t(group.label)}</ContextMenuLabel>}
            {group.actions.map((action) => {
              const Icon = action.icon;
              return (
                <ContextMenuItem
                  key={action.id}
                  disabled={action.disabled?.(row)}
                  onClick={() => void action.onClick(row)}
                >
                  {Icon && <Icon className="size-4" />}
                  {t(action.label)}
                  {action.shortcut ? (
                    <ContextMenuShortcut>{action.shortcut}</ContextMenuShortcut>
                  ) : null}
                </ContextMenuItem>
              );
            })}
          </ContextMenuGroup>
        ))}
        {destructiveActions.length > 0 && (
          <>
            {standardActions.length > 0 && <ContextMenuSeparator />}
            <ContextMenuGroup>
              {destructiveActions.map((action) => {
                const Icon = action.icon;
                return (
                  <ContextMenuItem
                    key={action.id}
                    variant="destructive"
                    disabled={action.disabled?.(row)}
                    onClick={() => void action.onClick(row)}
                  >
                    {Icon && <Icon className="size-4" />}
                    {t(action.label)}
                  </ContextMenuItem>
                );
              })}
            </ContextMenuGroup>
          </>
        )}
      </ContextMenuContent>
    </ContextMenu>
  );
}
