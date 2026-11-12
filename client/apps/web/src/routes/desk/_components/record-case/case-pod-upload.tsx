import { formatFileSize } from "@/components/documents/document-upload-zone";
import { UploadPanel } from "@/components/documents/upload-panel";
import type { RejectedFile } from "@/lib/document-upload-files";
import { useDocumentUpload } from "@/hooks/use-document-upload";
import { documentType } from "@/lib/queries/document-type";
import { shipment as shipmentQueries } from "@/lib/queries/shipment";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { useMemo } from "react";
import { toast } from "@trenova/shared/components/ui/toast";

/** How a proof of delivery is told from the other papers a shipment still needs. */
const PROOF_OF_DELIVERY = /\bpod\b|proof of delivery/i;

/**
 * Uploading the proof of delivery from the shipment's case. The paper is
 * filed on the shipment as the document its billing still needs, so the
 * condition that raised the case clears when it lands. It is the shipment's
 * own upload, opened here so the person does not leave the case to do it.
 */
export default function CasePodUpload({
  shipmentId,
  open,
  onClose,
  onUploaded,
}: {
  shipmentId: string;
  open: boolean;
  onClose: () => void;
  onUploaded: () => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const readiness = shipmentQueries.billingReadiness(shipmentId);
  // The organization's own document types: the proof of delivery is the one
  // its code or name says it is, whatever the shipment's billing asks for.
  const { data: types } = useQuery({
    ...documentType.options(),
    enabled: open,
    staleTime: 5 * 60 * 1000,
  });
  const wanted = useMemo(() => {
    const found = (types?.results ?? []).find(
      (option) =>
        PROOF_OF_DELIVERY.test(option.label ?? "") ||
        PROOF_OF_DELIVERY.test(option.description ?? ""),
    );

    return found ? { id: found.id, name: found.description ?? found.label ?? "" } : null;
  }, [types]);
  const uploadMetadata = useMemo(
    (): Record<string, string> => (wanted ? { documentTypeId: wanted.id } : {}),
    [wanted],
  );

  const { uploads, uploadFiles, cancelUpload, retryUpload, removeUpload, clearCompleted } =
    useDocumentUpload({
      resourceId: shipmentId,
      resourceType: "shipment",
      uploadMetadata,
      onSuccess: () => {
        void queryClient.invalidateQueries({ queryKey: readiness.queryKey });
        onUploaded();
      },
      onError: (error) => {
        toast.error(t("Upload failed: {0}", error.message));
      },
    });

  return (
    <UploadPanel
      isOpen={open}
      onClose={onClose}
      uploads={uploads}
      onFilesSelected={uploadFiles}
      onFilesRejected={(rejected: RejectedFile[]) => {
        for (const { file, reason } of rejected) {
          if (reason === "size") {
            toast.error(t("File too large: {0}", file.name), {
              description: t(
                "Maximum file size is 50MB. This file is {0}.",
                formatFileSize(file.size),
              ),
            });
          }
        }
      }}
      onCancel={cancelUpload}
      onRetry={retryUpload}
      onRemove={removeUpload}
      onClearCompleted={clearCompleted}
      title={t("Upload the proof of delivery")}
      description={
        wanted
          ? t("This upload will be classified as {0}.", wanted.name)
          : t(
              "The paper is filed on the shipment. Choose its type on the shipment if it is not read as a proof of delivery.",
            )
      }
    />
  );
}
