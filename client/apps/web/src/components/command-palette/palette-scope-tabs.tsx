import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { LayersIcon, LayoutGridIcon, TerminalSquareIcon } from "lucide-react";
import { PALETTE_ENTITIES } from "./palette-entities";
import type { PaletteIcon, PaletteScope } from "./palette-model";

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

function scopePresentation(scope: PaletteScope): { label: string; icon: PaletteIcon } {
  switch (scope) {
    case "all":
      return { label: "All", icon: LayersIcon };
    case "pages":
      return { label: "Pages", icon: LayoutGridIcon };
    case "commands":
      return { label: "Commands", icon: TerminalSquareIcon };
    default:
      return { label: PALETTE_ENTITIES[scope].pluralLabel, icon: PALETTE_ENTITIES[scope].icon };
  }
}

/**
 * The strip under the search box. Focus never leaves the box for it: Tab and
 * Shift+Tab move between scopes from the keyboard, and a click does the same
 * without taking the caret away.
 */
export function PaletteScopeTabs({
  scope,
  onScopeChange,
  counts,
}: {
  scope: PaletteScope;
  onScopeChange: (scope: PaletteScope) => void;
  /** Hits per record scope for the current query, once known. */
  counts: Partial<Record<PaletteScope, number>>;
}) {
  const t = useT();

  return (
    <div
      role="tablist"
      aria-label={t("Search scope")}
      className="border-border-subtle flex [scrollbar-width:none] items-center gap-0.5 overflow-x-auto border-b px-3"
    >
      {PALETTE_SCOPES.map((option) => {
        const { label, icon: Icon } = scopePresentation(option);
        const active = option === scope;
        const count = counts[option];
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
              "relative flex h-9 shrink-0 items-center gap-1.5 px-2.5 text-xs font-medium transition-colors",
              "after:bg-brand after:absolute after:inset-x-2 after:bottom-0 after:h-0.5 after:rounded-full after:opacity-0 after:transition-opacity",
              active
                ? "text-foreground after:opacity-100"
                : "text-foreground-subtle hover:text-foreground",
            )}
          >
            <Icon className="size-3.5" strokeWidth={1.75} />
            {t(label)}
            {count !== undefined && count > 0 && (
              <span className="bg-sunken text-2xs text-foreground-muted rounded-full px-1.5 tabular-nums">
                {count}
              </span>
            )}
          </button>
        );
      })}
    </div>
  );
}
