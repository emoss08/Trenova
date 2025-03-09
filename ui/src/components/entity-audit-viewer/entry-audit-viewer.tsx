import { AuditEntryActionBadge } from "@/app/audit-logs/_components/audit-column-components";
import { generateDateTimeStringFromUnixTimestamp } from "@/lib/date";
import { getAuditEntriesByResourceID } from "@/services/audit-entry";
import { AuditEntry } from "@/types/audit-entry";
import { TableSheetProps } from "@/types/data-table";
import { faEye } from "@fortawesome/pro-regular-svg-icons";
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
import { Icon } from "../ui/icons";
import { LazyImage } from "../ui/image";
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
          {isLoading && <ComponentLoader message="Loading audit entires..." />}
          {data && (
            <AuditEntryTable
              data={data.items}
              onOpenChange={() => onOpenChange(false)}
            />
          )}
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

function AuditEntryTable({
  data,
  onOpenChange,
}: {
  data: AuditEntry[];
  onOpenChange: () => void;
}) {
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
        <div className="flex flex-col items-center justify-center h-full text-center p-6 space-y-4">
          <LazyImage
            src="https://i.kym-cdn.com/photos/images/newsfeed/001/870/641/6c9.gif"
            alt="Awkward monkey puppet looking away meme"
            width={250}
            height={250}
          />
          <h3 className="text-xl font-semibold text-foreground animate-pulse">
            Well This Is Awkwaaaaard...
          </h3>
          <p className="text-base text-muted-foreground max-w-md">
            You caught us with our audit logs down! This record was secretly
            created by the system while nobody was looking.{" "}
            <span className="italic">*Nervous cough*</span>
          </p>
          <div className="bg-gray-100 border border-gray-200 rounded-lg p-4 max-w-md">
            <p className="text-sm text-gray-700">
              <span className="font-semibold">TECHNICALLY SPEAKING:</span> When
              records are created through backend wizardry or system gremlins,
              they don&apos;t leave an audit trail until a human touches them.
            </p>
          </div>
          <div className="flex flex-col items-center space-y-2">
            <p className="text-sm italic text-muted-foreground">
              Choose your reaction:
            </p>
            <div className="flex space-x-3">
              <Button
                variant="outline"
                className="text-sm"
                onClick={onOpenChange}
              >
                👀 Pretend I Didn&apos;t See This
              </Button>
              <Button
                variant="outline"
                className="text-sm"
                onClick={onOpenChange}
              >
                🤷‍♂️ Back to Normal Life
              </Button>
            </div>
          </div>
          <p className="text-xs text-muted-foreground mt-2">
            Don&apos;t worry, we&apos;ll start tracking changes as soon as
            someone does something. Promise. Cross our hearts. For real this
            time.
          </p>
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
