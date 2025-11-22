/* eslint-disable react-hooks/exhaustive-deps */
import { useDataTable } from "@/components/data-table/data-table-provider";
import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { JsonViewer } from "@/components/ui/json-viewer";
import { Kbd } from "@/components/ui/kbd";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { HeaderBackButton } from "@/components/ui/sheet-header-components";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { VisuallyHidden } from "@/components/ui/visually-hidden";
import { formatToUserTimezone } from "@/lib/date";
import { useUser } from "@/stores/user-store";
import { AuditEntry } from "@/types/audit-entry";
import { EditTableSheetProps } from "@/types/data-table";
import { faChevronDown, faChevronUp } from "@fortawesome/pro-solid-svg-icons";
import { memo, useCallback, useEffect, useMemo } from "react";
import { AuditActions } from "./audit-actions";
import { ActionBadge } from "./audit-column-components";
import { ChangesTable } from "./audit-log-data-section";
import { AuditLogDetails } from "./audit-log-details";

export function AuditLogDetailsSheet({
  currentRecord,
}: EditTableSheetProps<AuditEntry>) {
  const { table, rowSelection, isLoading } = useDataTable();
  const selectedRowKey = Object.keys(rowSelection)[0];

  const selectedRow = useMemo(() => {
    if (isLoading && !selectedRowKey) return;
    return table
      .getCoreRowModel()
      .flatRows.find((row) => row.id === selectedRowKey);
  }, [selectedRowKey, isLoading]);

  const index = table
    .getCoreRowModel()
    .flatRows.findIndex((row) => row.id === selectedRow?.id);

  const nextId = useMemo(
    () => table.getCoreRowModel().flatRows[index + 1]?.id,
    [index, isLoading],
  );

  const prevId = useMemo(
    () => table.getCoreRowModel().flatRows[index - 1]?.id,
    [index, isLoading],
  );

  const onPrev = useCallback(() => {
    if (prevId) table.setRowSelection({ [prevId]: true });
  }, [prevId, isLoading]);

  const onNext = useCallback(() => {
    if (nextId) table.setRowSelection({ [nextId]: true });
  }, [nextId, isLoading, table]);

  useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (!selectedRowKey) return;

      // REMINDER: prevent dropdown navigation inside of sheet to change row selection
      const activeElement = document.activeElement;
      const isMenuActive = activeElement?.closest('[role="menu"]');

      if (isMenuActive) return;

      if (e.key === "ArrowUp") {
        e.preventDefault();
        onPrev();
      }
      if (e.key === "ArrowDown") {
        e.preventDefault();
        onNext();
      }
    };

    document.addEventListener("keydown", down);
    return () => document.removeEventListener("keydown", down);
  }, [selectedRowKey, onNext, onPrev]);

  const handleExport = useCallback(() => {
    if (!currentRecord) return;

    // Create a JSON blob and download it
    const jsonStr = JSON.stringify(currentRecord, null, 2);
    const blob = new Blob([jsonStr], { type: "application/json" });
    const url = URL.createObjectURL(blob);

    const a = document.createElement("a");
    a.href = url;
    a.download = `audit-log-${currentRecord.id}.json`;
    document.body.appendChild(a);
    a.click();

    // Clean up
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }, [currentRecord]);

  return (
    <Sheet
      open={!!selectedRowKey}
      onOpenChange={(open) => {
        if (!open) {
          const el = selectedRowKey
            ? document.getElementById(selectedRowKey)
            : null;
          table.resetRowSelection();

          setTimeout(() => el?.focus(), 0);
        }
      }}
    >
      <SheetContent className="flex flex-col sm:max-w-xl" withClose={false}>
        <VisuallyHidden>
          <SheetHeader>
            <SheetTitle>Audit Log Details</SheetTitle>
            <SheetDescription>Audit log details</SheetDescription>
          </SheetHeader>
        </VisuallyHidden>
        <div className="size-full pt-4">
          <div className="flex items-center justify-between px-4">
            <HeaderBackButton onBack={() => table.resetRowSelection()} />
            <div className="flex h-7 items-center gap-1">
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      size="icon"
                      variant="ghost"
                      className="h-7 w-7"
                      disabled={!prevId}
                      onClick={onPrev}
                    >
                      <Icon icon={faChevronUp} className="h-5 w-5" />
                      <span className="sr-only">Previous</span>
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>
                    <p>
                      Navigate <Kbd>↑</Kbd>
                    </p>
                  </TooltipContent>
                </Tooltip>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      size="icon"
                      variant="ghost"
                      className="h-7 w-7"
                      disabled={!nextId}
                      onClick={onNext}
                    >
                      <Icon icon={faChevronDown} className="h-5 w-5" />
                      <span className="sr-only">Next</span>
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>
                    <p>
                      Navigate <Kbd>↓</Kbd>
                    </p>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
              <Separator orientation="vertical" className="mx-1" />
              <AuditActions onExport={handleExport} />
            </div>
          </div>
          <div className="mt-4 flex flex-col gap-2">
            <MemoizedAuditDetailsHeader entry={currentRecord} />
            <MemoizedAuditLogDetailsContent entry={currentRecord} />
          </div>
        </div>
      </SheetContent>
    </Sheet>
  );
}

