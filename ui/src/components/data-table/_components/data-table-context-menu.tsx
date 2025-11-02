import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuShortcut,
  ContextMenuSub,
  ContextMenuSubContent,
  ContextMenuSubTrigger,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import { ContextMenuAction } from "@/types/data-table";
import type { Row } from "@tanstack/react-table";
import React, { ReactNode } from "react";

export interface DataTableContextMenuProps<TData> {
  children: ReactNode;
  row: Row<TData>;
  actions: ContextMenuAction<TData>[];
}

export function DataTableContextMenu<TData>({
  children,
  row,
  actions,
}: DataTableContextMenuProps<TData>) {
  const renderAction = (action: ContextMenuAction<TData>) => {
    const isDisabled =
      typeof action.disabled === "function"
        ? action.disabled(row)
        : action.disabled;
    const isHidden =
      typeof action.hidden === "function" ? action.hidden(row) : action.hidden;

    if (isHidden) return null;

    if (action.subActions && action.subActions.length > 0) {
      return (
        <ContextMenuSub key={`${action.id}-${row.index}`}>
          <ContextMenuSubTrigger disabled={isDisabled}>
            {typeof action.label === "function"
              ? action.label(row)
              : action.label}
          </ContextMenuSubTrigger>
          <ContextMenuSubContent>
            {action.subActions.map((subAction) => renderAction(subAction))}
          </ContextMenuSubContent>
        </ContextMenuSub>
      );
    }

    return (
      <ContextMenuItem
        key={action.id}
        disabled={isDisabled}
        variant={action.variant}
        onClick={() => action.onClick?.(row)}
        className="flex flex-col gap-1 justify-start items-start"
      >
        {typeof action.label === "function" ? action.label(row) : action.label}
        {action.description && (
          <div className="text-xs text-muted-foreground">
            {typeof action.description === "function"
              ? action.description(row)
              : action.description}
          </div>
        )}
        {action.shortcut && (
          <ContextMenuShortcut>{action.shortcut}</ContextMenuShortcut>
        )}
      </ContextMenuItem>
    );
  };

  return (
    <ContextMenu>
      <ContextMenuTrigger asChild>{children}</ContextMenuTrigger>
      <ContextMenuContent className="w-64">
        {actions.map((action, index) => (
          <React.Fragment key={`${action.id}-${row.index}-${index}`}>
            {action.separator === "before" && index > 0 && (
              <ContextMenuSeparator />
            )}
            {renderAction(action)}
            {action.separator === "after" && index < actions.length - 1 && (
              <ContextMenuSeparator />
            )}
          </React.Fragment>
        ))}
      </ContextMenuContent>
    </ContextMenu>
  );
}
