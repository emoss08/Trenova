import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { MoreHorizontalIcon, type LucideIcon } from "lucide-react";

export type RowAction = {
  id: string;
  label: string;
  icon?: LucideIcon;
  onSelect: () => void;
  disabled?: boolean;
  destructive?: boolean;
};

type RowActionsMenuProps = {
  /** Accessible name of the trigger, e.g. "Actions for CDL". */
  label: string;
  actions: readonly RowAction[];
  disabled?: boolean;
  className?: string;
};

/**
 * One ellipsis per row instead of a strip of buttons. A row with nothing the
 * viewer may do renders no trigger at all, so a missing menu is itself the
 * answer to "can I act on this".
 */
export function RowActionsMenu({ label, actions, disabled, className }: RowActionsMenuProps) {
  const t = useT();

  if (actions.length === 0) return null;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button
            variant="ghost"
            size="icon-sm"
            className={className}
            aria-label={label}
            disabled={disabled}
          />
        }
      >
        <MoreHorizontalIcon className="size-4" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-48">
        {actions.map((action) => {
          const Icon = action.icon;
          return (
            <DropdownMenuItem
              key={action.id}
              title={t(action.label)}
              startContent={Icon ? <Icon className="size-3.5" /> : undefined}
              color={action.destructive ? "danger" : undefined}
              disabled={action.disabled}
              onClick={action.onSelect}
            />
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
