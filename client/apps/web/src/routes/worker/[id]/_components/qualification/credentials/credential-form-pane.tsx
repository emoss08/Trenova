import { WorkerCredentialTypeAutocompleteField } from "@/components/autocomplete-fields";
import { FileDropField, type AttachedFile } from "@/components/documents/file-drop-field";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { TextareaField } from "@/components/fields/textarea-field";
import {
  PaneFormLoading,
  WorkPane,
  WorkPaneBody,
  WorkPaneCancel,
  WorkPaneDirtyTracker,
  WorkPaneFooter,
} from "@/components/work-pane";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useProtectedFieldAccess } from "@/hooks/use-protected-fields";
import { selectOptionQueryOptions, useSelectOption } from "@/hooks/use-select-option";
import {
  createWorkerCredential,
  updateWorkerCredential,
  type WorkerCredentialRow,
} from "@/lib/graphql/worker-credential";
import type { SelectOption } from "@/lib/graphql/select-options";
import {
  selectOptionMetaBoolean,
  selectOptionMetaNumber,
  selectOptionMetaString,
} from "@/lib/select-option-meta";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { suggestExpiryUnix } from "@trenova/shared/lib/credential";
import { formatRange, formatUnixDateMedium, getTodayDate } from "@trenova/shared/lib/date";
import {
  credentialFormSchema,
  type CredentialFormValues,
} from "@trenova/shared/types/worker-credential";
import { useState } from "react";
import { Controller, FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { useWorkerInvalidation } from "../../use-worker-invalidation";
import { SubjectCard } from "../subject-card";
import { healthLabel, healthTone } from "./credential-model";

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

type TypeOption = SelectOption | null;

/** What the type a form holds asks of it: a number, an expiry. */
function credentialRequirements(
  type: TypeOption,
  numberWithheld: boolean,
): Pick<CredentialFormValues, "requiresNumber" | "requiresExpiry"> {
  return {
    requiresNumber:
      !numberWithheld && type ? selectOptionMetaBoolean(type, "requiresNumber") : false,
    requiresExpiry: type ? selectOptionMetaString(type, "profileField") === "LicenseExpiry" : false,
  };
}

function validityMonthsOf(type: TypeOption): number | null {
  return type ? selectOptionMetaNumber(type, "validityMonths") : null;
}

function buildDefaults(
  mode: CredentialFormMode,
  credential: CredentialFormPaneProps["credential"],
  credentialTypeId: string | null | undefined,
  presetType: TypeOption,
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
  const today = getTodayDate();
  return {
    credentialTypeId: typeId,
    number: mode === "renew" ? (credential?.number ?? null) : null,
    issuingAuthority: mode === "renew" ? (credential?.issuingAuthority ?? null) : null,
    issuedAt: today,
    expiresAt: presetType ? suggestExpiryUnix(today, validityMonthsOf(presetType)) : null,
    notes: null,
    requiresNumber: false,
    requiresExpiry: false,
    document: null,
  };
}

function paneCopy(t: TranslateFn, mode: CredentialFormMode) {
  switch (mode) {
    case "create":
      return {
        title: t("Add credential"),
        description: t("It joins the file and the expiry runway as soon as you save."),
        submit: t("Add credential"),
      };
    case "renew":
      return {
        title: t("Renew credential"),
        description: t("The current one moves to Earlier versions when you save."),
        submit: t("Renew credential"),
      };
    case "edit":
      return {
        title: t("Edit credential"),
        description: t(
          "Changing the number, dates or document clears any verification on this credential.",
        ),
        submit: t("Save changes"),
      };
  }
}

/**
 * Add, renew or edit a credential in the work pane beside the table. The type decides what must
 * be filled in and suggests the expiry from its validity until the person changes it; the
 * document is uploaded to the worker's files as it is dropped.
 *
 * The pane mounts fresh at each opening (`lazyWorkPane`). A new or renewed credential whose type
 * is set before it opens waits for that type, so the suggested expiry is in the form's defaults.
 */
export function CredentialFormPane(props: CredentialFormPaneProps) {
  const { open, onOpenChange, mode, credential, credentialTypeId } = props;
  const t = useT();
  const copy = paneCopy(t, mode);
  const presetTypeId = mode === "edit" ? null : (credentialTypeId ?? credential?.credentialTypeId);
  const preset = useSelectOption("WORKER_CREDENTIAL_TYPE", presetTypeId);

  return (
    <WorkPane
      open={open}
      onOpenChange={onOpenChange}
      title={copy.title}
      description={copy.description}
    >
      {preset.isLoading ? (
        <PaneFormLoading />
      ) : (
        <CredentialForm {...props} presetType={preset.option} submitLabel={copy.submit} />
      )}
    </WorkPane>
  );
}

function CredentialForm({
  onOpenChange,
  workerId,
  mode,
  credential,
  credentialTypeId,
  onSaved,
  presetType,
  submitLabel,
}: CredentialFormPaneProps & { presetType: TypeOption; submitLabel: string }) {
  const t = useT();
  const queryClient = useQueryClient();
  const invalidate = useWorkerInvalidation(workerId, "credentials");
  const isEdit = mode === "edit";
  const personalDetailsVisible = useProtectedFieldAccess("Personal");
  const numberWithheld = isEdit && !personalDetailsVisible;
  const schemaResolver = zodResolver(
    credentialFormSchema,
  ) as unknown as Resolver<CredentialPaneValues>;

  const form = useForm<CredentialPaneValues>({
    // What the type asks for is read at validation from the type the form holds then, so a
    // change of type can never leave the rule a step behind.
    resolver: async (values, context, options) => {
      const type = values.credentialTypeId
        ? await queryClient
            .fetchQuery(selectOptionQueryOptions("WORKER_CREDENTIAL_TYPE", values.credentialTypeId))
            .catch(() => null)
        : null;
      return schemaResolver(
        { ...values, ...credentialRequirements(type, numberWithheld) },
        context,
        options,
      );
    },
    defaultValues: buildDefaults(
      mode,
      credential,
      credentialTypeId,
      presetType,
      t("Attached document"),
    ),
  });
  const { control, handleSubmit, reset, setValue, getFieldState, getValues } = form;

  const selectedTypeId = useWatch({ control, name: "credentialTypeId" });
  const { option: selectedType } = useSelectOption("WORKER_CREDENTIAL_TYPE", selectedTypeId);
  const { requiresNumber, requiresExpiry } = credentialRequirements(selectedType, numberWithheld);
  const requiresDocument = selectedType
    ? selectOptionMetaBoolean(selectedType, "requiresDocument")
    : false;
  const validityMonths = validityMonthsOf(selectedType);

  // The suggestion follows the type and the issue date as the person changes them, until they
  // set the expiry themselves.
  const suggestExpiry = (issuedAt: number | null, months: number | null) => {
    if (isEdit || getFieldState("expiresAt").isDirty) return;
    setValue("expiresAt", suggestExpiryUnix(issuedAt, months), { shouldDirty: false });
  };

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

  return (
    <>
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
                  onOptionChange={(option) =>
                    suggestExpiry(getValues("issuedAt"), validityMonthsOf(option))
                  }
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
                  inputClassProps="font-mono"
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
                  rules={{
                    onChange: () => suggestExpiry(getValues("issuedAt"), validityMonths),
                  }}
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
              {submitLabel}
            </Button>
          </WorkPaneFooter>
        </Form>
      </FormProvider>
    </>
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
      ? formatRange(credential.issuedAt, credential.expiresAt, { year: "always", join: "words" })
      : credential.expiresAt
        ? t("Until {0}", formatUnixDateMedium(credential.expiresAt))
        : null;
  return (
    <SubjectCard
      title={credential.credentialType?.name ?? t("Current credential")}
      badge={
        credential.health
          ? { label: healthLabel(t, credential.health), variant: healthTone(credential.health) }
          : null
      }
      detail={[credential.number, span, credential.document?.originalName]
        .filter(Boolean)
        .join(" · ")}
    />
  );
}
