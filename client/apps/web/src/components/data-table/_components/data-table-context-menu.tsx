"use no memo";
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
import type { RowAction, Row } from "@trenova/shared/types/data-table";
import { Edit02Icon, EyeIcon } from "@trenova/shared/components/icons";
import type { ReactNode } from "react";

interface DataTableContextMenuProps<TData extends RowData> {
  children: ReactNode;
  row: Row<TData>;
  openPanelEdit: (row: Row<TData>) => void;
  hasPanel: boolean;
  canOpenPanel: boolean;
  canUpdate: boolean;
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
}: DataTableContextMenuProps<TData>) {
  const t = useT();
  const actions = useDataTableRowActions<TData>();

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

  const visibleActions = allActions.filter((action) => !action.hidden?.(row));

  if (visibleActions.length === 0) {
    return <>{children}</>;
  }

  const standardActions = visibleActions.filter((a) => a.variant !== "destructive");
  const destructiveActions = visibleActions.filter((a) => a.variant === "destructive");

  const standardGroups = groupActions(standardActions);

  return (
    <ContextMenu>
      <ContextMenuTrigger render={children as React.ReactElement} />
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
