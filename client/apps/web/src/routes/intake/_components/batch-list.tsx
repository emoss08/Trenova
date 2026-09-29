import {
  captureBatchStatusAttrs,
  captureBatchTitle,
  captureRetention,
  captureSourceLabel,
} from "@/lib/capture";
import type { CaptureBatchRow, CaptureBatchSort } from "@/lib/graphql/capture";
import { errorCopy } from "@trenova/shared/components/errors/error-copy";
import {
  Alert,
  AlertAction,
  AlertDescription,
  AlertTitle,
} from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { Input } from "@trenova/shared/components/ui/input";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatSecondsAgo } from "@trenova/shared/lib/date";
import { describeError } from "@trenova/shared/lib/error-presentation";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import { cn } from "@trenova/shared/lib/utils";
import { PrinterIcon, ScanLineIcon, SearchIcon, XIcon } from "lucide-react";
import { useEffect, useRef, type ReactNode } from "react";
import {
  INTAKE_RECEIVED,
  INTAKE_SORTS,
  receivedLabel,
  sortLabel,
  type IntakeReceived,
} from "./queue-filter";

export type BatchListState = {
  batches: CaptureBatchRow[];
  total: number | undefined;
  isLoading: boolean;
  isError: boolean;
  error: unknown;
  hasNextPage: boolean;
  isFetchingNextPage: boolean;
  fetchNextPage: () => void;
  retry: () => void;
};

function BatchRowItem({
  batch,
  open,
  now,
  onOpen,
}: {
  batch: CaptureBatchRow;
  open: boolean;
  now: number;
  onOpen: (id: string) => void;
}) {
  const t = useT();
  const attrs = captureBatchStatusAttrs(t)[batch.status];
  const retention = captureRetention(batch.retainUntil, now);
  const SourceIcon = batch.source === "Print" ? PrinterIcon : ScanLineIcon;
  const destination = batch.target
    ? batch.target.subtitle === ""
      ? batch.target.title
      : `${batch.target.title} · ${batch.target.subtitle}`
    : null;

  return (
    <li>
      <button
        type="button"
        onClick={() => onOpen(batch.id)}
        aria-current={open ? "true" : undefined}
        className={cn(
          "ui-inset-focus-ring border-border-subtle flex w-full gap-3 border-b px-4 py-3 text-left transition-colors",
          open ? "bg-surface-selected" : "hover:bg-surface-hover",
        )}
      >
        <span className="flex min-w-0 flex-1 flex-col gap-1">
          <span className="flex items-baseline justify-between gap-2">
            <span className="flex min-w-0 items-center gap-1.5">
              <SourceIcon
                className="text-foreground-subtle size-3.5 shrink-0 self-center"
                aria-label={captureSourceLabel(t, batch.source)}
                role="img"
              />
              <span className="truncate text-sm">{captureBatchTitle(t, batch)}</span>
            </span>
            <span className="text-foreground-subtle shrink-0 text-xs">
              {formatSecondsAgo(Math.max(0, now - batch.createdAt))}
            </span>
          </span>
          <span className="text-foreground-muted truncate text-xs">
            {[
              t("{0, plural, one {# page} other {# pages}}", batch.receivedPageCount),
              batch.itemCount > 0
                ? t("{0, plural, one {# document} other {# documents}}", batch.itemCount)
                : null,
              batch.device?.name,
              batch.user?.name,
            ]
              .filter(Boolean)
              .join(" · ")}
          </span>
          <span className="flex flex-wrap items-center gap-1.5">
            <Badge variant={phaseTone(attrs.phase)} title={attrs.description}>
              {attrs.text}
            </Badge>
            {batch.isEditable && batch.openItemCount > 0 && batch.filedItemCount > 0 && (
              <span className="text-foreground-subtle text-xs tabular-nums">
                {t("{0} of {1} filed", batch.filedItemCount, batch.itemCount)}
              </span>
            )}
            {destination !== null && (
              <span className="text-foreground-muted truncate text-xs">→ {destination}</span>
            )}
            {batch.isEditable && retention.state === "soon" && (
              <Badge variant="warning">
                {t(
                  "{0, plural, one {Deleted in # day} other {Deleted in # days}}",
                  retention.daysLeft,
                )}
              </Badge>
            )}
          </span>
        </span>
      </button>
    </li>
  );
}

/**
 * The queue: one list of stacks, filtered by the rail and sorted here. The
 * next page loads as the bottom comes into view, with a button for the same
 * thing when it does not.
 */
