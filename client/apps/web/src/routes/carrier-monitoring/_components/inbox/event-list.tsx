import { RelativeTime } from "@/components/carrier-intelligence/relative-time";
import { StatusDot, severityTone } from "@/components/carrier-intelligence/status-dot";
import type { CarrierIntelLabels } from "@/components/carrier-intelligence/use-carrier-intel-labels";
import type { CarrierIntelEvent } from "@/lib/graphql/carrier-intelligence";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { memo, useEffect, useRef, type Ref } from "react";
import type { EventPresentation } from "@/components/carrier-intelligence/use-event-presenter";
import type { InboxEventGroup } from "./group-events";

export type EventListProps = {
  groups: InboxEventGroup[];
  labels: CarrierIntelLabels;
  present: (event: CarrierIntelEvent) => EventPresentation;
  focusedId: string | null;
  openId: string | null;
  selectedIds: ReadonlySet<string>;
  onRowClick: (event: CarrierIntelEvent) => void;
  onToggleSelected: (event: CarrierIntelEvent) => void;
  footer?: React.ReactNode;
  sentinelRef?: Ref<HTMLDivElement>;
};

type EventRowProps = {
  event: CarrierIntelEvent;
  title: string;
  categoryLabel: string;
  focused: boolean;
  open: boolean;
  selected: boolean;
  selectionActive: boolean;
  onRowClick: (event: CarrierIntelEvent) => void;
  onToggleSelected: (event: CarrierIntelEvent) => void;
};

const EventRow = memo(function EventRow({
  event,
  title,
  categoryLabel,
  focused,
  open,
  selected,
  selectionActive,
  onRowClick,
  onToggleSelected,
}: EventRowProps) {
  const t = useT();
  const rowRef = useRef<HTMLDivElement>(null);
  const settled = event.status !== "Open";

  useEffect(() => {
    if (focused) {
      rowRef.current?.scrollIntoView?.({ block: "nearest" });
    }
  }, [focused]);

  return (
    <div
      ref={rowRef}
      role="option"
      id={`carrier-intel-event-${event.id}`}
      aria-selected={focused}
      data-event-id={event.id}
      data-focused={focused ? "true" : undefined}
      data-open={open ? "true" : undefined}
      data-selected={selected ? "true" : undefined}
      onClick={() => onRowClick(event)}
      className={cn(
        "group relative flex h-14 cursor-pointer items-center gap-3 border-b border-border/60 px-3 transition-colors",
        "hover:bg-muted/50 data-[selected=true]:bg-muted/40 data-[open=true]:bg-muted data-[focused=true]:bg-muted/70",
        "data-[focused=true]:before:absolute data-[focused=true]:before:inset-y-0 data-[focused=true]:before:left-0 data-[focused=true]:before:w-0.5 data-[focused=true]:before:bg-foreground/60",
      )}
    >
      <div className="relative flex size-4 shrink-0 items-center justify-center">
        <span
          className={cn(
            "flex items-center justify-center transition-opacity",
            (selected || selectionActive) && "opacity-0",
            !selected && !selectionActive && "group-hover:opacity-0",
          )}
        >
          <StatusDot tone={severityTone(event.severity)} />
        </span>
        <span
          className={cn(
            "absolute inset-0 flex items-center justify-center transition-opacity",
            selected || selectionActive ? "opacity-100" : "opacity-0 group-hover:opacity-100",
          )}
          onClick={(clickEvent) => clickEvent.stopPropagation()}
        >
          <Checkbox
            checked={selected}
            onCheckedChange={() => onToggleSelected(event)}
            aria-label={t("Select {0}", title)}
          />
        </span>
      </div>
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span
          className={cn(
            "truncate text-sm",
            settled ? "text-muted-foreground" : "text-foreground font-medium",
          )}
          title={title}
        >
          {title}
        </span>
        <span className="text-muted-foreground flex min-w-0 items-center gap-1.5 text-xs">
          <span className="truncate">{event.subjectName || t("USDOT {0}", event.dotNumber)}</span>
          {event.subjectName ? (
            <>
              <span aria-hidden>·</span>
              <span className="shrink-0 tabular-nums">{t("USDOT {0}", event.dotNumber)}</span>
            </>
          ) : null}
          <span aria-hidden>·</span>
          <span className="shrink-0">{categoryLabel}</span>
          <span aria-hidden>·</span>
          <RelativeTime timestamp={event.detectedAt} className="shrink-0" />
        </span>
      </div>
    </div>
  );
});

export function EventList({
  groups,
  labels,
  present,
  focusedId,
  openId,
  selectedIds,
  onRowClick,
  onToggleSelected,
  footer,
  sentinelRef,
}: EventListProps) {
  const t = useT();
  const selectionActive = selectedIds.size > 0;

  return (
    <div
      role="listbox"
      aria-label={t("Carrier changes")}
      aria-multiselectable="true"
      aria-activedescendant={focusedId ? `carrier-intel-event-${focusedId}` : undefined}
      className="flex flex-col"
    >
      {groups.map((group) => (
        <div key={group.key} role="group" aria-label={group.label}>
          <div className="bg-background/95 supports-[backdrop-filter]:bg-background/80 sticky top-0 z-10 flex items-center border-b border-border/60 px-3 py-1.5 backdrop-blur">
            <span className="text-muted-foreground truncate text-xs font-medium">
              {group.label}
            </span>
          </div>
          {group.events.map((event) => (
            <EventRow
              key={event.id}
              event={event}
              title={present(event).title}
              categoryLabel={labels.section[event.category]}
              focused={event.id === focusedId}
              open={event.id === openId}
              selected={selectedIds.has(event.id)}
              selectionActive={selectionActive}
              onRowClick={onRowClick}
              onToggleSelected={onToggleSelected}
            />
          ))}
        </div>
      ))}
      <div ref={sentinelRef} aria-hidden className="h-px" />
      {footer}
    </div>
  );
}
