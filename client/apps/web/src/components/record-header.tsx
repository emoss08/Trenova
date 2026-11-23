import { RefreshCw01Icon } from "@trenova/shared/components/icons";
import { Tabs, TabsList } from "@trenova/shared/components/ui/tabs";
import { useNow } from "@trenova/shared/hooks/use-now";
import { formatRelativeTime } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";

/** Up to two letters for a record's mark: the first of its first two words. */
export function initialsOf(name: string): string {
  const letters = name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((word) => word[0]?.toUpperCase() ?? "");
  return letters.join("") || "?";
}

/** The 40px square that stands for a record without a picture: its initials in mono. */
export function RecordMark({ name, className }: { name: string; className?: string }) {
  return (
    <span
      aria-hidden
      className={cn(
        "grid size-10 shrink-0 place-items-center rounded-md border bg-sunken font-mono text-sm font-semibold tracking-wide",
        className,
      )}
    >
      {initialsOf(name)}
    </span>
  );
}

/** One fact on a record header's meta line: a small icon and a value that truncates. */
export function MetaItem({ icon, children }: { icon: ReactNode; children: ReactNode }) {
  return (
    <span className="inline-flex min-w-0 items-center gap-1.5 [&>svg]:size-3 [&>svg]:shrink-0">
      {icon}
      <span className="truncate">{children}</span>
    </span>
  );
}

/** When the record was read, moving on the minute without redrawing the rest of the page. */
export function UpdatedAgo({
  at,
  onRefresh,
  busy,
}: {
  /** Epoch milliseconds of the read, as TanStack Query's `dataUpdatedAt`. */
  at: number;
  onRefresh: () => void;
  busy: boolean;
}) {
  const t = useT();
  const now = useNow("minute");
  return (
    <button
      type="button"
      onClick={onRefresh}
      disabled={busy}
      title={t("Read the figures again")}
      className="inline-flex items-center gap-1.5 rounded-sm ui-focus-ring text-xs text-foreground-subtle transition-colors hover:text-foreground disabled:opacity-60"
    >
      <RefreshCw01Icon className={busy ? "size-3 animate-spin" : "size-3"} />
      {t("Updated {0}", formatRelativeTime(Math.min(Math.round(at / 1000) - now, 0)))}
    </button>
  );
}

/** A figure after a tab's name: how many things the tab holds, or which version is in force. */
export function TabCount({ children }: { children: ReactNode }) {
  return <span className="font-normal text-foreground-subtle tabular-nums">{children}</span>;
}

/** A dot after a tab's name that says something in it needs a look; `label` is what it says. */
export function TabDot({ tone, label }: { tone: "warning" | "danger"; label: string }) {
  return (
    <span
      role="img"
      aria-label={label}
      className={cn("size-1.5 rounded-full", tone === "warning" ? "bg-warning" : "bg-danger")}
    />
  );
}

export type RecordHeaderProps = {
  /** `RecordMark`, or a picture of the same size. */
  mark: ReactNode;
  title: ReactNode;
  /** Badges after the name. */
  badges?: ReactNode;
  /** `MetaItem`s, laid out on one wrapping line under the name. */
  meta?: ReactNode;
  /** The record's actions at the right, `size="sm"` buttons and an `icon-sm` menu. */
  actions?: ReactNode;
  /** The record's tabs (`RecordTabs`). */
  children?: ReactNode;
};

/**
 * A record page's header, on a card band under the page bar: the mark, the name with its badges,
 * the facts a person reaches for, the record's actions and its tabs.
 */
export function RecordHeader({ mark, title, badges, meta, actions, children }: RecordHeaderProps) {
  return (
    <header className="flex flex-col gap-2.5 border-b bg-card px-6 pt-3.5">
      <div className="flex flex-wrap items-start gap-x-3.5 gap-y-3">
        {mark}
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <div className="flex flex-wrap items-center gap-2">
            <h2 className="text-lg leading-6 font-semibold">{title}</h2>
            {badges}
          </div>
          {meta ? (
            <p className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">{meta}</p>
          ) : null}
        </div>
        {actions ? <div className="flex shrink-0 items-center gap-2">{actions}</div> : null}
      </div>
      {children}
    </header>
  );
}

export type RecordTabsProps<V extends string> = {
  value: V;
  onValueChange: (value: V) => void;
  /** At the row's right end from `sm` up, usually `UpdatedAgo`. */
  trailing?: ReactNode;
  /** `TabsTab`s, with `TabCount` or `TabDot` after a name. */
  children: ReactNode;
};

/** A record's underline tabs, sitting on the header's bottom border. */
export function RecordTabs<V extends string>({
  value,
  onValueChange,
  trailing,
  children,
}: RecordTabsProps<V>) {
  return (
    <Tabs value={value} onValueChange={(next) => onValueChange(next as V)}>
      <div className="-mb-px flex items-center gap-3">
        <TabsList variant="underline" className="-ml-2.5 max-w-full min-w-0 overflow-x-auto">
          {children}
        </TabsList>
        {trailing ? (
          <span className="ml-auto hidden shrink-0 sm:inline-flex">{trailing}</span>
        ) : null}
      </div>
    </Tabs>
  );
}