export function BatchList({
  title,
  list,
  openId,
  now,
  search,
  onSearchChange,
  sort,
  onSortChange,
  received,
  onReceivedChange,
  onOpen,
  empty,
}: {
  title: string;
  list: BatchListState;
  openId: string | null;
  now: number;
  search: string;
  onSearchChange: (value: string) => void;
  sort: CaptureBatchSort;
  onSortChange: (sort: CaptureBatchSort) => void;
  received: IntakeReceived;
  onReceivedChange: (received: IntakeReceived) => void;
  onOpen: (id: string) => void;
  empty: { title: string; description: string; action?: ReactNode };
}) {
  const t = useT();
  const sentinelRef = useRef<HTMLDivElement>(null);
  const { hasNextPage, isFetchingNextPage, fetchNextPage } = list;

  useEffect(() => {
    const sentinel = sentinelRef.current;
    if (sentinel === null || !hasNextPage) {
      return;
    }
    const observer = new IntersectionObserver((entries) => {
      if (entries.some((entry) => entry.isIntersecting) && !isFetchingNextPage) {
        fetchNextPage();
      }
    });
    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  const sortItems = INTAKE_SORTS.map((value) => ({ value, label: sortLabel(t, value) }));
  const receivedItems = INTAKE_RECEIVED.map((value) => ({
    value,
    label: receivedLabel(t, value),
  }));

  return (
    <section aria-label={title} className="flex min-h-0 flex-1 flex-col">
      <div className="border-border flex flex-col gap-2 border-b px-4 py-3">
        <div className="flex items-baseline justify-between gap-2">
          <h2 className="truncate text-sm font-semibold">{title}</h2>
          {list.total !== undefined && (
            <span className="text-foreground-subtle text-xs tabular-nums">
              {t("{0, plural, one {# stack} other {# stacks}}", list.total)}
            </span>
          )}
        </div>
        <Input
          value={search}
          onChange={(event) => onSearchChange(event.target.value)}
          placeholder={t("Search PRO, barcode, scanner, computer or person")}
          aria-label={t("Search PRO, barcode, scanner, computer or person")}
          className="w-full"
          leftElement={<SearchIcon className="text-foreground-subtle size-3.5" />}
          rightElement={
            search === "" ? undefined : (
              <Button
                size="icon-xs"
                variant="ghost"
                aria-label={t("Clear the search")}
                onClick={() => onSearchChange("")}
              >
                <XIcon className="size-3.5" />
              </Button>
            )
          }
        />
        <div className="flex gap-2">
          <Select
            value={received}
            items={receivedItems}
            onValueChange={(value) => {
              const next = INTAKE_RECEIVED.find((option) => option === value);
              if (next !== undefined) {
                onReceivedChange(next);
              }
            }}
          >
            <SelectTrigger className="min-w-0 flex-1" aria-label={t("Received")}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {receivedItems.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Select
            value={sort}
            items={sortItems}
            onValueChange={(value) => {
              const next = INTAKE_SORTS.find((option) => option === value);
              if (next !== undefined) {
                onSortChange(next);
              }
            }}
          >
            <SelectTrigger className="min-w-0 flex-1" aria-label={t("Sort")}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {sortItems.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      <ScrollArea className="min-h-0 flex-1">
        {list.isLoading ? (
          <div className="flex flex-col gap-3 p-4" aria-busy="true">
            {[0, 1, 2, 3, 4].map((index) => (
              <div key={index} className="flex gap-3">
                <div className="flex flex-1 flex-col gap-1.5">
                  <Skeleton className="h-3.5 w-1/2" />
                  <Skeleton className="h-3.5 w-4/5" />
                  <Skeleton className="h-3 w-2/5" />
                </div>
              </div>
            ))}
          </div>
        ) : list.isError ? (
          <div className="p-4">
            <Alert variant="destructive" size="sm">
              <AlertTitle>{t("The queue could not be loaded")}</AlertTitle>
              <AlertDescription>
                {errorCopy(t, describeError(list.error)).description}
              </AlertDescription>
              <AlertAction>
                <Button size="xs" variant="outline" onClick={list.retry}>
                  {t("Try again")}
                </Button>
              </AlertAction>
            </Alert>
          </div>
        ) : list.batches.length === 0 ? (
          <EmptySheet
            title={empty.title}
            description={empty.description}
            action={empty.action}
            sketch={
              <div className="flex flex-col gap-3 px-6">
                {[0, 1, 2].map((index) => (
                  <div key={index} className="flex gap-3">
                    <div className="flex flex-1 flex-col gap-1.5">
                      <GhostLine className="w-1/2" />
                      <GhostLine className="w-4/5" />
                    </div>
                  </div>
                ))}
              </div>
            }
          />
        ) : (
          <>
            <ul>
              {list.batches.map((batch) => (
                <BatchRowItem
                  key={batch.id}
                  batch={batch}
                  open={batch.id === openId}
                  now={now}
                  onOpen={onOpen}
                />
              ))}
            </ul>
            <div ref={sentinelRef} className="flex justify-center p-3">
              {list.hasNextPage && (
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={list.fetchNextPage}
                  isLoading={list.isFetchingNextPage}
                  loadingText={t("Loading")}
                >
                  {t("Load more")}
                </Button>
              )}
            </div>
          </>
        )}
      </ScrollArea>
    </section>
  );
}
