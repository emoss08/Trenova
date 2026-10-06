import { useT } from "@trenova/shared/i18n/use-t";
import { MessageSquare01Icon } from "@trenova/shared/components/icons";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { cn } from "@trenova/shared/lib/utils";
import type { Row, RowAction } from "@trenova/shared/types/data-table";
import type { Shipment } from "@trenova/shared/types/shipment";

const QUICK_ACTION_IDS = ["edit", "duplicate", "transfer-ownership", "copy-link"] as const;

function QuickActionButton({
  label,
  icon: Icon,
  shortcut,
  destructive,
  disabled,
  busy,
  onClick,
}: {
  label: string;
  icon?: RowAction<Shipment>["icon"];
  shortcut?: string;
  destructive?: boolean;
  disabled?: boolean;
  busy?: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      disabled={disabled || busy}
      aria-busy={busy || undefined}
      onClick={onClick}
      className={cn(
        "ui-focus-ring hover:bg-surface-hover flex h-7.5 min-w-0 items-center gap-2 rounded-md px-2 text-left text-sm disabled:opacity-50",
        destructive && "text-danger",
      )}
    >
      {Icon ? <Icon className="size-3.5 shrink-0" aria-hidden /> : null}
      <span className="min-w-0 flex-1 truncate">{label}</span>
      {shortcut ? <Kbd>{shortcut}</Kbd> : null}
    </button>
  );
}

/**
 * The shipment's everyday actions, laid out for the open row. They are the
 * row menu's own actions, so a hidden or disabled action stays so here.
 */
type QuickActionsProps = {
  row: Row<Shipment>;
  actions: RowAction<Shipment>[];
  onAddComment: (shipment: Shipment) => void;
};

export function QuickActions({ row, actions, onAddComment }: QuickActionsProps) {
  const t = useT();
  const rowActions = actions;
  const byId = new Map(rowActions.map((action) => [action.id, action]));
  const visible = (action: RowAction<Shipment> | undefined): action is RowAction<Shipment> =>
    !!action && !action.hidden?.(row);
  const standard = QUICK_ACTION_IDS.map((id) => byId.get(id)).filter(visible);
  const cancel = [byId.get("cancel"), byId.get("uncancel")].filter(visible);

  return (
    <div className="flex flex-col gap-1">
      <span className="text-muted-foreground text-xs font-medium">{t("Quick actions")}</span>
      <div className="grid grid-cols-2 gap-x-1">
        {standard.map((action) => (
          <QuickActionButton
            key={action.id}
            label={t(action.label)}
            icon={action.icon}
            shortcut={action.shortcut}
            disabled={action.disabled?.(row)}
            busy={action.isPending?.(row)}
            onClick={() => void action.onClick(row)}
          />
        ))}
        <QuickActionButton
          label={t("Add comment")}
          icon={MessageSquare01Icon}
          onClick={() => onAddComment(row.original)}
        />
        {cancel.map((action) => (
          <QuickActionButton
            key={action.id}
            label={t(action.label)}
            icon={action.icon}
            destructive={action.variant === "destructive"}
            disabled={action.disabled?.(row)}
            busy={action.isPending?.(row)}
            onClick={() => void action.onClick(row)}
          />
        ))}
      </div>
    </div>
  );
}
