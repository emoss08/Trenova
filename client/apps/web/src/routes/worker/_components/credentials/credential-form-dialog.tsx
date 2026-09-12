import { useT } from "@trenova/shared/i18n/use-t";
import { DocumentUploadZone } from "@/components/documents/document-upload-zone";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useDocumentUpload } from "@/hooks/use-document-upload";
import {
  createWorkerCredential,
  fetchActiveWorkerCredentialTypes,
  updateWorkerCredential,
  WORKER_CREDENTIAL_TYPES_KEY,
  type WorkerCredentialRow,
  type WorkerCredentialTypeRow,
} from "@/lib/graphql/worker-credential";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { suggestExpiryUnix } from "@trenova/shared/lib/credential";
import { getTodayDate } from "@trenova/shared/lib/date";
import { formatFileSize } from "@trenova/shared/lib/utils";
import {
  credentialFormSchema,
  type CredentialFormValues,
} from "@trenova/shared/types/worker-credential";
import { CheckCircle2Icon, PaperclipIcon, XIcon } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { useCredentialInvalidation } from "./use-credential-invalidation";

export type CredentialFormMode = "create" | "renew" | "edit";

export type CredentialFormDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  mode: CredentialFormMode;
  credential?: Pick<
    WorkerCredentialRow,
    | "id"
    | "credentialTypeId"
    | "number"
    | "issuingAuthority"
    | "issuedAt"
    | "expiresAt"
    | "documentId"
    | "notes"
    | "version"
  > &
    Partial<Pick<WorkerCredentialRow, "document">>;
  credentialTypeId?: string | null;
  onSaved?: (credential: WorkerCredentialRow) => void;
};

type AttachedDocument = { id: string; name: string; size: number };

const COPY: Record<CredentialFormMode, { title: string; description: string; submit: string }> = {
  create: {
    title: "Add credential",
    description:
      "Record a licence, card, endorsement or certificate. Attach a scan so it can be verified.",
    submit: "Add credential",
  },
  renew: {
    title: "Renew credential",
    description:
      "Enter the new card's details. The previous one is archived and kept in the worker's history.",
    submit: "Renew credential",
  },
  edit: {
    title: "Edit credential",
    description:
      "Changing the number, dates or document clears any verification on this credential.",
    submit: "Save changes",
  },
};

function buildDefaults(
  mode: CredentialFormMode,
  credential: CredentialFormDialogProps["credential"],
  credentialTypeId: string | null | undefined,
  today: number,
): CredentialFormValues {
  const typeId = credentialTypeId ?? credential?.credentialTypeId ?? "";
  if (mode === "edit" && credential) {
    return {
      credentialTypeId: typeId,
      number: credential.number ?? null,
      issuingAuthority: credential.issuingAuthority ?? null,
      issuedAt: credential.issuedAt ?? null,
      expiresAt: credential.expiresAt ?? null,
      notes: credential.notes ?? null,
      requiresNumber: false,
      requiresExpiry: false,
    };
  }
  return {
    credentialTypeId: typeId,
    number: mode === "renew" ? (credential?.number ?? null) : null,
    issuingAuthority: mode === "renew" ? (credential?.issuingAuthority ?? null) : null,
    issuedAt: today,
    expiresAt: null,
    notes: null,
    requiresNumber: false,
    requiresExpiry: false,
  };
}

