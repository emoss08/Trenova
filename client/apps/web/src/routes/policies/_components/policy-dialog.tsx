import { useT } from "@trenova/shared/i18n/use-t";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  createWorkerPolicy,
  updateWorkerPolicy,
  WORKER_POLICIES_KEY,
  type WorkerPolicyRow,
} from "@/lib/graphql/self-service";
import { apiService } from "@/services/api";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
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
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { getTodayDate } from "@trenova/shared/lib/date";
import { POLICY_AUDIENCE_ORDER, policyAudienceLabel } from "@trenova/shared/lib/self-service";
import { cn } from "@trenova/shared/lib/utils";
import {
  workerPolicyFormSchema,
  type WorkerPolicyFormValues,
} from "@trenova/shared/types/self-service";
import {
  AlertTriangleIcon,
  FileTextIcon,
  PenLineIcon,
  ShieldCheckIcon,
  UploadCloudIcon,
  XIcon,
} from "lucide-react";
import { useEffect, useRef, useState, type DragEvent } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";

type Source = "text" | "document";

const SOURCE_ITEMS = [
  { value: "text", label: "Write it here", icon: PenLineIcon },
  { value: "document", label: "Attach a document", icon: FileTextIcon },
] satisfies { value: Source; label: string; icon: typeof PenLineIcon }[];

const AUDIENCE_OPTIONS = POLICY_AUDIENCE_ORDER.map((value) => ({
  value,
  label: policyAudienceLabel(value),
}));

const STATUS_OPTIONS = [
  { value: "Active", label: "In force" },
  { value: "Inactive", label: "Retired" },
];

// A document is filed against the policy it belongs to, so the office's
// document tools find it under the policy rather than under nothing.
const POLICY_RESOURCE_TYPE = "worker_policy";
const PENDING_RESOURCE_ID = "pending";
const ACCEPTED_TYPES = "application/pdf,image/*";

export type PolicyDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  policy: WorkerPolicyRow | null;
};

function defaultsFor(policy: WorkerPolicyRow | null): WorkerPolicyFormValues {
  if (!policy) {
    return {
      code: "",
      title: "",
      summary: null,
      body: null,
      documentId: null,
      versionLabel: "1",
      requiresSignature: true,
      appliesTo: "All",
      effectiveFrom: getTodayDate(),
      status: "Active",
    };
  }
  return {
    code: policy.code,
    title: policy.title,
    summary: policy.summary ?? null,
    body: policy.body ?? null,
    documentId: policy.documentId ?? null,
    versionLabel: policy.versionLabel,
    requiresSignature: policy.requiresSignature,
    appliesTo: policy.appliesTo as WorkerPolicyFormValues["appliesTo"],
    effectiveFrom: policy.effectiveFrom,
    status: policy.status === "Active" ? "Active" : "Inactive",
  };
}

function SectionHeading({ children, hint }: { children: string; hint?: string }) {
  return (
    <div className="col-span-full -mb-1 flex items-baseline justify-between gap-2 border-b pb-1.5 not-first:mt-2">
      <p className="text-[11px] font-semibold tracking-wide uppercase">{children}</p>
      {hint ? <p className="text-muted-foreground text-xs">{hint}</p> : null}
    </div>
  );
}

/**
 * Publishing or revising a policy. The version label is asked for up front
 * because it is what a signature records: a revision that changes the words
 * under a signed version is refused by the server, and the dialog says so
 * before anybody gets that far.
 */
