import type { IconComponent } from "@trenova/shared/components/icons";
import { Tabs, TabsList, TabsTab } from "@trenova/shared/components/ui/tabs";
import { cn } from "@trenova/shared/lib/utils";

export type SectionStripTone = "none" | "success" | "warning" | "danger";

export type SectionStripItem<V extends string = string> = {
  value: V;
  label: string;
  icon: IconComponent;
  /** One line on how this part of the record stands ("8 of 8 required on file · 1 expiring"). */
  state?: string | null;
  tone?: SectionStripTone;
  disabled?: boolean;
};

export type SectionStripProps<V extends string = string> = {
  items: readonly SectionStripItem<V>[];
  value: V;
  onValueChange: (value: V) => void;
  "aria-label": string;
  /** The state lines are still being read: each cell draws a shimmer where its line goes. */
  loading?: boolean;
  /** The pointer or focus rests on a cell: start reading that view's code. */
  onValueIntent?: (value: V) => void;
  className?: string;
};

const TONE_CLASS: Record<SectionStripTone, string> = {
  none: "text-muted-foreground",
  success: "text-success-foreground",
  warning: "text-warning-foreground",
  danger: "text-danger-foreground",
};

/**
 * The views of one tab as a row of two-line cells: each names a view and says in one line how
 * that part of the record stands, so the strip is the tab's navigation and its status line at
 * once. It is the product's default tab list grown to two lines, so its selection plate is the
 * same gliding indicator, and it re-places itself when the strip's width changes.
 */
export function SectionStrip<V extends string>({
  items,
  value,
  onValueChange,
  "aria-label": ariaLabel,
  loading = false,
  onValueIntent,
  className,
}: SectionStripProps<V>) {
  return (
    <Tabs
      value={value}
      onValueChange={(next) => onValueChange(next as V)}
      className={cn("@container min-w-0 gap-0", className)}
    >
      <TabsList
        aria-label={ariaLabel}
        activateOnFocus
        className={cn(
          "grid w-full auto-cols-[minmax(0,1fr)] grid-flow-col items-stretch justify-stretch gap-0.5",
          "overflow-x-auto rounded-lg border bg-sunken p-1 text-muted-foreground [scrollbar-width:none]",
          "@max-[900px]:auto-cols-[minmax(11rem,1fr)]",
        )}
      >
        {items.map((item) => {
          const Icon = item.icon;
          return (
            <TabsTab
              key={item.value}
              value={item.value}
              disabled={item.disabled}
              onPointerEnter={onValueIntent ? () => onValueIntent(item.value) : undefined}
              onFocus={onValueIntent ? () => onValueIntent(item.value) : undefined}
              className={cn(
                "h-auto min-w-0 grow-0 flex-col items-stretch justify-start gap-0.5 px-3 py-2 text-left",
                "sm:h-auto sm:text-sm text-sm hover:text-foreground",
              )}
            >
              <span className="flex min-w-0 items-center gap-1.5 font-medium whitespace-nowrap">
                <Icon className="size-3.5" aria-hidden />
                <span className="truncate">{item.label}</span>
              </span>
              {loading ? (
                <span aria-hidden className="my-0.5 block h-3 w-3/4 ui-shimmer rounded-sm" />
              ) : item.state ? (
                <span
                  data-tone={item.tone ?? "none"}
                  className={cn(
                    "truncate text-xs font-normal tabular-nums",
                    TONE_CLASS[item.tone ?? "none"],
                  )}
                >
                  {item.state}
                </span>
              ) : null}
            </TabsTab>
          );
        })}
      </TabsList>
    </Tabs>
  );
}
