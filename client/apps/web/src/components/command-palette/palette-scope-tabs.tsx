import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  Grid01Icon,
  MessageChatSquareIcon,
  LayersThree01Icon,
  TerminalSquareIcon,
} from "@trenova/shared/components/icons";
import type { GlobalSearchEntityType } from "@/services/global-search";
import { Fragment } from "react";
import { PALETTE_ENTITY_ICONS, PALETTE_RECORD_TYPES, paletteEntityText } from "./palette-entities";
import type { PaletteIcon, PaletteScope } from "./palette-model";
import { isRecordScope } from "./palette-model";

/**
 * The scopes the palette offers, in order: everything, each kind of record the
 * person may read (`useReadableRecordTypes`), then the pages and the commands.
 * Before their permissions are known no kind of record is offered.
 */
export function paletteScopes(
  readable: readonly GlobalSearchEntityType[] | null,
  conversations = false,
): readonly PaletteScope[] {
  return [
    "all",
    ...PALETTE_RECORD_TYPES.filter((type) => readable?.includes(type) ?? false),
    ...(conversations ? (["conversations"] as const) : []),
    "pages",
    "commands",
  ];
}

/** The scope a step away from the current one, among those offered, wrapping round. */
export function nextScope(
  current: PaletteScope,
  step: 1 | -1,
  scopes: readonly PaletteScope[],
): PaletteScope {
  const index = Math.max(scopes.indexOf(current), 0);
  const next = (index + step + scopes.length) % scopes.length;
  return scopes[next] ?? "all";
}

export function scopePresentation(
  scope: PaletteScope,
  t: TranslateFn,
): { label: string; icon: PaletteIcon } {
  if (isRecordScope(scope)) {
    return { label: paletteEntityText(scope, t).plural, icon: PALETTE_ENTITY_ICONS[scope] };
  }
  switch (scope) {
    case "conversations":
      return { label: t("Conversations"), icon: MessageChatSquareIcon };
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
 * holds, and a tab with none fades but stays reachable. Until the person's
 * permissions are known the kinds of record they may read hold their place as
 * placeholders, so the strip does not jump when they arrive.
 */
export function PaletteScopeTabs({
  scopes,
  scope,
  onScopeChange,
  counts,
  pending = false,
}: {
  /** The scopes offered: only the kinds of record the person may read. */
  scopes: readonly PaletteScope[];
  scope: PaletteScope;
  onScopeChange: (scope: PaletteScope) => void;
  /** Results per scope for the current query; absent before anything is typed. */
  counts: Partial<Record<PaletteScope, number>> | null;
  /** The kinds of record the person may read are not known yet. */
  pending?: boolean;
}) {
  const t = useT();

  return (
    <div
      role="tablist"
      aria-label={t("Search scope")}
      className="border-border-subtle flex [scrollbar-width:none] items-center gap-0.5 overflow-x-auto border-b px-3 pb-2"
    >
      {scopes.map((option) => (
        <Fragment key={option}>
          {pending && option === "pages" && <PendingRecordScopes />}
          <ScopeTab option={option} scope={scope} counts={counts} onScopeChange={onScopeChange} />
        </Fragment>
      ))}
    </div>
  );
}

/** A placeholder for each kind of record, where the kinds the person may read will be. */
function PendingRecordScopes() {
  return PALETTE_RECORD_TYPES.map((type) => (
    <Skeleton
      key={type}
      data-testid="pending-scope"
      aria-hidden
      className="mx-2 h-2.5 w-14 shrink-0"
    />
  ));
}

function ScopeTab({
  option,
  scope,
  counts,
  onScopeChange,
}: {
  option: PaletteScope;
  scope: PaletteScope;
  counts: Partial<Record<PaletteScope, number>> | null;
  onScopeChange: (scope: PaletteScope) => void;
}) {
  const t = useT();
  const { label } = scopePresentation(option, t);
  const active = option === scope;
  const count = counts?.[option];
  const zero = counts !== null && option !== "all" && count === 0;

  return (
    <button
      type="button"
      role="tab"
      aria-selected={active}
      tabIndex={-1}
      onMouseDown={(event) => event.preventDefault()}
      onClick={() => onScopeChange(option)}
      className={cn(
        "rounded-control flex h-6.5 shrink-0 items-center gap-1.5 px-2 text-sm font-medium transition-[color,background-color,opacity]",
        active ? "bg-surface-hover text-foreground" : "text-foreground-muted hover:text-foreground",
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
}
