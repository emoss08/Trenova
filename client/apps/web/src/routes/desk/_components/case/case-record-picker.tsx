"use no memo";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import type { CaseSubjectType, MentionCandidateRecord, MentionPageKind } from "@/types/assistant";
import { SearchLgIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useVirtualizer } from "@tanstack/react-virtual";
import { useEffect, useId, useRef, useState, type KeyboardEvent } from "react";
import { useCaseRecords } from "./use-case-records";

/** Rows from the end at which the next page is asked for. */
const PREFETCH_ROWS = 6;
const ROW_HEIGHT = 44;
const STATUS_HEIGHT = 40;

/** Which search finds each kind of case record. */
const PICKER_KINDS: readonly { subject: CaseSubjectType; kind: MentionPageKind }[] = [
  { subject: "Shipment", kind: "shipment" },
  { subject: "Invoice", kind: "invoice" },
  { subject: "InvoiceDispute", kind: "invoice_dispute" },
];

function kindLabel(subject: CaseSubjectType, t: TranslateFn): string {
  switch (subject) {
    case "Shipment":
      return t("Shipment");
    case "Invoice":
      return t("Invoice");
    case "InvoiceDispute":
      return t("Dispute");
  }
}

/**
 * The records a conversation can be made a case about: a kind, a search, and
 * every matching record in one list that loads a page at a time as it
 * scrolls and draws only the rows in view, however far it goes.
 */
