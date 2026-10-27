import { useT } from "@trenova/shared/i18n/use-t";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import {
  AlertCircleIcon,
  AlertTriangleIcon,
  ArrowNarrowRightIcon,
  CheckCircleIcon,
  CircleIcon,
  Copy01Icon,
  InfoCircleIcon,
  XCircleIcon,
  type IconComponent,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet } from "@trenova/shared/components/ui/empty-sheet";
import { HighlightedCode } from "@trenova/shared/components/ui/highlighted-code";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import type { LadingSeverity } from "@trenova/shared/edi/lading";
import { x12Spans } from "@trenova/shared/lib/highlight/x12";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";

// The parts every studio panel is built from, so the six tabs read as one tool: a toolbar that
// stays put while the panel scrolls, an empty state, status and severity marks, filter chips with
// counts, a proportion bar, code with a copy button and a before-and-after list.

/** The row of controls at the top of a panel; it stays in view while the panel scrolls. */
export function PanelToolbar({ children, end }: { children: ReactNode; end?: ReactNode }) {
  return (
    <div className="bg-background sticky top-0 z-10 flex flex-wrap items-center gap-2 border-b px-3 py-2">
      <div className="flex min-w-0 flex-1 flex-wrap items-center gap-2">{children}</div>
      {end ? <div className="flex shrink-0 flex-wrap items-center gap-2">{end}</div> : null}
    </div>
  );
}

type Tone = "neutral" | "success" | "warning" | "danger" | "info";

const TILE_TONE: Record<Tone, string> = {
  neutral: "bg-muted text-muted-foreground",
  success: "bg-success-subtle text-success-subtle-foreground",
  warning: "bg-warning-subtle text-warning-subtle-foreground",
  danger: "bg-danger-subtle text-danger-subtle-foreground",
  info: "bg-info-subtle text-info-subtle-foreground",
};

/**
 * An empty panel drawn as what it will hold (`EmptySheet`): a faint sketch of the tab's own
 * layout over what the tab is for and the one thing to do, sized for the short panel under the
 * editor. Each tab's sketch is in `lading-panel-sketches.tsx`.
 */
export function PanelSheet({
  sketch,
  title,
  description,
  action,
  compact = false,
}: {
  sketch: ReactNode;
  title: string;
  description: string;
  action?: ReactNode;
  compact?: boolean;
}) {
  return (
    <EmptySheet
      sketch={sketch}
      title={title}
      description={description}
      action={action}
      className={compact ? "gap-3 py-4" : "gap-4 py-6"}
      sketchClassName="max-w-sm"
    />
  );
}

/** What a panel shows when its work failed: why, and the way out. */
export function PanelFailure({
  icon: Icon,
  title,
  description,
  action,
  tone = "neutral",
  compact = false,
}: {
  icon: IconComponent;
  title: string;
  description?: string;
  action?: ReactNode;
  tone?: Tone;
  compact?: boolean;
}) {
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center gap-2 px-4 text-center",
        compact ? "py-5" : "py-10",
      )}
    >
      <span className={cn("flex size-9 items-center justify-center rounded-lg", TILE_TONE[tone])}>
        <Icon className="size-4.5" />
      </span>
      <div className="flex max-w-md flex-col gap-1">
        <p className="text-sm font-medium">{title}</p>
        {description ? <p className="text-muted-foreground text-xs">{description}</p> : null}
      </div>
      {action ? <div className="mt-1">{action}</div> : null}
    </div>
  );
}

const SEVERITY_ICON: Record<LadingSeverity, { icon: IconComponent; className: string }> = {
  error: { icon: AlertCircleIcon, className: "text-danger" },
  warning: { icon: AlertTriangleIcon, className: "text-warning" },
  info: { icon: InfoCircleIcon, className: "text-info" },
};

export function SeverityIcon({
  severity,
  className,
}: {
  severity: LadingSeverity;
  className?: string;
}) {
  const { icon: Icon, className: tone } = SEVERITY_ICON[severity];
  return <Icon aria-hidden className={cn("size-3.5 shrink-0", tone, className)} />;
}

export type Status = "passed" | "failed" | "warning" | "running" | "pending";

/** A check's outcome as a mark: passed, failed, passed with something to look at, or under way. */
export function StatusIcon({ status, className }: { status: Status; className?: string }) {
  const base = cn("size-3.5 shrink-0", className);
  switch (status) {
    case "passed":
      return <CheckCircleIcon aria-hidden className={cn(base, "text-success")} />;
    case "failed":
      return <XCircleIcon aria-hidden className={cn(base, "text-danger")} />;
    case "warning":
      return <AlertTriangleIcon aria-hidden className={cn(base, "text-warning")} />;
    case "running":
      return <Spinner aria-hidden className={cn(base, "text-muted-foreground")} />;
    case "pending":
      return <CircleIcon aria-hidden className={cn(base, "text-muted-foreground")} />;
  }
}

export type Chip<T extends string> = { value: T; label: string; count?: number; tone?: Tone };

const COUNT_TONE: Record<Tone, string> = {
  neutral: "text-muted-foreground",
  success: "text-success",
  warning: "text-warning",
  danger: "text-danger",
  info: "text-info",
};

