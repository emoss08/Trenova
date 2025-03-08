import { ShikiJsonViewer } from "@/components/ui/json-viewer";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { VisuallyHidden } from "@/components/ui/visually-hidden";
import { AuditEntry } from "@/types/audit-entry";
import { EditTableSheetProps } from "@/types/data-table";
import AuditDetailsHeader from "./audit-details-header";
import { ChangesTable, DataSection } from "./audit-log-data-section";
import { AuditLogDetails } from "./audit-log-details";
import { AuditLogHeader } from "./audit-log-header";

export function AuditLogDetailsSheet({
  currentRecord,
  open,
  onOpenChange,
}: EditTableSheetProps<AuditEntry>) {
  const handleExport = () => {
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
  };

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="flex flex-col sm:max-w-2xl" withClose={false}>
        <VisuallyHidden>
          <SheetHeader>
            <SheetTitle>Audit Log Details</SheetTitle>
            <SheetDescription>Audit log details</SheetDescription>
          </SheetHeader>
        </VisuallyHidden>
        <div className="size-full">
          <div className="pt-4">
            <AuditLogHeader
              onBack={() => onOpenChange(false)}
              onExport={handleExport}
            />
            <div className="flex flex-col gap-2 mt-4">
              <AuditDetailsHeader entry={currentRecord} />
              <AuditLogDetailsContent entry={currentRecord} />
            </div>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  );
}

export function AuditLogDetailsContent({ entry }: { entry?: AuditEntry }) {
  if (!entry) {
    return null;
  }

  return (
    <ScrollArea className="flex flex-col overflow-y-auto max-h-[calc(100vh-8.5rem)] px-4">
      <div className="space-y-6 pb-8">
        <AuditLogDetails entry={entry} />

        <ChangesTable changes={entry.changes} />

        <DataSection
          title="Metadata"
          description="Additional contextual information"
        >
          <ShikiJsonViewer data={entry.metadata} />
        </DataSection>

        <DataSection
          title="Previous State"
          description="State before the operation"
        >
          <ShikiJsonViewer data={entry.previousState} />
        </DataSection>

        <DataSection
          title="Current State"
          description="State after the operation"
        >
          <ShikiJsonViewer data={entry.currentState} />
        </DataSection>

        <DataSection
          title="Full Event Data"
          description="Complete raw data"
          defaultCollapsed
        >
          <ShikiJsonViewer data={entry} />
        </DataSection>
      </div>
    </ScrollArea>
  );
}
