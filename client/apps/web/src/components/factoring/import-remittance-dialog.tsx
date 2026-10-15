import { FactoringCompanyAutocompleteField } from "@/components/autocomplete-fields";
import { DocumentUploadZone } from "@/components/documents/document-upload-zone";
import { DateField } from "@/components/fields/date-field/date-field";
import { SelectField } from "@/components/fields/select-field";
import { recordPath } from "@/config/record-links";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { RemittanceUploader } from "@/components/factoring/remittance-uploader";
import {
  FACTORING_REMITTANCE_FILE_TYPES,
  FACTORING_REMITTANCE_LINE_KIND_LABELS,
  FACTORING_REMITTANCE_MAX_BYTES,
  FACTORING_REMITTANCE_REPORT_KINDS,
} from "@/lib/factoring-remittance";
import {
  createFactoringRemittance,
  stageFactoringRemittance,
  type FactoringRemittance,
} from "@/lib/graphql/factoring-remittance";
import { invalidateFactoringQueries } from "@/lib/queries/factoring-roots";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { AlertCircleIcon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { translate } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import type { Document } from "@trenova/shared/types/document";
import { useCallback, useMemo, useState } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { useNavigate } from "react-router";
import { z } from "zod";

const importRemittanceSchema = z.object({
  factoringCompanyId: z
    .string()
    .min(1, { error: () => translate("Choose the factor that sent the report") }),
  defaultKind: z.enum(["Funding", "ReserveRelease", "Rejection", "Chargeback"]),
  remittedOn: z.number().int().positive().nullable(),
});

type ImportRemittanceValues = z.infer<typeof importRemittanceSchema>;

const DEFAULTS: ImportRemittanceValues = {
  factoringCompanyId: "",
  defaultKind: "Funding",
  remittedOn: null,
};

type ImportRemittanceDialogProps = {
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
  readonly factoringCompanyId?: string;
};

export function ImportRemittanceDialog({
  open,
  onOpenChange,
  factoringCompanyId,
}: ImportRemittanceDialogProps) {
  if (!open) return null;
  return <ImportSession onOpenChange={onOpenChange} factoringCompanyId={factoringCompanyId} />;
}

function ImportSession({
  onOpenChange,
  factoringCompanyId,
}: Omit<ImportRemittanceDialogProps, "open">) {
  const t = useT();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [remittance, setRemittance] = useState<FactoringRemittance | undefined>();
  const [pendingFile, setPendingFile] = useState<File | null>(null);
  const [problem, setProblem] = useState<string | null>(null);

  const form = useForm<ImportRemittanceValues>({
    resolver: zodResolver(importRemittanceSchema) as Resolver<ImportRemittanceValues>,
    defaultValues: { ...DEFAULTS, factoringCompanyId: factoringCompanyId ?? "" },
  });

  const kindOptions = useMemo(
    () =>
      FACTORING_REMITTANCE_REPORT_KINDS.map((kind) => ({
        value: kind,
        label: FACTORING_REMITTANCE_LINE_KIND_LABELS[kind],
      })),
    [],
  );

  const openRemittance = useCallback(
    (staged: FactoringRemittance) => {
      invalidateFactoringQueries(queryClient);
      onOpenChange(false);
      void navigate(recordPath("factoring_remittance", staged.id));
    },
    [navigate, onOpenChange, queryClient],
  );

  const { mutate: create, isPending: creating } = useMutation({
    mutationFn: (values: ImportRemittanceValues) =>
      createFactoringRemittance({
        factoringCompanyId: values.factoringCompanyId,
        defaultKind: values.defaultKind,
        remittedOn: values.remittedOn,
      }),
    onSuccess: (created) => setRemittance(created),
    onError: (error) => {
      setPendingFile(null);
      handleMutationError({ error, form, resourceName: "Factor remittance" });
    },
  });

  const { mutate: stage, isPending: staging } = useMutation({
    mutationFn: ({ current, document }: { current: FactoringRemittance; document: Document }) =>
      stageFactoringRemittance({
        id: current.id,
        version: current.version,
        documentId: document.id,
      }),
    onSuccess: openRemittance,
    onError: (error) => {
      setPendingFile(null);
      setProblem(error instanceof Error ? error.message : t("The remittance could not be read"));
      if (remittance) invalidateFactoringQueries(queryClient);
    },
  });

  const handleFile = useCallback(
    async (file: File) => {
      const valid = await form.trigger();
      if (!valid) return;
      setProblem(null);
      setPendingFile(file);
      create(form.getValues());
    },
    [create, form],
  );

  const handleUploaded = useCallback(
    (document: Document) => {
      if (!remittance) return;
      stage({ current: remittance, document });
    },
    [remittance, stage],
  );

  const handleUploadFailed = useCallback((message: string) => {
    setPendingFile(null);
    setProblem(message);
  }, []);

  const uploading = Boolean(remittance && pendingFile);
  const busy = creating || staging || uploading;

  return (
    <Dialog
      open
      onOpenChange={(next) => {
        if (!next && !busy) onOpenChange(false);
      }}
    >
      <DialogContent size="lg">
        <DialogHeader>
          <DialogTitle>{t("Import remittance")}</DialogTitle>
          <DialogDescription>
            {t(
              "Upload a factor's funding or reserve report as a CSV, spreadsheet or PDF. Its lines are matched to the factor's schedules for you to review before anything is booked.",
            )}
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-4">
          <FormProvider {...form}>
            <Form onSubmit={(event) => event.preventDefault()}>
              <FormGroup cols={2}>
                <FormControl cols="full">
                  <FactoringCompanyAutocompleteField<ImportRemittanceValues>
                    control={form.control}
                    name="factoringCompanyId"
                    label={t("Factor")}
                    placeholder={t("Select the factor")}
                    rules={{ required: true }}
                    disabled={busy || Boolean(factoringCompanyId)}
                  />
                </FormControl>
                <FormControl>
                  <SelectField
                    control={form.control}
                    name="defaultKind"
                    label={t("Report of")}
                    options={kindOptions}
                    rules={{ required: true }}
                    isReadOnly={busy}
                    description={t("What a line records when the file has no column saying so.")}
                  />
                </FormControl>
                <FormControl>
                  <DateField
                    control={form.control}
                    name="remittedOn"
                    label={t("Remitted on")}
                    placeholder={t("The report's date")}
                    clearable
                    disabled={busy}
                  />
                </FormControl>
              </FormGroup>
            </Form>
          </FormProvider>

          {uploading && remittance && pendingFile ? (
            <RemittanceUploader
              remittance={remittance}
              file={pendingFile}
              onUploaded={handleUploaded}
              onFailed={handleUploadFailed}
            />
          ) : (
            <DocumentUploadZone
              accept={FACTORING_REMITTANCE_FILE_TYPES}
              maxSize={FACTORING_REMITTANCE_MAX_BYTES}
              disabled={busy}
              onFilesSelected={(files) => {
                const file = files[0];
                if (file) void handleFile(file);
              }}
              onFilesRejected={() => setProblem(t("That file is too large; the limit is 32 MiB."))}
              className="min-h-32"
            />
          )}

          {problem ? (
            <Alert variant="destructive" size="sm">
              <AlertCircleIcon className="size-4" />
              <AlertTitle>{t("The remittance could not be read")}</AlertTitle>
              <AlertDescription>{problem}</AlertDescription>
            </Alert>
          ) : null}
        </div>
      </DialogContent>
    </Dialog>
  );
}
