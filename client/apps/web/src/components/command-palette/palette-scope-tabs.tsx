import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  Grid01Icon,
  LayersThree01Icon,
  TerminalSquareIcon,
} from "@trenova/shared/components/icons";
import { PALETTE_ENTITY_ICONS, paletteEntityText } from "./palette-entities";
import type { PaletteIcon, PaletteScope } from "./palette-model";
import { isRecordScope } from "./palette-model";

export const PALETTE_SCOPES: readonly PaletteScope[] = [
  "all",
  "shipment",
  "customer",
  "worker",
  "document",
  "pages",
  "commands",
];

export function nextScope(current: PaletteScope, step: 1 | -1): PaletteScope {
  const index = PALETTE_SCOPES.indexOf(current);
  const next = (index + step + PALETTE_SCOPES.length) % PALETTE_SCOPES.length;
  return PALETTE_SCOPES[next] ?? "all";
}

export function scopePresentation(
  scope: PaletteScope,
  t: TranslateFn,
): { label: string; icon: PaletteIcon } {
  if (isRecordScope(scope)) {
    return { label: paletteEntityText(scope, t).plural, icon: PALETTE_ENTITY_ICONS[scope] };
  }
  switch (scope) {
    case "pages":
      return { label: t("Pages"), icon: Grid01Icon };
    case "commands":
      return { label: t("Commands"), icon: TerminalSquareIcon };
    default:
      return { label: t("All"), icon: LayersThree01Icon };
  }
}

/**
 * The strip under the search box. Focus never leaves the box for it: Tab and
 * Shift+Tab move between scopes from the keyboard, and a click does the same
 * without taking the caret away. With a query, each tab says how many it
 * holds, and a tab with none fades but stays reachable.
 */
export function PaletteScopeTabs({
  scope,
  onScopeChange,
  counts,
}: {
  scope: PaletteScope;
  onScopeChange: (scope: PaletteScope) => void;
  /** Results per scope for the current query; absent before anything is typed. */
  counts: Partial<Record<PaletteScope, number>> | null;
}) {
  const t = useT();

  return (
    <div
      role="tablist"
      aria-label={t("Search scope")}
      className="border-border-subtle flex [scrollbar-width:none] items-center gap-0.5 overflow-x-auto border-b px-3 pb-2"
    >
      {PALETTE_SCOPES.map((option) => {
        const { label } = scopePresentation(option, t);
        const active = option === scope;
        const count = counts?.[option];
        const zero = counts !== null && option !== "all" && count === 0;
        return (
          <button
            key={option}
            type="button"
            role="tab"
            aria-selected={active}
            tabIndex={-1}
            onMouseDown={(event) => event.preventDefault()}
            onClick={() => onScopeChange(option)}
            className={cn(
              "rounded-control flex h-6.5 shrink-0 items-center gap-1.5 px-2 text-sm font-medium transition-[color,background-color,opacity]",
              active
                ? "bg-surface-hover text-foreground"
                : "text-foreground-muted hover:text-foreground",
              zero && "opacity-50",
            )}
          >
            {label}
            {counts !== null && count !== undefined && (
              <span className="text-foreground-faint font-mono text-2xs font-normal tabular-nums">
                {count.toLocaleString()}
              </span>
            )}
          </button>
        );
      })}
    </div>
  );
}
