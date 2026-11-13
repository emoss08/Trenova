import type { VirtualRow, VirtualRowsHandle } from "@/components/virtual-rows";
import { DeskList, DeskListHeading, DeskListStill, deskListRow } from "../list/desk-list";
import { Button } from "@trenova/shared/components/ui/button";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useMemo, useRef, type ReactNode } from "react";

const ROW_ESTIMATE = 46;
const OVERSCAN = 6;

/** One run of rows under a heading: a lane, a kind of write, what was decided today. */
export type QueueSection = {
  key: string;
  header: ReactNode;
  rows: readonly { key: string; render: () => ReactNode }[];
};

/**
 * The queue's rows, in the Desk's one list pane (list/desk-list.tsx), mounted
 * only near the viewport so a queue of a thousand reads as fast as one of ten. The selected row is kept in view as the person
 * walks the queue from the keyboard.
 */
export function QueueList({
  label,
  sections,
  selectedKey,
  empty,
  footer,
  onEndReached,
  className,
}: {
  label: string;
  sections: readonly QueueSection[];
  selectedKey: string | null;
  /** Classes on the list itself: whether its rows have settled, a dimming while it is read again. */
  className?: string;
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
          render: () => <DeskListHeading>{section.header}</DeskListHeading>,
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
    <DeskList
      rows={rows}
      handleRef={handle}
      estimateSize={ROW_ESTIMATE}
      overscan={OVERSCAN}
      aria-label={label}
      className={className}
      empty={<DeskListStill>{empty}</DeskListStill>}
      footer={footer}
      onEndReached={onEndReached}
    />
  );
}

/** A section's heading and how many rows are under it. */
export function QueueGroupHeader({ label, count }: { label: ReactNode; count: number }) {
  return (
    <span className="flex items-center gap-1.5">
      {label}
      <i className="text-dsk-subtle not-italic tabular-nums">{count}</i>
    </span>
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
      data-glide-target
      className={cn(
        "dk-dc2-qi flex w-full gap-2.25 px-2 py-1.75",
        deskListRow(selected).className,
        selected && "dk-on",
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
