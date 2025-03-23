import { AuditEntryActionBadge } from "@/app/audit-logs/_components/audit-column-components";
import { generateDateTimeStringFromUnixTimestamp } from "@/lib/date";
import { getAuditEntriesByResourceID } from "@/services/audit-entry";
import { AuditEntry } from "@/types/audit-entry";
import { TableSheetProps } from "@/types/data-table";
import {
  faEye,
  faFileArchive,
  faFileCircleQuestion,
  faFileLines,
} from "@fortawesome/pro-regular-svg-icons";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { BetaTag } from "../ui/beta-tag";
import { Button } from "../ui/button";
import { ComponentLoader } from "../ui/component-loader";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "../ui/dialog";
import { EmptyState } from "../ui/empty-state";
import { Icon } from "../ui/icons";
import { JsonViewerDialog } from "../ui/json-viewer";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../ui/table";

type EntryAuditViewerProps = TableSheetProps & {
  resourceId: string;
};

export function EntryAuditViewer({
  resourceId,
  open,
  onOpenChange,
}: EntryAuditViewerProps) {
  const { data, isLoading, isError } = useQuery({
    queryKey: ["audit-entries", resourceId],
    queryFn: async () => getAuditEntriesByResourceID(resourceId),
    enabled: !!resourceId && open,
  });

  if (!open && resourceId) return null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-5xl">
        <DialogHeader>
          <DialogTitle>
            Audit Entries <BetaTag />
          </DialogTitle>
          <DialogDescription>
            View the audit log for the resource with ID: {resourceId}
          </DialogDescription>
        </DialogHeader>
        <DialogBody className="p-0">
          {isLoading && <ComponentLoader message="Loading audit entries..." />}
          {data && <AuditEntryTable data={data.items} />}
          {isError && (
            <div className="p-4">
              <p className="text-sm text-red-500">
                Error loading audit entries. Please try again later.
              </p>
            </div>
          )}
        </DialogBody>
      </DialogContent>
    </Dialog>
  );
}

function AuditEntryTable({ data }: { data: AuditEntry[] }) {
  const [jsonViewerOpen, setJsonViewerOpen] = useState<boolean>(false);
  const [selectedEntry, setSelectedEntry] = useState<AuditEntry | null>(null);

  const handleOpenJsonViewer = (entry: AuditEntry) => {
    setSelectedEntry(entry);
    setJsonViewerOpen(true);
  };

  return (
    <>
      {data.length > 0 ? (
        <Table>
          <TableHeader>
            <TableRow className="hover:bg-transparent">
              <TableHead>Action</TableHead>
              <TableHead>Modified By</TableHead>
              <TableHead>Timestamp</TableHead>
              <TableHead>Comment</TableHead>
              <TableHead>Changes</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {Object.entries(data).map(([key, value]) => {
              return (
                <TableRow key={key} className="hover:bg-transparent">
                  <TableCell>
                    <AuditEntryActionBadge
                      action={value.action}
                      withDot={false}
                    />
                  </TableCell>
                  <TableCell>{value.user?.name || "-"}</TableCell>
                  <TableCell>
                    {generateDateTimeStringFromUnixTimestamp(value.timestamp)}
                  </TableCell>
                  <TableCell>{value.comment || "-"}</TableCell>
                  <TableCell>
                    <Button
                      variant="outline"
                      onClick={() => handleOpenJsonViewer(value)}
                    >
                      <Icon icon={faEye} />
                      <span className="sr-only">View</span>
                    </Button>
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      ) : (
        <div className="flex h-full items-center justify-center p-4">
          <EmptyState
            icons={[faFileLines, faFileArchive, faFileCircleQuestion]}
            className="size-full border-none"
            title="No audit entries found"
            description="No audit entries were found for this resource."
          />
        </div>
      )}

      {selectedEntry && (
        <JsonViewerDialog
          data={selectedEntry}
          open={jsonViewerOpen}
          onOpenChange={setJsonViewerOpen}
        />
      )}
    </>
  );
}