export function CaseRecordPicker({
  heading,
  onPick,
  onBack,
}: {
  heading: string;
  onPick: (subjectType: CaseSubjectType, subjectId: string) => void;
  onBack?: () => void;
}) {
  const t = useT();
  const listId = useId();
  const [kind, setKind] = useState(PICKER_KINDS[0]);
  const [search, setSearch] = useState("");
  const [active, setActive] = useState(0);
  const scrollRef = useRef<HTMLDivElement>(null);
  const found = useCaseRecords(search, kind.kind);
  const { records, hasNextPage, isFetchingNextPage, fetchNextPage } = found;
  const count = records.length + (hasNextPage ? 1 : 0);

  const virtualizer = useVirtualizer({
    count,
    getScrollElement: () => scrollRef.current,
    estimateSize: (index) => (index < records.length ? ROW_HEIGHT : STATUS_HEIGHT),
    getItemKey: (index) => records[index]?.id ?? "status",
    overscan: 8,
    initialRect: { width: 320, height: 320 },
  });
  const virtualItems = virtualizer.getVirtualItems();
  const lastVisible = virtualItems.at(-1)?.index ?? 0;

  useEffect(() => {
    if (hasNextPage && !isFetchingNextPage && lastVisible >= records.length - PREFETCH_ROWS) {
      void fetchNextPage();
    }
  }, [fetchNextPage, hasNextPage, isFetchingNextPage, lastVisible, records.length]);

  const activeIndex = Math.min(active, records.length - 1);
  const pick = (record: MentionCandidateRecord) => onPick(kind.subject, record.id);

  const moveTo = (position: number) => {
    if (records.length === 0) {
      return;
    }
    const bounded = Math.max(0, Math.min(position, records.length - 1));
    setActive(bounded);
    virtualizer.scrollToIndex(bounded, { align: "auto" });
  };

  const onKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    switch (event.key) {
      case "ArrowDown":
        event.preventDefault();
        moveTo(activeIndex + 1);
        break;
      case "ArrowUp":
        event.preventDefault();
        moveTo(activeIndex - 1);
        break;
      case "Home":
        event.preventDefault();
        moveTo(0);
        break;
      case "End":
        event.preventDefault();
        moveTo(records.length - 1);
        break;
      case "Enter": {
        const record = records[activeIndex];
        if (record) {
          event.preventDefault();
          pick(record);
        }
        break;
      }
      default:
        break;
    }
  };

  const optionId = (index: number) => `${listId}-${index}`;
  const placeholder =
    kind.subject === "Shipment" ? t("PRO or BOL number") : t("Invoice number or customer");

  return (
    <div className="flex flex-col">
      <div className="flex items-center gap-1 px-2 pt-2">
        {onBack && (
          <Button size="icon-xs" variant="ghost" aria-label={t("Back")} onClick={onBack}>
            <DeskIcon name="chevL" size={13} />
          </Button>
        )}
        <p className="text-sm font-medium">{heading}</p>
      </div>
      <div className="flex gap-1 px-2 pt-2" role="tablist" aria-label={t("Kind of record")}>
        {PICKER_KINDS.map((candidate) => (
          <Button
            key={candidate.subject}
            variant="bare"
            size="bare"
            role="tab"
            aria-selected={candidate.subject === kind.subject}
            className={cn(
              "text-muted-foreground hover:text-foreground h-6 rounded-md px-2 text-xs",
              candidate.subject === kind.subject && "bg-surface-active text-foreground",
            )}
            onClick={() => {
              setKind(candidate);
              setActive(0);
            }}
          >
            {kindLabel(candidate.subject, t)}
          </Button>
        ))}
      </div>
      <div className="border-border border-b p-2">
        <Input
          autoFocus
          value={search}
          onChange={(event) => {
            setSearch(event.target.value);
            setActive(0);
          }}
          onKeyDown={onKeyDown}
          placeholder={placeholder}
          aria-label={t("Find the record")}
          role="combobox"
          aria-expanded
          aria-controls={listId}
          aria-activedescendant={activeIndex >= 0 ? optionId(activeIndex) : undefined}
          aria-autocomplete="list"
          className="h-8 text-sm"
          leftElement={<SearchLgIcon className="text-muted-foreground size-3.5" />}
          rightElement={
            found.refreshing ? <Spinner className="text-muted-foreground mr-1 size-3.5" /> : null
          }
        />
      </div>

      {found.loading ? (
        <div className="flex flex-col gap-1.5 p-2" aria-busy>
          <Skeleton className="h-9" />
          <Skeleton className="h-9" />
          <Skeleton className="h-9" />
        </div>
      ) : found.failed && records.length === 0 ? (
        <div className="flex flex-col items-center gap-2 px-4 py-6 text-center">
          <p className="text-muted-foreground text-sm">{t("The records could not be loaded.")}</p>
          <Button size="xs" variant="outline" onClick={() => void found.refetch()}>
            {t("Try again")}
          </Button>
        </div>
      ) : records.length === 0 ? (
        <p className="text-muted-foreground px-4 py-6 text-center text-sm">
          {found.settledSearch === ""
            ? t("There are none to pick yet.")
            : t("Nothing matches “{0}”.", found.settledSearch)}
        </p>
      ) : (
        <div
          ref={scrollRef}
          id={listId}
          role="listbox"
          aria-label={t("Records")}
          className={cn(
            "scrollbar-overlay max-h-80 overflow-y-auto overscroll-contain p-1 transition-opacity",
            found.refreshing && "opacity-70",
          )}
        >
          <div className="relative w-full" style={{ height: virtualizer.getTotalSize() }}>
            {virtualItems.map((item) => {
              const record = records[item.index];

              return (
                <div
                  key={item.key}
                  data-index={item.index}
                  ref={virtualizer.measureElement}
                  className="absolute inset-x-0 top-0"
                  style={{ transform: `translateY(${item.start}px)` }}
                >
                  {record ? (
                    <div
                      id={optionId(item.index)}
                      role="option"
                      aria-selected={item.index === activeIndex}
                      onMouseMove={() => setActive(item.index)}
                      onMouseDown={(event) => event.preventDefault()}
                      onClick={() => pick(record)}
                      className={cn(
                        "flex h-11 cursor-pointer items-center gap-2.5 rounded-md px-2 transition-colors",
                        item.index === activeIndex && "bg-surface-hover",
                      )}
                    >
                      <DeskIcon name={kind.subject === "Shipment" ? "truck" : "receipt"} size={13} />
                      <span className="flex min-w-0 flex-col">
                        <span className="truncate text-sm font-medium">{record.label}</span>
                        {record.subtitle && (
                          <span className="text-muted-foreground truncate text-xs">
                            {record.subtitle}
                          </span>
                        )}
                      </span>
                    </div>
                  ) : (
                    <div role="presentation" className="flex h-10 items-center justify-center">
                      <Spinner className="text-muted-foreground size-3.5" />
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
