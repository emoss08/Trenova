import { DocumentUploadZone } from "@/components/documents/document-upload-zone";
import { SectionPanel } from "@/components/section-panel";
import {
  stageFactoringRemittance,
  type FactoringRemittance,
} from "@/lib/graphql/factoring-remittance";
import { invalidateFactoringQueries } from "@/lib/queries/factoring";
import { RemittanceUploader } from "@/routes/factoring/_components/import-remittance-dialog";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { useT } from "@trenova/shared/i18n/use-t";
import { describeError } from "@trenova/shared/lib/error-presentation";
import type { Document } from "@trenova/shared/types/document";
import { useState } from "react";

const REMITTANCE_FILE_TYPES = ".csv,.txt,.xlsx,.pdf";
const REMITTANCE_MAX_BYTES = 32 * 1024 * 1024;

/** Reads a remittance again from a new file, after its first could not be read. */
export function RemittanceReupload({ remittance }: { remittance: FactoringRemittance }) {
  const t = useT();
  const queryClient = useQueryClient();
  const [file, setFile] = useState<File | null>(null);
  const [problem, setProblem] = useState<string | null>(null);

  const stage = useMutation({
    mutationFn: (document: Document) =>
      stageFactoringRemittance({
        id: remittance.id,
        version: remittance.version,
        documentId: document.id,
      }),
    onSuccess: () => {
      setFile(null);
      invalidateFactoringQueries(queryClient);
    },
    onError: (error) => {
      setFile(null);
      const described = describeError(error);
      setProblem(described.detail ?? described.message);
    },
  });

  return (
    <SectionPanel title={t("Upload the file again")}>
      <div className="flex flex-col gap-3 p-3">
        {file ? (
          <RemittanceUploader
            remittance={remittance}
            file={file}
            onUploaded={(document) => stage.mutate(document)}
            onFailed={(message) => {
              setFile(null);
              setProblem(message);
            }}
          />
        ) : (
          <DocumentUploadZone
            accept={REMITTANCE_FILE_TYPES}
            maxSize={REMITTANCE_MAX_BYTES}
            disabled={stage.isPending}
            onFilesSelected={(files) => {
              const next = files[0];
              if (next) {
                setProblem(null);
                setFile(next);
              }
            }}
            onFilesRejected={() => setProblem(t("That file is too large; the limit is 32 MiB."))}
            className="min-h-28"
          />
        )}
        {problem ? (
          <Alert size="sm" variant="destructive">
            <AlertDescription>{problem}</AlertDescription>
          </Alert>
        ) : null}
      </div>
    </SectionPanel>
  );
}
