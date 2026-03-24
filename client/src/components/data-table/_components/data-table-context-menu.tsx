"use no memo";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuGroup,
  ContextMenuItem,
  ContextMenuLabel,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import { useDataTable } from "@/contexts/data-table-context";
import type { RowAction } from "@/types/data-table";
import type { Row } from "@tanstack/react-table";
import { PencilIcon } from "lucide-react";
import type { ReactNode } from "react";

interface DataTableContextMenuProps<TData> {
  children: ReactNode;
  row: Row<TData>;
  actions?: RowAction<TData>[];
}

type ActionGroup<TData> = {
  id: string;
  label?: string;
  actions: RowAction<TData>[];
};

function groupActions<TData>(actions: RowAction<TData>[]): ActionGroup<TData>[] {
  const groups: ActionGroup<TData>[] = [];
  const groupMap = new Map<string, ActionGroup<TData>>();

  for (const action of actions) {
    const groupDef = action.group;
    const key = groupDef == null ? "__default__" : typeof groupDef === "string" ? groupDef : groupDef.id;
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

export function DataTableContextMenu<TData>({
  children,
  row,
  actions = [],
}: DataTableContextMenuProps<TData>) {
  const { openPanelEdit, hasPanel, canUpdate } = useDataTable<TData, unknown>();

  const allActions: RowAction<TData>[] = [];

  if (hasPanel && canUpdate) {
    allActions.push({
      id: "edit",
      label: "Edit",
      icon: PencilIcon,
      onClick: openPanelEdit,
    });
  }

  allActions.push(...actions);

  const visibleActions = allActions.filter(
    (action) => !action.hidden?.(row),
  );

  if (visibleActions.length === 0) {
    return <>{children}</>;
  }

  const standardActions = visibleActions.filter(
    (a) => a.variant !== "destructive",
  );
  const destructiveActions = visibleActions.filter(
    (a) => a.variant === "destructive",
  );

  const standardGroups = groupActions(standardActions);

  return (
    <ContextMenu>
      <ContextMenuTrigger render={children as React.ReactElement} />
      <ContextMenuContent className="w-auto min-w-[160px]">
        {standardGroups.map((group, groupIndex) => (
          <ContextMenuGroup key={group.id}>
            {groupIndex > 0 && <ContextMenuSeparator />}
            {group.label && <ContextMenuLabel>{group.label}</ContextMenuLabel>}
            {group.actions.map((action) => {
              const Icon = action.icon;
              return (
                <ContextMenuItem
                  key={action.id}
                  disabled={action.disabled?.(row)}
                  onClick={() => action.onClick(row)}
                >
                  {Icon && <Icon className="size-4" />}
                  {action.label}
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
                    onClick={() => action.onClick(row)}
                  >
                    {Icon && <Icon className="size-4" />}
                    {action.label}
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