/** One-of-many filter chips, each with the number of rows it keeps. */
export function FilterChips<T extends string>({
  value,
  items,
  onChange,
  label,
}: {
  value: T;
  items: readonly Chip<T>[];
  onChange: (value: T) => void;
  label: string;
}) {
  return (
    <div role="radiogroup" aria-label={label} className="flex flex-wrap items-center gap-1">
      {items.map((item) => {
        const selected = item.value === value;
        return (
          <button
            key={item.value}
            type="button"
            role="radio"
            aria-checked={selected}
            onClick={() => onChange(item.value)}
            className={cn(
              "ui-focus-ring flex h-6 items-center gap-1.5 rounded-md border px-2 text-xs",
              selected
                ? "bg-surface-active border-border text-foreground"
                : "text-muted-foreground hover:bg-surface-hover border-transparent",
            )}
          >
            {item.label}
            {item.count !== undefined ? (
              <span
                className={cn(
                  "tabular-nums",
                  item.count > 0 ? COUNT_TONE[item.tone ?? "neutral"] : "text-muted-foreground",
                )}
              >
                {item.count}
              </span>
            ) : null}
          </button>
        );
      })}
    </div>
  );
}

const BAR_TONE: Record<Tone, string> = {
  neutral: "bg-foreground-subtle",
  success: "bg-success",
  warning: "bg-warning",
  danger: "bg-danger",
  info: "bg-info",
};

/**
 * Parts of a whole as one bar; a part of nothing takes no width. The whole is the parts' sum
 * unless `total` names a larger one, whose remainder stays the empty track.
 */
export function ProportionBar({
  parts,
  label,
  total: whole,
  className,
}: {
  parts: readonly { value: number; tone: Tone }[];
  label: string;
  total?: number;
  className?: string;
}) {
  const total = Math.max(
    whole ?? 0,
    parts.reduce((sum, part) => sum + part.value, 0),
  );
  return (
    <div
      role="img"
      aria-label={label}
      className={cn("bg-muted flex h-1.5 w-full overflow-hidden rounded-full", className)}
    >
      {total > 0
        ? parts.map((part, index) =>
            part.value > 0 ? (
              <span
                // oxlint-disable-next-line react/no-array-index-key -- parts are positional
                key={index}
                className={BAR_TONE[part.tone]}
                style={{ width: `${(part.value / total) * 100}%` }}
              />
            ) : null,
          )
        : null}
    </div>
  );
}

/** Code under a titled bar with a copy button; long or wide code scrolls inside the frame. */
export function CodeBlock({
  title,
  code,
  language,
  end,
}: {
  title: string;
  code: string;
  language?: string;
  end?: ReactNode;
}) {
  const t = useT();
  const { copy, isCopied } = useCopyToClipboard();
  return (
    <div className="overflow-hidden rounded-md border">
      <div className="bg-muted flex h-8 items-center gap-2 border-b pr-1 pl-3">
        <span className="min-w-0 flex-1 truncate text-xs font-medium">{title}</span>
        {end}
        <Button
          type="button"
          variant="ghost"
          size="xxs"
          onClick={() => void copy(code)}
          aria-label={t("Copy {0}", title)}
        >
          <Copy01Icon className="size-3" />
          {isCopied ? t("Copied") : t("Copy")}
        </Button>
      </div>
      <ScrollArea className="max-h-72">
        <HighlightedCode
          code={code}
          language={language}
          spans={language === "x12" ? x12Spans(code) : undefined}
          className="w-max min-w-full p-3"
        />
      </ScrollArea>
    </div>
  );
}

/** A value as the differences show it: JSON, or a dash for a value that is absent. */
function shown(value: unknown): string {
  return value === undefined ? "—" : JSON.stringify(value);
}

/** Fields that changed, each its path with the old value struck and the new one beside it. */
export function DiffList({
  differences,
}: {
  differences: readonly { path: string; before?: unknown; after?: unknown }[];
}) {
  const t = useT();
  return (
    <ul className="flex flex-col gap-1">
      {differences.map((difference) => (
        <li
          key={difference.path}
          className="grid grid-cols-[minmax(8rem,14rem)_minmax(0,1fr)] items-baseline gap-3 font-mono text-xs"
        >
          <span className="text-muted-foreground truncate" title={difference.path}>
            {difference.path}
          </span>
          <span className="flex min-w-0 flex-wrap items-center gap-1.5">
            <span className="sr-only">{t("was")}</span>
            <span className="bg-danger-subtle text-danger-subtle-foreground max-w-full truncate rounded px-1 line-through">
              {shown(difference.before)}
            </span>
            <ArrowNarrowRightIcon aria-hidden className="text-muted-foreground size-3 shrink-0" />
            <span className="sr-only">{t("now")}</span>
            <span className="bg-success-subtle text-success-subtle-foreground max-w-full truncate rounded px-1">
              {shown(difference.after)}
            </span>
          </span>
        </li>
      ))}
    </ul>
  );
}

/** Small labelled facts in a row, for a result's header. */
export function MetaList({
  items,
}: {
  items: readonly { label: string; value: ReactNode; hidden?: boolean }[];
}) {
  return (
    <dl className="flex flex-wrap items-baseline gap-x-5 gap-y-1 text-xs">
      {items
        .filter((item) => !item.hidden)
        .map((item) => (
          <div key={item.label} className="flex items-baseline gap-1.5">
            <dt className="text-muted-foreground">{item.label}</dt>
            <dd className="tabular-nums">{item.value}</dd>
          </div>
        ))}
    </dl>
  );
}

/** A small heading inside a panel, with an optional count and a line of help. */
export function PanelHeading({
  title,
  count,
  help,
  end,
}: {
  title: string;
  count?: number;
  help?: string;
  end?: ReactNode;
}) {
  return (
    <div className="flex items-start gap-2">
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <h3 className="flex items-baseline gap-1.5 text-sm font-semibold">
          {title}
          {count !== undefined ? (
            <span className="text-muted-foreground text-xs font-normal tabular-nums">{count}</span>
          ) : null}
        </h3>
        {help ? <p className="text-muted-foreground text-xs">{help}</p> : null}
      </div>
      {end}
    </div>
  );
}
