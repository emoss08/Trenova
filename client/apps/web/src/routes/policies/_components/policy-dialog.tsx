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
import { Badge } from "@trenova/shared/components/ui/badge";
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
import { Label } from "@trenova/shared/components/ui/label";
import { getTodayDate } from "@trenova/shared/lib/date";
import { POLICY_AUDIENCE_ORDER, policyAudienceLabel } from "@trenova/shared/lib/self-service";
import {
  workerPolicyFormSchema,
  type WorkerPolicyFormValues,
} from "@trenova/shared/types/self-service";
import { FileTextIcon, PaperclipIcon, XIcon } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";

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

/**
 * Publishing or revising a policy. The version label is asked for up front
 * because it is what a signature records: a revision that changes the words
 * under a signed version is refused by the server, and the dialog says so
 * before anybody gets that far.
 */
export function PolicyDialog({ open, onOpenChange, policy }: PolicyDialogProps) {
  const queryClient = useQueryClient();
  const isEdit = Boolean(policy);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [attachedName, setAttachedName] = useState<string | null>(null);

  const form = useForm<WorkerPolicyFormValues>({
    resolver: zodResolver(workerPolicyFormSchema) as Resolver<WorkerPolicyFormValues>,
    defaultValues: defaultsFor(policy),
  });
  const { control, handleSubmit, reset, setValue } = form;

  useEffect(() => {
    if (!open) return;
    reset(defaultsFor(policy));
  }, [open, policy, reset]);

  const documentId = useWatch({ control, name: "documentId" });
  const body = useWatch({ control, name: "body" });
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
      toast.success("Document attached");
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
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{isEdit ? "Revise the policy" : "Publish a policy"}</DialogTitle>
          <DialogDescription>
            Either a short text or an attached document — a signature has to be on something.
            Everybody it applies to is asked to sign the version in force.
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
              <FormControl>
                <InputField<WorkerPolicyFormValues>
                  control={control}
                  name="code"
                  label="Code"
                  placeholder="e.g. HANDBOOK"
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <InputField<WorkerPolicyFormValues>
                  control={control}
                  name="title"
                  label="Title"
                  placeholder="e.g. Driver handbook"
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl cols="full">
                <InputField<WorkerPolicyFormValues>
                  control={control}
                  name="summary"
                  label="One-line summary"
                  placeholder="What a driver sees before opening it"
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField<WorkerPolicyFormValues>
                  control={control}
                  name="body"
                  label="Policy text"
                  placeholder="Short policies can be written here. Longer ones are better attached as a document."
                />
              </FormControl>

              <FormControl cols="full">
                <div className="flex flex-col gap-1.5">
                  <Label>Attached document</Label>
                  <div className="flex flex-wrap items-center gap-2">
                    {documentId ? (
                      <Badge variant="secondary" className="gap-1">
                        <FileTextIcon className="size-3" />
                        {attachedName ?? "Attached document"}
                        <button
                          type="button"
                          className="ml-1"
                          aria-label="Remove the attached document"
                          onClick={() => {
                            setValue("documentId", null, {
                              shouldDirty: true,
                              shouldValidate: true,
                            });
                            setAttachedName(null);
                          }}
                        >
                          <XIcon className="size-3" />
                        </button>
                      </Badge>
                    ) : null}
                    <Button
                      type="button"
                      size="xs"
                      variant="outline"
                      isLoading={attach.isPending}
                      onClick={() => fileInputRef.current?.click()}
                    >
                      <PaperclipIcon className="size-3.5" />
                      {documentId ? "Replace" : "Attach a PDF"}
                    </Button>
                    <input
                      ref={fileInputRef}
                      type="file"
                      accept="application/pdf,image/*"
                      className="hidden"
                      onChange={(event) => {
                        const file = event.target.files?.[0];
                        event.target.value = "";
                        if (file) attach.mutate(file);
                      }}
                    />
                  </div>
                  <p className="text-muted-foreground text-xs">
                    Drivers open it from Dash. The file&apos;s checksum is copied onto every
                    signature, so a signature is tied to the bytes and not just the row.
                  </p>
                </div>
              </FormControl>

              <FormControl>
                <InputField<WorkerPolicyFormValues>
                  control={control}
                  name="versionLabel"
                  label="Version"
                  placeholder="e.g. 2026.1"
                  rules={{ required: true }}
                  description={
                    wordsChanged
                      ? "The words changed — give this a new version, or the server will refuse it if anybody has signed."
                      : "What a signature records."
                  }
                />
              </FormControl>
              <FormControl>
                <SelectField<WorkerPolicyFormValues>
                  control={control}
                  name="appliesTo"
                  label="Applies to"
                  options={AUDIENCE_OPTIONS}
                  rules={{ required: true }}
                  description="A handbook for employees is not a contract term for an owner-operator."
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<WorkerPolicyFormValues>
                  control={control}
                  name="effectiveFrom"
                  label="Effective from"
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <SelectField<WorkerPolicyFormValues>
                  control={control}
                  name="status"
                  label="Status"
                  options={STATUS_OPTIONS}
                  rules={{ required: true }}
                  description="A retired policy is kept so old signatures still point at something."
                />
              </FormControl>
              <FormControl cols="full">
                <SwitchField<WorkerPolicyFormValues>
                  control={control}
                  name="requiresSignature"
                  label="Needs a signature"
                  description="Off means drivers only confirm they have read it."
                />
              </FormControl>
            </FormGroup>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" isLoading={isPending}>
                {isEdit ? "Save" : "Publish"}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
