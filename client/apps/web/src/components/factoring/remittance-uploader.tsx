import { useDocumentUpload } from "@/hooks/use-document-upload";
import { isPdfRemittance } from "@/lib/factoring-remittance";
import {
  FACTORING_REMITTANCE_RESOURCE,
  type FactoringRemittance,
} from "@/lib/graphql/factoring-remittance";
import { Progress } from "@trenova/shared/components/ui/progress";
import { useT } from "@trenova/shared/i18n/use-t";
import type { Document } from "@trenova/shared/types/document";
import { useEffect, useRef } from "react";

type RemittanceUploaderProps = {
  remittance: FactoringRemittance;
  file: File;
  onUploaded: (document: Document) => void;
  onFailed: (message: string) => void;
};

export function RemittanceUploader({
  remittance,
  file,
  onUploaded,
  onFailed,
}: RemittanceUploaderProps) {
  const t = useT();
  const { uploads, uploadFiles } = useDocumentUpload({
    resourceId: remittance.id,
    resourceType: FACTORING_REMITTANCE_RESOURCE,
    processingProfile: isPdfRemittance(file.name) ? "factoring_remittance" : undefined,
    onSuccess: onUploaded,
    onError: (error) => onFailed(error.message),
  });
  const started = useRef(false);

  useEffect(() => {
    if (started.current) return;
    started.current = true;
    uploadFiles([file]);
  }, [file, uploadFiles]);

  const upload = uploads[0];
  const progress = upload?.progress ?? 0;
  return (
    <div className="flex flex-col gap-2 rounded-lg border p-3" aria-busy="true">
      <p className="text-sm font-medium">
        {upload?.status === "success"
          ? t("Reading the remittance…")
          : t("Uploading {0}…", file.name)}
      </p>
      <Progress value={progress} aria-label={t("Upload progress")} />
      <p className="text-muted-foreground text-xs">
        {t("Nothing is booked until you review the matches and apply them.")}
      </p>
    </div>
  );
}
