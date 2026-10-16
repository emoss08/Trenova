import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { Button } from "@trenova/shared/components/ui/button";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  ArrowUpRightIcon,
  ChevronRightIcon,
  DotsHorizontalIcon,
  Link01Icon,
} from "@trenova/shared/components/icons";
import { ACTION_ASK, ACTION_COPY_LINK, ACTION_NEW_TAB, findAction } from "../../palette-actions";
import type { PaletteAction, PaletteIcon, PaletteIntent } from "../../palette-model";
import { PaletteTile, ShortcutKeys, StatusDot, type StatusState } from "../../palette-parts";

/** Keeps focus in the search box: nothing in the preview takes it on a click. */
function keepFocus(event: React.MouseEvent) {
  event.preventDefault();
}

/**
 * The frame every preview sits in, in the Desk artifact's vocabulary: the
 * body scrolls and re-enters when the selection changes, and the action bar
 * stays pinned under it with the first action, Ask, and the link, tab and
 * all-actions buttons.
 */
export function PreviewFrame({
  children,
  actions,
  onRun,
  onMore,
}: {
  children: React.ReactNode;
  actions: readonly PaletteAction[];
  onRun: (intent: PaletteIntent) => void;
  onMore: () => void;
}) {
  const t = useT();
  const primary = actions[0];
  const ask = findAction(actions, ACTION_ASK);
  const link = findAction(actions, ACTION_COPY_LINK);
  const tab = findAction(actions, ACTION_NEW_TAB);

  return (
    <div className="flex h-full min-h-0 flex-col">
      <ScrollArea className="min-h-0 flex-1" maskHeight={0}>
        <div className="animate-rise flex flex-col gap-4.5 px-5 py-4">{children}</div>
      </ScrollArea>
      {primary && (
        <footer className="border-border-subtle flex shrink-0 items-center gap-1.5 border-t px-3 py-2.5">
          <Button
            type="button"
            size="sm"
            tabIndex={-1}
            onMouseDown={keepFocus}
            onClick={() => onRun(primary.intent)}
            className={cn("max-w-56", primary.destructive && "bg-danger hover:bg-danger-hover")}
          >
            <span className="truncate">{primary.label}</span>
            <ShortcutKeys keys={["↵"]} onSolid />
          </Button>
          {ask && (
            <Button
              type="button"
              size="sm"
              variant="outline"
              tabIndex={-1}
              onMouseDown={keepFocus}
              onClick={() => onRun(ask.intent)}
            >
              <NovaBadge size="sm" />
              {t("Ask")}
            </Button>
          )}
          <span className="flex-1" />
          {link && (
            <Button
              type="button"
              size="icon-sm"
              variant="ghost"
              tabIndex={-1}
              aria-label={t("Copy link")}
              title={`${t("Copy link")}  ${link.shortcut?.join(" ") ?? ""}`}
              onMouseDown={keepFocus}
              onClick={() => onRun(link.intent)}
              className="text-foreground-muted size-7.5"
            >
              <Link01Icon className="size-3.5" />
            </Button>
          )}
          {tab && (
            <Button
              type="button"
              size="icon-sm"
              variant="ghost"
              tabIndex={-1}
              aria-label={t("Open in new tab")}
              title={`${t("Open in new tab")}  ${tab.shortcut?.join(" ") ?? ""}`}
              onMouseDown={keepFocus}
              onClick={() => onRun(tab.intent)}
              className="text-foreground-muted size-7.5"
            >
              <ArrowUpRightIcon className="size-3.5" />
            </Button>
          )}
          {actions.length > 1 && (
            <Button
              type="button"
              size="icon-sm"
              variant="ghost"
              tabIndex={-1}
              aria-label={t("All actions")}
              title={t("All actions")}
              onMouseDown={keepFocus}
              onClick={onMore}
              className="text-foreground-muted size-7.5"
            >
              <DotsHorizontalIcon className="size-3.5" />
            </Button>
          )}
        </footer>
      )}
    </div>
  );
}

/** "shipment · Live · Updated 13:05": what the preview is of, and how fresh it is. */
export function PreviewSource({
  kind,
  facts = [],
}: {
  kind: string;
  facts?: (string | null | undefined | false)[];
}) {
  const present = facts.filter((fact): fact is string => Boolean(fact));
  return (
    <div className="text-foreground-subtle flex min-w-0 items-center gap-1.5 text-sm">
      <code className="bg-sunken text-foreground ring-border-subtle rounded-[4px] px-1.5 py-px font-mono text-xs ring-1 ring-inset">
        {kind}
      </code>
      {present.map((fact, index) => (
        <span key={`${fact}-${index}`} className="flex min-w-0 items-center gap-1.5">
          <span aria-hidden className="text-foreground-faint">
            ·
          </span>
          <span className="truncate">{fact}</span>
        </span>
      ))}
    </div>
  );
}

export function StatePill({ label, state = "neutral" }: { label: string; state?: StatusState }) {
  return (
    <span className="bg-sunken text-foreground inline-flex h-6 shrink-0 items-center gap-1.5 rounded-badge px-2.5 text-sm font-medium">
      <StatusDot state={state} className="size-1.5" />
      {label}
    </span>
  );
}