function AuditLogDetailsContent({ entry }: { entry?: AuditEntry }) {
  if (!entry) {
    return null;
  }

  return (
    <ScrollArea className="flex max-h-[calc(100vh-8.5rem)] flex-col px-4 [&_[data-slot=scroll-area-viewport]>div]:block!">
      <div className="w-full max-w-full space-y-6 overflow-hidden pb-8">
        <div className="w-full max-w-full overflow-hidden">
          <AuditLogDetails entry={entry} />
        </div>

        <div className="w-full max-w-full overflow-hidden">
          <ChangesTable changes={entry.changes} />
        </div>

        <AuditLogDetailsSection
          title="Metadata"
          description="Additional contextual information"
        >
          <div className="w-full max-w-full overflow-hidden">
            <JsonViewer data={entry.metadata} />
          </div>
        </AuditLogDetailsSection>

        <AuditLogDetailsSection
          title="Previous State"
          description="State before the operation"
        >
          <div className="w-full max-w-full overflow-hidden">
            <JsonViewer data={entry.previousState} />
          </div>
        </AuditLogDetailsSection>

        <AuditLogDetailsSection
          title="Current State"
          description="State after the operation"
        >
          <div className="w-full max-w-full overflow-hidden">
            <JsonViewer data={entry.currentState} />
          </div>
        </AuditLogDetailsSection>

        <AuditLogDetailsSection
          title="Full Event Data"
          description="Complete raw data"
        >
          <div className="w-full max-w-full overflow-hidden">
            <JsonViewer data={entry} />
          </div>
        </AuditLogDetailsSection>
      </div>
    </ScrollArea>
  );
}

function AuditLogDetailsSection({
  children,
  title,
  description,
}: {
  children: React.ReactNode;
  title: string;
  description: string;
}) {
  return (
    <div className="flex w-full max-w-full flex-col gap-2 overflow-hidden">
      <div className="flex flex-col">
        <h3 className="text-sm font-normal">{title}</h3>
        <p className="text-2xs text-muted-foreground">{description}</p>
      </div>
      <div className="w-full max-w-full overflow-hidden">{children}</div>
    </div>
  );
}

const MemoizedAuditLogDetailsContent = memo(
  AuditLogDetailsContent,
  (prev, next) => {
    return prev.entry === next.entry;
  },
) as typeof AuditLogDetailsContent;

function AuditDetailsHeader({ entry }: { entry?: AuditEntry }) {
  const user = useUser();

  if (!entry) {
    return null;
  }

  const { timestamp, comment, operation } = entry;

  return (
    <div className="border-bg-sidebar-border flex flex-col border-b px-4 pb-2">
      <div className="flex items-center justify-between">
        <h2 className="flex items-center gap-x-2 leading-none font-semibold tracking-tight">
          {comment || "-"}
        </h2>
        <ActionBadge operation={operation} />
      </div>
      <p className="text-2xs font-normal text-muted-foreground">
        Entry created on{" "}
        {formatToUserTimezone(timestamp, {
          timeFormat: user?.timeFormat,
        })}
      </p>
    </div>
  );
}

const MemoizedAuditDetailsHeader = memo(AuditDetailsHeader, (prev, next) => {
  return prev.entry === next.entry;
}) as typeof AuditDetailsHeader;
