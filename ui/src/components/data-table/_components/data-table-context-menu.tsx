/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

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
import type { Row } from "@tanstack/react-table";
import { ReactNode } from "react";

export interface ContextMenuAction<TData> {
  id: string;
  label: string | ((row: Row<TData>) => string);
  shortcut?: string;
  variant?: "default" | "destructive";
  disabled?: boolean | ((row: Row<TData>) => boolean);
  hidden?: boolean | ((row: Row<TData>) => boolean);
  onClick?: (row: Row<TData>) => void;
  separator?: "before" | "after";
  subActions?: ContextMenuAction<TData>[];
}

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
        <ContextMenuSub key={action.id}>
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
      >
        {typeof action.label === "function" ? action.label(row) : action.label}
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
          <>
            {action.separator === "before" && index > 0 && (
              <ContextMenuSeparator />
            )}
            {renderAction(action)}
            {action.separator === "after" && index < actions.length - 1 && (
              <ContextMenuSeparator />
            )}
          </>
        ))}
      </ContextMenuContent>
    </ContextMenu>
  );
}