export function PreviewHead({
  title,
  subtitle,
  mono = false,
  status,
}: {
  title: string;
  subtitle?: React.ReactNode;
  mono?: boolean;
  status?: React.ReactNode;
}) {
  return (
    <div className="flex items-start justify-between gap-3">
      <div className="flex min-w-0 flex-col gap-1">
        <h3
          className={cn(
            "text-foreground text-2xl leading-tight font-medium break-words",
            mono && "font-mono",
          )}
        >
          {title}
        </h3>
        {subtitle && <p className="text-foreground-muted truncate text-base">{subtitle}</p>}
      </div>
      {status}
    </div>
  );
}

export interface PreviewFieldRow {
  label: string;
  value: React.ReactNode;
  state?: StatusState;
}

/** Label on the left, value on the right, one row per fact, ruled between. */
export function PreviewFields({ rows }: { rows: readonly (PreviewFieldRow | null | false)[] }) {
  const present = rows.filter((row): row is PreviewFieldRow => Boolean(row));
  if (present.length === 0) {
    return null;
  }
  return (
    <dl className="flex flex-col">
      {present.map((row) => (
        <div
          key={row.label}
          className="border-border-subtle flex min-h-8.5 items-center justify-between gap-4 border-b py-1.5 text-base"
        >
          <dt className="text-foreground-subtle shrink-0">{row.label}</dt>
          <dd className="text-foreground flex min-w-0 items-center justify-end gap-1.5 text-right">
            {row.state && row.state !== "neutral" && <StatusDot state={row.state} />}
            <span className="truncate">{row.value}</span>
          </dd>
        </div>
      ))}
    </dl>
  );
}

export function PreviewSection({
  label,
  aside,
  children,
  className,
}: {
  label: string;
  aside?: React.ReactNode;
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <section className={cn("flex flex-col gap-1.5", className)}>
      <h4 className="text-foreground-faint flex items-center justify-between gap-2 font-mono text-2xs uppercase">
        <span>{label}</span>
        {aside !== undefined && aside !== null && <span className="normal-case">{aside}</span>}
      </h4>
      {children}
    </section>
  );
}

export interface PreviewStat {
  label: string;
  value: string;
  state?: StatusState;
}

const STAT_COLUMNS = ["grid-cols-1", "grid-cols-2", "grid-cols-3", "grid-cols-4"] as const;

/** A row of figures on a sunken strip, split by hairlines. */
export function PreviewStats({ items }: { items: readonly PreviewStat[] }) {
  if (items.length === 0) {
    return null;
  }
  return (
    <div
      className={cn(
        "bg-sunken rounded-surface divide-border-subtle grid divide-x",
        STAT_COLUMNS[Math.min(items.length, STAT_COLUMNS.length) - 1],
      )}
    >
      {items.map((item) => (
        <div key={item.label} className="flex min-w-0 flex-col gap-1 px-3.5 py-3">
          <span className="text-foreground-subtle truncate text-sm">{item.label}</span>
          <span className="text-foreground flex items-center gap-1.5 text-xl font-medium tabular-nums">
            <span className="truncate">{item.value}</span>
            {item.state && item.state !== "neutral" && (
              <StatusDot state={item.state} className="size-1.5" />
            )}
          </span>
        </div>
      ))}
    </div>
  );
}

/** A record the preview points at: hover fills it, a click opens it. */
export function RelatedRow({
  icon,
  initials,
  title,
  mono = false,
  sub,
  note,
  onOpen,
}: {
  icon?: PaletteIcon;
  initials?: string;
  title: string;
  mono?: boolean;
  sub?: React.ReactNode;
  note?: React.ReactNode;
  onOpen: () => void;
}) {
  return (
    <button
      type="button"
      tabIndex={-1}
      onMouseDown={keepFocus}
      onClick={onOpen}
      className="hover:bg-surface-hover rounded-control -mx-1.5 flex items-center gap-2.5 px-1.5 py-1.5 text-left transition-colors"
    >
      <PaletteTile icon={icon} initials={initials} size="related" />
      <span className="flex min-w-0 flex-1 flex-col">
        <span className={cn("text-foreground truncate text-base font-medium", mono && "font-mono")}>
          {title}
        </span>
        {sub && <span className="text-foreground-subtle truncate text-sm">{sub}</span>}
      </span>
      {note && <span className="text-foreground-subtle shrink-0 text-sm">{note}</span>}
      <ChevronRightIcon className="text-foreground-faint size-3 shrink-0" />
    </button>
  );
}

export function PreviewSkeleton() {
  return (
    <div aria-hidden className="flex flex-col gap-4.5">
      <Skeleton className="h-4 w-32" />
      <div className="flex flex-col gap-2">
        <Skeleton className="h-6 w-44" />
        <Skeleton className="h-3.5 w-56" />
      </div>
      <Skeleton className="rounded-surface h-24 w-full" />
      <div className="flex flex-col gap-3">
        {Array.from({ length: 4 }, (_, index) => (
          <div key={index} className="flex justify-between">
            <Skeleton className="h-3 w-20" />
            <Skeleton className="h-3 w-24" />
          </div>
        ))}
      </div>
    </div>
  );
}

export function PreviewMessage({ children }: { children: React.ReactNode }) {
  return (
    <div className="text-foreground-subtle flex h-full items-center justify-center px-6 text-center text-sm">
      {children}
    </div>
  );
}

export function PreviewError() {
  const t = useT();
  return (
    <PreviewMessage>
      {t("Couldn't load a preview. The record may have moved, or you may not have access to it.")}
    </PreviewMessage>
  );
}
