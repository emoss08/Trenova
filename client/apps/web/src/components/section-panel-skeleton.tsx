import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";

/**
 * A `SectionPanel` while what fills it loads: the bordered card, its header
 * with the title and action where they land, and the body the section draws.
 * Only primitives, so a page can import it up front beside a lazy section.
 */
export function SectionPanelSkeleton({
  label,
  action = false,
  children,
}: {
  /** Names what loads, for assistive technology. */
  label: string;
  /** Whether the real header carries a button on its right. */
  action?: boolean;
  children: ReactNode;
}) {
  return (
    <section
      aria-busy="true"
      aria-label={label}
      className="bg-card flex flex-col overflow-hidden rounded-lg border"
    >
      <div aria-hidden className="contents">
        <div className="border-border flex min-h-9 items-center justify-between gap-2 border-b px-3 py-1.5">
          <Skeleton className="h-4 w-32" />
          {action ? <Skeleton className="h-6 w-24" /> : null}
        </div>
        {children}
      </div>
    </section>
  );
}

/** A section's body while its own query answers, drawn with the same placeholders. */
export function SectionBodySkeleton({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div aria-busy="true" aria-label={label}>
      <div aria-hidden className="contents">
        {children}
      </div>
    </div>
  );
}

const STACKED_COLUMNS = {
  2: "grid-cols-2",
  3: "grid-cols-2 md:grid-cols-3",
  4: "grid-cols-2 md:grid-cols-4",
} as const;

/** A stacked `DescriptionList`'s placeholders: a label over a value, per item. */
export function DescriptionListSkeleton({
  items,
  columns,
}: {
  items: number;
  columns: keyof typeof STACKED_COLUMNS;
}) {
  return (
    <div className={cn("grid gap-x-6 gap-y-3", STACKED_COLUMNS[columns])}>
      {Array.from({ length: items }, (_, index) => (
        <div key={index} className="flex flex-col gap-1">
          <Skeleton className="h-3 w-20" />
          <Skeleton className="h-4 w-28" />
        </div>
      ))}
    </div>
  );
}
