import { WorkerCredentialTypeAutocompleteField } from "@/components/autocomplete-fields";
import { FileDropField, type AttachedFile } from "@/components/documents/file-drop-field";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { TextareaField } from "@/components/fields/textarea-field";
import {
  WorkPane,
  WorkPaneBody,
  WorkPaneCancel,
  WorkPaneDirtyTracker,
  WorkPaneFooter,
} from "@/components/work-pane";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useProtectedFieldAccess } from "@/hooks/use-protected-fields";
import { useSelectOption } from "@/hooks/use-select-option";
import {
  createWorkerCredential,
  updateWorkerCredential,
  type WorkerCredentialRow,
} from "@/lib/graphql/worker-credential";
import {
  selectOptionMetaBoolean,
  selectOptionMetaNumber,
  selectOptionMetaString,
} from "@/lib/select-option-meta";
import { zodResolver } from "@hookform/resolvers/zod";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { suggestExpiryUnix } from "@trenova/shared/lib/credential";
import { formatUnixDateMedium, getTodayDate } from "@trenova/shared/lib/date";
import {
  credentialFormSchema,
  type CredentialFormValues,
} from "@trenova/shared/types/worker-credential";
import { useEffect, useMemo, useRef, useState } from "react";
import { Controller, FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { healthLabel, healthTone } from "./credential-model";
import { useCredentialInvalidation } from "./use-credential-invalidation";

export type CredentialFormMode = "create" | "renew" | "edit";

/** The form's values: the schema's, plus the document the drop field holds. */
type CredentialPaneValues = CredentialFormValues & { document: AttachedFile | null };

export type CredentialFormPaneProps = {
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
    Partial<Pick<WorkerCredentialRow, "document" | "credentialType" | "health">>;
  credentialTypeId?: string | null;
  onSaved?: (credential: WorkerCredentialRow) => void;
};

function attachedDocumentOf(
  credential: CredentialFormPaneProps["credential"],
  fallbackName: string,
): AttachedFile | null {
  return credential?.documentId
    ? {
        id: credential.documentId,
        name: credential.document?.originalName ?? fallbackName,
        size: credential.document?.fileSize ?? 0,
        fileType: credential.document?.fileType ?? null,
        previewStatus: credential.document?.previewStatus ?? null,
      }
    : null;
}

function buildDefaults(
  mode: CredentialFormMode,
  credential: CredentialFormPaneProps["credential"],
  credentialTypeId: string | null | undefined,
  today: number,
  documentName: string,
): CredentialPaneValues {
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
      document: attachedDocumentOf(credential, documentName),
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
    document: null,
  };
}

/**
 * Add, renew or edit a credential in the work pane beside the table. The type decides what must
 * be filled in and suggests the expiry from its validity until the person changes it; the
 * document is uploaded to the worker's files as it is dropped.
 */
