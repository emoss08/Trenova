import { useAccountingMappingLabels } from "@/hooks/use-accounting-mapping-labels";
import type { AccountingMapping } from "@/lib/graphql/accounting-sync";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { MappingStateBadge } from "./mapping-state-badge";

export function MappingRow({
  mapping,
  selected,
  checked,
  canCheck,
  onSelect,
  onCheckedChange,
}: {
  mapping: AccountingMapping;
  selected: boolean;
  checked: boolean;
  canCheck: boolean;
  onSelect: () => void;
  onCheckedChange: (checked: boolean) => void;
}) {
  const t = useT();
  const labels = useAccountingMappingLabels();
  const checkable = canCheck && mapping.state === "Proposed";

  return (
    <div
      className={cn(
        "flex items-start gap-2.5 px-3 py-2.5",
        selected ? "bg-surface-selected" : "hover:bg-surface-hover",
      )}
    >
      <div className="flex h-5 w-4 shrink-0 items-center">
        {checkable ? (
          <Checkbox
            checked={checked}
            onCheckedChange={(value) => onCheckedChange(value === true)}
            aria-label={t("Confirm {0}", mapping.targetLabel)}
          />
        ) : null}
      </div>
      <button
        type="button"
        onClick={onSelect}
        aria-current={selected ? "true" : undefined}
        className="ui-inset-focus-ring min-w-0 flex-1 rounded-sm text-left"
      >
        <div className="flex items-center justify-between gap-2">
          <span className="truncate text-sm font-medium">{mapping.targetLabel}</span>
          <MappingStateBadge state={mapping.state} source={mapping.source} />
        </div>
        <div className="text-foreground-muted mt-0.5 flex items-center justify-between gap-2 text-xs">
          <span className="truncate">
            {mapping.externalName ? t("→ {0}", mapping.externalName) : t("No record chosen")}
          </span>
          <span className="shrink-0">
            {mapping.required ? t("Required") : labels.targetType(mapping.targetType)}
          </span>
        </div>
      </button>
    </div>
  );
}
