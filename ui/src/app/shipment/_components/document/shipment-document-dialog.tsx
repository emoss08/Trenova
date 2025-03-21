import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { PDFViewerDialog } from "@/components/pdf-viewer/pdf-viewer-dialog";
import { DocumentTypeBadge } from "@/components/status-badge";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Icon } from "@/components/ui/icons";
import { Separator } from "@/components/ui/separator";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { queries } from "@/lib/queries";
import { formatFileSize, getFileIcon } from "@/lib/utils";
import { Resource } from "@/types/audit-entry";
import { TableSheetProps } from "@/types/data-table";
import { Document } from "@/types/document";
import { Shipment } from "@/types/shipment";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";

type ShipmentDocumentDialogProps = {
  shipmentId: Shipment["id"];
} & TableSheetProps;

export function ShipmentDocumentDialog({
  shipmentId,
  ...props
}: ShipmentDocumentDialogProps) {
  return (
    <Dialog {...props}>
      <DialogContent className="max-w-[900px]">
        <DialogHeader>
          <DialogTitle>Shipment Documents</DialogTitle>
          <DialogDescription>
            View and manage documents associated with this shipment.
          </DialogDescription>
        </DialogHeader>
        <DialogBody className="p-0">
          <DocumentTable shipmentId={shipmentId} />
        </DialogBody>
      </DialogContent>
    </Dialog>
  );
}

function DocumentTable({ shipmentId }: { shipmentId: Shipment["id"] }) {
  const { data: documents, isLoading } = useQuery({
    ...queries.document.documentsByResourceID(Resource.Shipment, shipmentId),
  });

  const [selectedDocument, setSelectedDocument] = useState<Document | null>(
    null,
  );
  const [pdfViewerOpen, setPdfViewerOpen] = useState(false);

  const handleDocumentClick = (doc: Document) => {
    if (doc.fileType === ".pdf") {
      setSelectedDocument(doc);
      setPdfViewerOpen(true);
    } else {
      console.log("Document not a pdf", doc);
    }
  };

  return (
    <>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Document</TableHead>
            <TableHead>Document Class</TableHead>
            <TableHead>Description</TableHead>
            <TableHead>Uploaded By</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading ? (
            <TableRow>
              <TableCell colSpan={3}>Loading...</TableCell>
            </TableRow>
          ) : (
            documents?.results.map((doc) => (
              <TableRow key={doc.id}>
                <TableCell>
                  <DocumentTableCell
                    doc={doc}
                    onClick={() => handleDocumentClick(doc)}
                  />
                </TableCell>
                <TableCell>
                  <DocumentTypeBadge documentType={doc.documentType} />
                </TableCell>
                <TableCell>
                  <DataTableDescription description={doc.description} />
                </TableCell>
                <TableCell>{doc.uploadedBy?.name}</TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
      {pdfViewerOpen && (
        <PDFViewerDialog
          fileUrl={selectedDocument?.presignedURL ?? ""}
          open={pdfViewerOpen}
          onOpenChange={setPdfViewerOpen}
        />
      )}
    </>
  );
}

function DocumentTableCell({
  doc,
  onClick,
}: {
  doc: Document;
  onClick: () => void;
}) {
  return (
    <div
      onClick={onClick}
      className="group flex items-center gap-2 px-1 py-1.5 text-left text-sm cursor-pointer"
    >
      <div className="relative flex size-8 shrink-0 overflow-hidden rounded-sm">
        <div className="bg-muted flex size-full items-center justify-center rounded-sm">
          <Icon icon={getFileIcon(doc.fileType)} className="size-4" />
        </div>
      </div>
      <div className="grid w-full flex-1 text-left leading-tight">
        <span className="group-hover:underline text-sm font-semibold">
          {doc.fileName}
        </span>
        <div className="flex items-center gap-2">
          <span className="text-xs">{formatFileSize(doc.fileSize)}</span>
          <Separator className="h-6 w-px bg-border" orientation="vertical" />
          <span className="text-xs">{doc.fileType}</span>
        </div>
      </div>
    </div>
  );
}
