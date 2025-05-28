import { DataTable } from "@/components/data-table/data-table";
import { PDFViewerDialog } from "@/components/pdf-viewer/pdf-viewer-dialog";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { Resource } from "@/types/audit-entry";
import { Document } from "@/types/document";
import { API_ENDPOINTS } from "@/types/server";
import { useCallback, useEffect, useMemo, useState } from "react";
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

  // Add debugging for the presignedURL
  useEffect(() => {
    if (selectedDocument) {
      console.log("Selected document:", selectedDocument);
      console.log("Presigned URL:", selectedDocument.presignedUrl);

      // Check if the URL is valid
      if (!selectedDocument.presignedUrl) {
        console.error("No presigned URL available for this document");
      }
    }
  }, [selectedDocument]);

  const handleDocumentClick = useCallback(
    (doc: Document) => {
      if (doc.fileType === ".pdf") {
        console.log("Opening PDF document:", doc.fileName);
        setSelectedDocument(doc);
        setPdfViewerOpen(true);
      } else {
        console.log("Document not a pdf", doc);
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
