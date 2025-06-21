import { DataTable } from "@/components/data-table/data-table";
import { PDFViewerDialog } from "@/components/pdf-viewer/pdf-viewer-dialog";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { Resource } from "@/types/audit-entry";
import { Document } from "@/types/document";
import { API_ENDPOINTS } from "@/types/server";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { getColumns } from "./shipment-document-columns";

export default function ShipmentDocumentTable({
  shipmentId,
}: {
  shipmentId: ShipmentSchema["id"];
}) {
  const [selectedDocument, setSelectedDocument] = useState<Document | null>(
    null,
  );
  const [pdfViewerOpen, setPdfViewerOpen] = useState(false);

  const handleDocumentClick = useCallback(
    (doc: Document) => {
      if (doc.fileType === ".pdf") {
        setSelectedDocument(doc);
        setPdfViewerOpen(true);
      } else {
        toast.info("Cannot open document as it is not a PDF");
      }
    },
    [setSelectedDocument, setPdfViewerOpen],
  );

  const columns = useMemo(
    () => getColumns({ handleDocumentClick }),
    [handleDocumentClick],
  );

  return (
    <>
      <DataTable<Document>
        name="Shipment Document"
        link={`/documents/shipment/${shipmentId}` as API_ENDPOINTS}
        extraSearchParams={{
          expandShipmentDetails: true,
        }}
        queryKey="shipment-document-list"
        exportModelName="shipment-document"
        //   TableModal={ShipmentCreateSheet}
        //   TableEditModal={ShipmentEditSheet}
        columns={columns}
        // includeHeader={false}
        includeOptions={false}
        resource={Resource.Document}
      />
      {pdfViewerOpen && selectedDocument && (
        <PDFViewerDialog
          fileUrl={selectedDocument?.presignedUrl ?? ""}
          open={pdfViewerOpen}
          onOpenChange={setPdfViewerOpen}
        />
      )}
    </>
  );
}