export function CredentialFormPane({
  open,
  onOpenChange,
  workerId,
  mode,
  credential,
  credentialTypeId,
  onSaved,
}: CredentialFormPaneProps) {
  const t = useT();
  const invalidate = useCredentialInvalidation(workerId);
  const today = useMemo(() => getTodayDate(), []);
  const isEdit = mode === "edit";
  const personalDetailsVisible = useProtectedFieldAccess("Personal");
  const numberWithheld = isEdit && !personalDetailsVisible;
  const documentName = t("Attached document");

  const form = useForm<CredentialPaneValues>({
    resolver: zodResolver(credentialFormSchema) as unknown as Resolver<CredentialPaneValues>,
    defaultValues: buildDefaults(mode, credential, credentialTypeId, today, documentName),
  });
  const { control, handleSubmit, reset, setValue, getFieldState, getValues } = form;

  const loadedFor = useRef<string | null>(null);
  useEffect(() => {
    const key = open ? `${mode}:${credential?.id ?? ""}:${credentialTypeId ?? ""}` : null;
    if (key === loadedFor.current) return;
    loadedFor.current = key;
    if (key) reset(buildDefaults(mode, credential, credentialTypeId, today, documentName));
  }, [open, mode, credential, credentialTypeId, today, documentName, reset]);

  const selectedTypeId = useWatch({ control, name: "credentialTypeId" });
  const issuedAt = useWatch({ control, name: "issuedAt" });
  const { option: selectedType } = useSelectOption("WORKER_CREDENTIAL_TYPE", selectedTypeId);
  const requiresNumber =
    !numberWithheld && selectedType
      ? selectOptionMetaBoolean(selectedType, "requiresNumber")
      : false;
  const requiresDocument = selectedType
    ? selectOptionMetaBoolean(selectedType, "requiresDocument")
    : false;
  const requiresExpiry = selectedType
    ? selectOptionMetaString(selectedType, "profileField") === "LicenseExpiry"
    : false;
  const validityMonths = selectedType
    ? selectOptionMetaNumber(selectedType, "validityMonths")
    : null;

  useEffect(() => {
    setValue("requiresNumber", requiresNumber);
    setValue("requiresExpiry", requiresExpiry);
  }, [requiresNumber, requiresExpiry, setValue]);

  useEffect(() => {
    if (isEdit || !selectedType) return;
    if (getFieldState("expiresAt").isDirty) return;
    setValue("expiresAt", suggestExpiryUnix(issuedAt, validityMonths), { shouldDirty: false });
  }, [isEdit, selectedType, validityMonths, issuedAt, getFieldState, setValue]);

  const [uploading, setUploading] = useState(false);
  const { mutateAsync, isPending } = useApiMutation<
    WorkerCredentialRow,
    CredentialFormValues,
    unknown,
    CredentialPaneValues
  >({
    form,
    resourceName: "Credential",
    mutationFn: async (values) => {
      const documentId = getValues("document")?.id ?? null;
      if (isEdit && credential) {
        return updateWorkerCredential({
          id: credential.id,
          number: values.number,
          issuingAuthority: values.issuingAuthority,
          issuedAt: values.issuedAt,
          expiresAt: values.expiresAt,
          documentId,
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
        documentId,
        notes: values.notes,
        renew: mode === "renew",
      });
    },
    onSuccess: (saved) => {
      reset(getValues());
      void invalidate();
      onSaved?.(saved);
      onOpenChange(false);
    },
  });

  const lockType = mode !== "create" || Boolean(credentialTypeId);
  const title =
    mode === "create"
      ? t("Add credential")
      : mode === "renew"
        ? t("Renew credential")
        : t("Edit credential");
  const description =
    mode === "create"
      ? t("It joins the file and the expiry runway as soon as you save.")
      : mode === "renew"
        ? t("The current one moves to Earlier versions when you save.")
        : t("Changing the number, dates or document clears any verification on this credential.");
  const submit =
    mode === "create"
      ? t("Add credential")
      : mode === "renew"
        ? t("Save renewal")
        : t("Save changes");

  return (
    <WorkPane open={open} onOpenChange={onOpenChange} title={title} description={description}>
      <WorkPaneDirtyTracker form={form} />
      <FormProvider {...form}>
        <Form
          onSubmit={(event) => {
            event.preventDefault();
            event.stopPropagation();
            void handleSubmit(async (values) => {
              await mutateAsync(values);
            })(event);
          }}
        >
          <WorkPaneBody>
            {mode === "renew" && credential ? <CurrentCredential credential={credential} /> : null}
            <FormGroup cols={2}>
              <FormControl cols="full">
                <WorkerCredentialTypeAutocompleteField<CredentialPaneValues>
                  control={control}
                  name="credentialTypeId"
                  label={t("Credential type")}
                  required
                  placeholder={t("Choose a credential type")}
                  disabled={lockType}
                  description={
                    validityMonths
                      ? t("Typically valid for {0} months.", String(validityMonths))
                      : t("Drives the expiry warnings and the compliance grade.")
                  }
                />
              </FormControl>
              <FormControl>
                <InputField<CredentialPaneValues>
                  control={control}
                  name="number"
                  label={t("Number")}
                  placeholder={numberWithheld ? t("Hidden") : t("e.g. 12345678")}
                  description={
                    numberWithheld ? t("Your role does not include personal details.") : undefined
                  }
                  required={requiresNumber}
                  disabled={numberWithheld}
                />
              </FormControl>
              <FormControl>
                <InputField<CredentialPaneValues>
                  control={control}
                  name="issuingAuthority"
                  label={t("Issuing authority")}
                  placeholder={t("e.g. IL SOS, FMCSA examiner")}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<CredentialPaneValues>
                  control={control}
                  name="issuedAt"
                  label={t("Issued")}
                  placeholder={t("Date on the card")}
                  description={t("Used with the type's validity to suggest an expiry.")}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<CredentialPaneValues>
                  control={control}
                  name="expiresAt"
                  label={t("Expires")}
                  placeholder={t("Leave empty if it never expires")}
                  description={
                    isEdit ? undefined : t("Suggested from the type until you change it.")
                  }
                  required={requiresExpiry}
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField<CredentialPaneValues>
                  control={control}
                  name="notes"
                  label={t("Notes")}
                  placeholder={t("e.g. Class A, no air-brake restriction")}
                  maxLength={2000}
                />
              </FormControl>
              <FormControl cols="full">
                <Controller
                  control={control}
                  name="document"
                  render={({ field }) => (
                    <FileDropField
                      label={t("Document")}
                      note={
                        requiresDocument ? t("needed before this credential can be verified") : null
                      }
                      value={field.value}
                      onChange={field.onChange}
                      resourceId={workerId}
                      resourceType="worker"
                      prompt={mode === "renew" ? t("Drop the new document here") : undefined}
                      onUploadingChange={setUploading}
                    />
                  )}
                />
              </FormControl>
            </FormGroup>
          </WorkPaneBody>
          <WorkPaneFooter>
            <WorkPaneCancel />
            <Button type="submit" size="sm" isLoading={isPending} disabled={uploading}>
              {submit}
            </Button>
          </WorkPaneFooter>
        </Form>
      </FormProvider>
    </WorkPane>
  );
}

/** The credential being renewed, as a reminder of what the new one replaces. */
function CurrentCredential({
  credential,
}: {
  credential: NonNullable<CredentialFormPaneProps["credential"]>;
}) {
  const t = useT();
  const span =
    credential.issuedAt && credential.expiresAt
      ? t(
          "{0} to {1}",
          formatUnixDateMedium(credential.issuedAt),
          formatUnixDateMedium(credential.expiresAt),
        )
      : credential.expiresAt
        ? t("Until {0}", formatUnixDateMedium(credential.expiresAt))
        : null;
  return (
    <div className="grid grid-cols-[minmax(0,1fr)_auto] gap-x-3 gap-y-0.5 rounded-md border border-border-subtle bg-sunken px-2.5 py-2">
      <span className="truncate text-sm font-medium">
        {credential.credentialType?.name ?? t("Current credential")}
      </span>
      {credential.health ? (
        <Badge variant={healthTone(credential.health)} dot>
          {healthLabel(t, credential.health)}
        </Badge>
      ) : (
        <span />
      )}
      <span className="col-span-2 truncate text-xs text-muted-foreground tabular-nums">
        {[credential.number, span, credential.document?.originalName].filter(Boolean).join(" · ")}
      </span>
    </div>
  );
}
