import { VirtualRows, type VirtualRow, type VirtualRowsHandle } from "@/components/virtual-rows";
import { Button } from "@trenova/shared/components/ui/button";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useMemo, useRef, type ReactNode } from "react";

const ROW_ESTIMATE = 46;
const OVERSCAN = 6;

/** One run of rows under a heading: a lane, a kind of write, what was decided today. */
export type QueueSection = {
  key: string;
  header: ReactNode;
  className?: string;
  rows: readonly { key: string; render: () => ReactNode }[];
};

/**
 * The queue's rows, mounted only near the viewport so a queue of a thousand
 * reads as fast as one of ten. The selected row is kept in view as the person
 * walks the queue from the keyboard.
 */
export function QueueList({
  label,
  sections,
  selectedKey,
  empty,
  footer,
  onEndReached,
}: {
  label: string;
  sections: readonly QueueSection[];
  selectedKey: string | null;
  empty?: ReactNode;
  footer?: ReactNode;
  onEndReached?: () => void;
}) {
  const handle = useRef<VirtualRowsHandle>(null);
  const rows = useMemo<VirtualRow[]>(
    () =>
      sections.flatMap((section) => [
        {
          key: `section:${section.key}`,
          render: () => <div className={cn("dk-dc2-g", section.className)}>{section.header}</div>,
        },
        ...section.rows,
      ]),
    [sections],
  );

  useEffect(() => {
    if (selectedKey !== null) {
      handle.current?.scrollToKey(selectedKey);
    }
  }, [selectedKey]);

  return (
    <VirtualRows
      rows={rows}
      handleRef={handle}
      estimateSize={ROW_ESTIMATE}
      overscan={OVERSCAN}
      className="dk-dc2-ql"
      aria-label={label}
      empty={<div className="dk-dc2-ql">{empty}</div>}
      footer={footer}
      onEndReached={onEndReached}
    />
  );
}

/** A section's heading and how many rows are under it. */
export function QueueGroupHeader({ label, count }: { label: ReactNode; count: number }) {
  return (
    <div className="dk-dc2-gh">
      {label}
      <i>{count}</i>
    </div>
  );
}

/**
 * One row of the queue: a mark for what it is, its title over one line of
 * context, and whatever trails it — a countdown, an unopened dot.
 */
export function QueueRow({
  selected,
  marked = false,
  onSelect,
  leading,
  title,
  subtitle,
  trailing,
  className,
}: {
  selected: boolean;
  marked?: boolean;
  onSelect: () => void;
  leading: ReactNode;
  title: string;
  subtitle: ReactNode;
  trailing?: ReactNode;
  className?: string;
}) {
  return (
    <Button
      variant="bare"
      size="bare"
      className={cn(
        "dk-dc2-qi relative flex w-full gap-2.25 rounded-lg px-2 py-1.75 text-left transition-colors duration-100",
        selected ? "dk-on bg-dsk-card ring-1 ring-dsk-b" : "hover:bg-dsk-hover",
        marked && "dk-in",
        className,
      )}
      aria-current={selected ? "true" : undefined}
      onClick={onSelect}
    >
      {leading}
      <span className="flex min-w-0 flex-1 flex-col">
        <b className="truncate text-sm font-medium">{title}</b>
        <em className="truncate text-xs text-dsk-subtle not-italic @max-[900px]/mainc:hidden">
          {subtitle}
        </em>
      </span>
      {trailing}
    </Button>
  );
}

/** The dot on a row nobody has opened yet. */
export function QueueUnseenDot({ label }: { label: string }) {
  return (
    <span className="dk-dc2-new">
      <span className="sr-only">{label}</span>
    </span>
  );
}
