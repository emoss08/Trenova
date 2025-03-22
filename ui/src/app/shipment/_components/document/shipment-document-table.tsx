import { DataTable } from "@/components/data-table/data-table";
import { PDFViewerDialog } from "@/components/pdf-viewer/pdf-viewer-dialog";
import { Document } from "@/types/document";
import { API_ENDPOINTS } from "@/types/server";
import { Shipment } from "@/types/shipment";
import { useCallback, useMemo, useState } from "react";
import { getColumns } from "./shipment-document-columns";

export default function ShipmentDocumentTable({
  shipmentId,
}: {
  shipmentId: Shipment["id"];
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
      />
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