export function PolicyDialog({ open, onOpenChange, policy }: PolicyDialogProps) {
  const t = useT();

  const queryClient = useQueryClient();
  const isEdit = Boolean(policy);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [attachedName, setAttachedName] = useState<string | null>(null);
  // The source follows the policy until somebody picks one; picking is an
  // event, so the choice is held as an override rather than synced in an effect.
  const [sourceOverride, setSourceOverride] = useState<Source | null>(null);
  const [dragging, setDragging] = useState(false);

  const form = useForm<WorkerPolicyFormValues>({
    resolver: zodResolver(workerPolicyFormSchema) as Resolver<WorkerPolicyFormValues>,
    defaultValues: defaultsFor(policy),
  });
  const { control, handleSubmit, reset, setValue, formState } = form;

  useEffect(() => {
    if (!open) return;
    reset(defaultsFor(policy));
  }, [open, policy, reset]);

  const documentId = useWatch({ control, name: "documentId" });
  const body = useWatch({ control, name: "body" });
  const source: Source = sourceOverride ?? (documentId ? "document" : "text");

  function close() {
    setSourceOverride(null);
    setAttachedName(null);
    onOpenChange(false);
  }
  const wordsChanged =
    Boolean(policy) &&
    ((body ?? "").trim() !== (policy?.body ?? "").trim() ||
      (documentId ?? null) !== (policy?.documentId ?? null));

  const attach = useMutation({
    mutationFn: (file: File) =>
      apiService.documentService.upload({
        file,
        resourceType: POLICY_RESOURCE_TYPE,
        resourceId: policy?.id ?? PENDING_RESOURCE_ID,
        description: `Policy document: ${file.name}`,
      }),
    onSuccess: (document, file) => {
      setValue("documentId", document.id, { shouldDirty: true, shouldValidate: true });
      setAttachedName(file.name);
      toast.success(t("Document attached"));
    },
    onError: (error: Error) => toast.error(error.message || "Could not attach the document"),
  });

  const { mutateAsync, isPending } = useApiMutation<
    { id: string },
    WorkerPolicyFormValues,
    unknown,
    WorkerPolicyFormValues
  >({
    form,
    resourceName: "Policy",
    mutationFn: (values) => {
      const input = {
        code: values.code,
        title: values.title,
        summary: values.summary ?? undefined,
        body: values.body ?? undefined,
        documentId: values.documentId ?? undefined,
        versionLabel: values.versionLabel,
        requiresSignature: values.requiresSignature,
        appliesTo: values.appliesTo,
        effectiveFrom: values.effectiveFrom,
        status: values.status,
      };
      return policy ? updateWorkerPolicy(policy.id, input) : createWorkerPolicy(input);
    },
    onSuccess: () => {
      toast.success(isEdit ? "Policy updated" : "Policy published");
      void queryClient.invalidateQueries({ queryKey: [WORKER_POLICIES_KEY] });
      close();
    },
  });

  function detach() {
    setValue("documentId", null, { shouldDirty: true, shouldValidate: true });
    setAttachedName(null);
  }

  function pickFile(file: File | undefined) {
    if (!file) return;
    if (!file.type.startsWith("image/") && file.type !== "application/pdf") {
      toast.error(t("Attach a PDF or an image"));
      return;
    }
    attach.mutate(file);
  }

  function onDrop(event: DragEvent<HTMLDivElement>) {
    event.preventDefault();
    setDragging(false);
    pickFile(event.dataTransfer.files?.[0]);
  }

  const bodyError = formState.errors.body?.message;

  return (
    <Dialog open={open} onOpenChange={(next) => (next ? onOpenChange(true) : close())}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{isEdit ? t("Revise the policy") : t("Publish a policy")}</DialogTitle>
          <DialogDescription>
            {t(
              "Either a short text or an attached document — a signature has to be on something. Everybody it applies to is asked to sign the version in force.",
            )}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            onSubmit={(submitEvent) => {
              submitEvent.preventDefault();
              submitEvent.stopPropagation();
              void handleSubmit((values) => mutateAsync(values))(submitEvent);
            }}
          >
            <FormGroup className="pb-2" cols={2}>
              <SectionHeading>{t("What it is")}</SectionHeading>
              <FormControl>
                <InputField<WorkerPolicyFormValues>
                  control={control}
                  name="code"
                  label={t("Code")}
                  placeholder={t("e.g. HANDBOOK")}
                  description={t("A short reference for the policy; it is stored in upper case.")}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <InputField<WorkerPolicyFormValues>
                  control={control}
                  name="title"
                  label={t("Title")}
                  placeholder={t("e.g. Driver handbook")}
                  description={t(
                    "The name drivers see in their list of policies to read and sign.",
                  )}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl cols="full">
                <InputField<WorkerPolicyFormValues>
                  control={control}
                  name="summary"
                  label={t("One-line summary")}
                  placeholder={t("e.g. Hours, conduct and equipment rules for every driver")}
                  description={t("Optional line a driver sees before opening the policy.")}
                />
              </FormControl>

              <SectionHeading hint={t("What the signature is on")}>{t("The text")}</SectionHeading>
              <FormControl cols="full">
                <SegmentedControl<Source>
                  items={SOURCE_ITEMS}
                  value={source}
                  onValueChange={setSourceOverride}
                  fullWidth
                  aria-label={t("Where the policy text lives")}
                />
              </FormControl>
              {source === "text" ? (
                <FormControl cols="full">
                  <TextareaField<WorkerPolicyFormValues>
                    control={control}
                    name="body"
                    label={t("Policy text")}
                    placeholder={t(
                      "e.g. Drivers must complete a pre-trip inspection before every dispatch...",
                    )}
                    description={t(
                      "The wording drivers read and sign; short policies fit here, longer ones are better attached as a document.",
                    )}
                    rows={6}
                  />
                </FormControl>
              ) : (
                <FormControl cols="full">
                  <div className="flex flex-col gap-1.5">
                    {documentId ? (
                      <div className="bg-muted/30 flex items-center gap-3 rounded-lg border p-3">
                        <span className="bg-accent inline-flex size-7 shrink-0 items-center justify-center rounded-md">
                          <FileTextIcon className="size-4" />
                        </span>
                        <span className="min-w-0 flex-1">
                          <span className="block truncate text-sm font-medium">
                            {attachedName ?? t("Attached document")}
                          </span>
                          <span className="text-muted-foreground flex items-center gap-1 text-xs">
                            <ShieldCheckIcon className="size-3" />
                            {t("Its checksum is copied onto every signature")}
                          </span>
                        </span>
                        <Button
                          type="button"
                          size="xs"
                          variant="outline"
                          isLoading={attach.isPending}
                          onClick={() => fileInputRef.current?.click()}
                        >
                          {t("Replace")}
                        </Button>
                        <Button
                          type="button"
                          size="icon-xs"
                          variant="ghost"
                          aria-label={t("Remove the attached document")}
                          onClick={detach}
                        >
                          <XIcon className="size-3.5" />
                        </Button>
                      </div>
                    ) : (
                      <div
                        role="button"
                        tabIndex={0}
                        onClick={() => fileInputRef.current?.click()}
                        onKeyDown={(event) => {
                          if (event.key === "Enter" || event.key === " ") {
                            event.preventDefault();
                            fileInputRef.current?.click();
                          }
                        }}
                        onDragOver={(event) => {
                          event.preventDefault();
                          setDragging(true);
                        }}
                        onDragLeave={() => setDragging(false)}
                        onDrop={onDrop}
                        className={cn(
                          "text-muted-foreground hover:border-border hover:bg-muted/40 flex cursor-pointer flex-col items-center justify-center gap-1.5 rounded-lg border border-dashed px-4 py-6 text-center text-xs transition-colors",
                          dragging && "border-primary bg-primary/5 text-foreground",
                          bodyError && "border-destructive/60",
                        )}
                      >
                        <UploadCloudIcon
                          className={cn(
                            "size-5 transition-transform",
                            dragging && "-translate-y-0.5",
                          )}
                        />
                        <span className="text-foreground font-medium">
                          {attach.isPending
                            ? t("Uploading…")
                            : t("Drop a PDF here, or click to choose")}
                        </span>
                        <span>{t("Drivers open it from Dash. PDFs and images are accepted.")}</span>
                      </div>
                    )}
                    {bodyError ? <p className="text-destructive text-xs">{bodyError}</p> : null}
                    <input
                      ref={fileInputRef}
                      type="file"
                      accept={ACCEPTED_TYPES}
                      className="hidden"
                      onChange={(event) => {
                        const file = event.target.files?.[0];
                        event.target.value = "";
                        pickFile(file);
                      }}
                    />
                  </div>
                </FormControl>
              )}

              {wordsChanged ? (
                <div
                  className="border-warning/40 bg-warning/10 text-warning-foreground col-span-full flex items-start gap-2 rounded-lg border px-3 py-2 text-xs"
                  role="status"
                >
                  <AlertTriangleIcon className="mt-0.5 size-3.5 shrink-0" />
                  <span>
                    {t(
                      "The words changed. Give this a new version, or the server will refuse it if anybody has signed the current one.",
                    )}
                  </span>
                </div>
              ) : null}

              <SectionHeading>{t("Who and when")}</SectionHeading>
              <FormControl>
                <InputField<WorkerPolicyFormValues>
                  control={control}
                  name="versionLabel"
                  label={t("Version")}
                  placeholder={t("e.g. 2026.1")}
                  rules={{ required: true }}
                  description={t(
                    "What a signature records; a new version asks everybody to sign again, the same label keeps existing signatures.",
                  )}
                />
              </FormControl>
              <FormControl>
                <SelectField<WorkerPolicyFormValues>
                  control={control}
                  name="appliesTo"
                  label={t("Applies to")}
                  options={AUDIENCE_OPTIONS}
                  placeholder={t("Choose who it binds")}
                  rules={{ required: true }}
                  description={t(
                    "Which worker types are asked to sign; a handbook for employees is not a contract term for an owner-operator.",
                  )}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<WorkerPolicyFormValues>
                  control={control}
                  name="effectiveFrom"
                  label={t("Effective from")}
                  placeholder={t("e.g. Jan 1")}
                  rules={{ required: true }}
                  description={t("The day this version of the policy takes effect.")}
                />
              </FormControl>
              <FormControl>
                <SelectField<WorkerPolicyFormValues>
                  control={control}
                  name="status"
                  label={t("Status")}
                  options={STATUS_OPTIONS}
                  placeholder={t("Choose a status")}
                  rules={{ required: true }}
                  description={t(
                    "Only a policy in force can be signed; a retired one is kept so old signatures still point at something.",
                  )}
                />
              </FormControl>
              <FormControl cols="full">
                <SwitchField<WorkerPolicyFormValues>
                  control={control}
                  name="requiresSignature"
                  label={t("Needs a signature")}
                  description={t(
                    "On, drivers type their name to sign it; off means they only confirm they have read it.",
                  )}
                />
              </FormControl>
            </FormGroup>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={close}>
                {t("Cancel")}
              </Button>
              <Button type="submit" isLoading={isPending}>
                {isEdit ? t("Save") : t("Publish")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