export function CredentialFormDialog({
  open,
  onOpenChange,
  workerId,
  mode,
  credential,
  credentialTypeId,
  onSaved,
}: CredentialFormDialogProps) {
  const t = useT();

  const invalidate = useCredentialInvalidation(workerId);
  const today = useMemo(() => getTodayDate(), []);
  const isEdit = mode === "edit";
  const copy = COPY[mode];

  const { data: types = [], isLoading: typesLoading } = useQuery({
    queryKey: [WORKER_CREDENTIAL_TYPES_KEY],
    queryFn: ({ signal }) => fetchActiveWorkerCredentialTypes({ signal }),
    enabled: open,
    staleTime: 5 * 60 * 1000,
  });

  const form = useForm<CredentialFormValues>({
    resolver: zodResolver(credentialFormSchema) as Resolver<CredentialFormValues>,
    defaultValues: buildDefaults(mode, credential, credentialTypeId, today),
  });
  const { control, handleSubmit, reset, setValue, getFieldState } = form;

  const [attached, setAttached] = useState<AttachedDocument | null>(() =>
    credential?.documentId
      ? {
          id: credential.documentId,
          name: credential.document?.originalName ?? "Attached document",
          size: credential.document?.fileSize ?? 0,
        }
      : null,
  );

  useEffect(() => {
    if (!open) return;
    reset(buildDefaults(mode, credential, credentialTypeId, today));
    setAttached(
      credential?.documentId
        ? {
            id: credential.documentId,
            name: credential.document?.originalName ?? "Attached document",
            size: credential.document?.fileSize ?? 0,
          }
        : null,
    );
  }, [open, mode, credential, credentialTypeId, today, reset]);

  const selectedTypeId = useWatch({ control, name: "credentialTypeId" });
  const issuedAt = useWatch({ control, name: "issuedAt" });
  const selectedType = useMemo<WorkerCredentialTypeRow | undefined>(
    () => types.find((type) => type.id === selectedTypeId),
    [types, selectedTypeId],
  );

  useEffect(() => {
    setValue("requiresNumber", Boolean(selectedType?.requiresNumber));
    setValue("requiresExpiry", selectedType?.profileField === "LicenseExpiry");
  }, [selectedType, setValue]);

  useEffect(() => {
    if (isEdit || !selectedType) return;
    if (getFieldState("expiresAt").isDirty) return;
    setValue("expiresAt", suggestExpiryUnix(issuedAt, selectedType.validityMonths), {
      shouldDirty: false,
    });
  }, [isEdit, selectedType, issuedAt, getFieldState, setValue]);

  const { uploads, uploadFiles, cancelUpload } = useDocumentUpload({
    resourceId: workerId,
    resourceType: "worker",
    onSuccess: (document) => {
      setAttached({ id: document.id, name: document.originalName, size: document.fileSize });
    },
    onError: (error) => {
      toast.error(t("Upload failed"), { description: error.message });
    },
  });
  const activeUpload = uploads.find(
    (upload) => upload.status === "uploading" || upload.status === "pending",
  );

  const { mutateAsync, isPending } = useApiMutation<
    WorkerCredentialRow,
    CredentialFormValues,
    unknown,
    CredentialFormValues
  >({
    form,
    resourceName: "Credential",
    mutationFn: async (values) => {
      if (isEdit && credential) {
        return updateWorkerCredential({
          id: credential.id,
          number: values.number,
          issuingAuthority: values.issuingAuthority,
          issuedAt: values.issuedAt,
          expiresAt: values.expiresAt,
          documentId: attached?.id ?? null,
          notes: values.notes,
          version: credential.version,
        });
      }
      return createWorkerCredential({
        workerId,
        credentialTypeId: values.credentialTypeId,
        number: values.number,
        issuingAuthority: values.issuingAuthority,
        issuedAt: values.issuedAt,
        expiresAt: values.expiresAt,
        documentId: attached?.id ?? null,
        notes: values.notes,
        renew: mode === "renew",
      });
    },
    onSuccess: (saved) => {
      toast.success(
        mode === "edit"
          ? "Credential updated"
          : mode === "renew"
            ? "Credential renewed"
            : "Credential added",
        {
          description:
            mode === "renew"
              ? "The previous credential was archived into the worker's history."
              : "The worker's qualification file has been updated.",
        },
      );
      void invalidate();
      onSaved?.(saved);
      onOpenChange(false);
    },
  });

  const onSubmit = useCallback(
    async (values: CredentialFormValues) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  const typeOptions = useMemo(
    () =>
      types.map((type) => ({
        value: type.id,
        label: type.name,
      })),
    [types],
  );

  const lockType = mode !== "create" || Boolean(credentialTypeId);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t(copy.title)}</DialogTitle>
          <DialogDescription>{t(copy.description)}</DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            onSubmit={(event) => {
              event.preventDefault();
              event.stopPropagation();
              void handleSubmit(onSubmit)(event);
            }}
          >
            <FormGroup className="pb-2" cols={2}>
              <FormControl cols="full">
                <SelectField<CredentialFormValues>
                  control={control}
                  name="credentialTypeId"
                  label={t("Credential type")}
                  options={typeOptions}
                  rules={{ required: true }}
                  placeholder={typesLoading ? "Loading types..." : "Choose a credential type"}
                  isReadOnly={lockType || typesLoading}
                  description={
                    selectedType?.description ??
                    (selectedType?.validityMonths
                      ? `Typically valid for ${selectedType.validityMonths} months.`
                      : "Decides what must be filled in and which profile field, if any, it backs.")
                  }
                />
              </FormControl>
              <FormControl>
                <InputField<CredentialFormValues>
                  control={control}
                  name="number"
                  label={t("Number")}
                  placeholder={t("e.g. 12345678")}
                  description={t("The number printed on the card or certificate.")}
                  rules={{ required: Boolean(selectedType?.requiresNumber) }}
                />
              </FormControl>
              <FormControl>
                <InputField<CredentialFormValues>
                  control={control}
                  name="issuingAuthority"
                  label={t("Issuing authority")}
                  placeholder={t("e.g. TX DPS, FMCSA examiner")}
                  description={t("Who issued it, so a verifier knows where to check.")}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<CredentialFormValues>
                  control={control}
                  name="issuedAt"
                  label={t("Issued")}
                  placeholder={t("Date on the card")}
                  description={t("Used with the type's validity to suggest an expiry.")}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<CredentialFormValues>
                  control={control}
                  name="expiresAt"
                  label={t("Expires")}
                  placeholder={t("Leave empty if it never expires")}
                  description={t("Drives the expiry warnings and the worker's compliance grade.")}
                  rules={{ required: selectedType?.profileField === "LicenseExpiry" }}
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField<CredentialFormValues>
                  control={control}
                  name="notes"
                  label={t("Notes")}
                  placeholder={t("e.g. Class A, no air-brake restriction")}
                  description={t("Kept on the credential and shown on the worker's file.")}
                  maxLength={2000}
                />
              </FormControl>
              <FormControl cols="full">
                <div className="flex flex-col gap-2">
                  <p className="text-sm font-medium">
                    {t("Document")}
                    {selectedType?.requiresDocument ? (
                      <span className="text-muted-foreground ml-1 text-xs font-normal">
                        {t("needed before this credential can be verified")}
                      </span>
                    ) : null}
                  </p>
                  {attached ? (
                    <div className="border-border bg-muted/40 flex items-center justify-between gap-2 rounded-md border px-3 py-2 text-sm">
                      <span className="flex min-w-0 items-center gap-2">
                        <CheckCircle2Icon className="size-4 shrink-0 text-green-600 dark:text-green-400" />
                        <span className="truncate">{attached.name}</span>
                        {attached.size > 0 ? (
                          <span className="text-muted-foreground shrink-0 text-xs">
                            {formatFileSize(attached.size)}
                          </span>
                        ) : null}
                      </span>
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        className="size-7"
                        aria-label={t("Remove document")}
                        onClick={() => setAttached(null)}
                      >
                        <XIcon className="size-3.5" />
                      </Button>
                    </div>
                  ) : activeUpload ? (
                    <div className="border-border flex items-center justify-between gap-2 rounded-md border px-3 py-2 text-sm">
                      <span className="flex min-w-0 items-center gap-2">
                        <PaperclipIcon className="text-muted-foreground size-4 shrink-0 animate-pulse" />
                        <span className="truncate">
                          {t(
                            "Uploading {0}… {1}%",
                            activeUpload.file.name,
                            Math.round(activeUpload.progress),
                          )}
                        </span>
                      </span>
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => cancelUpload(activeUpload.id)}
                      >
                        {t("Cancel")}
                      </Button>
                    </div>
                  ) : (
                    <DocumentUploadZone
                      accept=".pdf,.jpg,.jpeg,.png,.webp"
                      onFilesSelected={(files) => uploadFiles(files.slice(0, 1))}
                      className="min-h-24"
                    />
                  )}
                </div>
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button type="submit" disabled={isPending || Boolean(activeUpload)}>
                {isPending ? t("Saving...") : copy.submit}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
