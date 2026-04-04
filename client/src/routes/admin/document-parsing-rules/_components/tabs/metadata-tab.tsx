import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@/components/ui/alert";
import { Form, FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { FormSaveDock } from "@/components/form-save-dock";
import { InputField } from "@/components/fields/input-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { SelectField } from "@/components/fields/select-field";
import { NumberField } from "@/components/fields/number-field";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";
import { formatToUserTimezone } from "@/lib/date";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import {
  ruleSetSchema,
  type RuleSet,
  type RuleSetFormValues,
} from "@/types/document-parsing-rule";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
import { InfoIcon, PackageIcon } from "lucide-react";
import { useCallback } from "react";
import { FormProvider, useForm } from "react-hook-form";

const DOCUMENT_KIND_OPTIONS = [
  { value: "RateConfirmation", label: "Rate Confirmation" },
];

export default function MetadataTab({ ruleSetId }: { ruleSetId: string }) {
  const { data } = useQuery({
    ...queries.documentParsingRule.detail(ruleSetId),
  });

  if (!data) return null;

  return <MetadataForm ruleSet={data} />;
}

function PublishedVersionInfo({ ruleSet }: { ruleSet: RuleSet }) {
  const { data: versions } = useQuery({
    ...queries.documentParsingRule.versions(ruleSet.id!),
    enabled: !!ruleSet.publishedVersionId,
  });

  const publishedVersion = versions?.find(
    (v) => v.id === ruleSet.publishedVersionId,
  );

  return (
    <FormSection
      title="Published Version"
      description="The currently active version used for document parsing in production."
    >
      {publishedVersion ? (
        <div className="grid grid-cols-2 gap-4 rounded-lg border p-4 text-sm">
          <div>
            <p className="text-muted-foreground">Version</p>
            <p className="font-medium">
              v{publishedVersion.versionNumber}
              {publishedVersion.label ? ` — ${publishedVersion.label}` : ""}
            </p>
          </div>
          <div>
            <p className="text-muted-foreground">Published</p>
            <p className="font-medium">
              {publishedVersion.publishedAt
                ? formatToUserTimezone(publishedVersion.publishedAt)
                : "N/A"}
            </p>
          </div>
          <div>
            <p className="text-muted-foreground">Parser Mode</p>
            <p className="font-medium capitalize">
              {publishedVersion.parserMode.replace(/_/g, " ")}
            </p>
          </div>
          <div>
            <p className="text-muted-foreground">Status</p>
            <p className="font-medium">{publishedVersion.status}</p>
          </div>
        </div>
      ) : (
        <div className="flex items-center gap-3 rounded-lg border border-dashed p-4">
          <PackageIcon className="size-5 shrink-0 text-muted-foreground" />
          <p className="text-sm text-muted-foreground">
            No version has been published yet. Create and publish a version from
            the Versions tab to start parsing documents with this rule set.
          </p>
        </div>
      )}
    </FormSection>
  );
}

function MetadataForm({ ruleSet }: { ruleSet: RuleSet }) {
  const form = useForm<RuleSetFormValues, unknown, RuleSet>({
    resolver: zodResolver(ruleSetSchema),
    defaultValues: ruleSet,
  });

  const { handleSubmit, reset, setError, control } = form;

  const { mutateAsync } = useOptimisticMutation({
    queryKey: queries.documentParsingRule.detail._def,
    mutationFn: async (values: RuleSet) =>
      apiService.documentParsingRuleService.update(ruleSet.id!, values),
    resourceName: "Rule Set",
    resetForm: reset,
    setFormError: setError,
    invalidateQueries: [
      queries.documentParsingRule.detail._def,
      queries.documentParsingRule.list._def,
    ],
  });

  const onSubmit = useCallback(
    async (values: RuleSet) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  return (
    <FormProvider {...form}>
      <Form onSubmit={handleSubmit(onSubmit)}>
        <div className="space-y-6">
          <PublishedVersionInfo ruleSet={ruleSet} />
          <FormSection
            title="Rule Set Details"
            description="Configure the name, document kind, and priority for this rule set."
          >
            <FormGroup cols={2}>
              <FormControl>
                <InputField
                  control={control}
                  name="name"
                  label="Name"
                  placeholder="e.g. CH Robinson Rate Confirmation"
                  description="A descriptive name to identify this rule set."
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <SelectField
                  control={control}
                  name="documentKind"
                  label="Document Kind"
                  options={DOCUMENT_KIND_OPTIONS}
                  description="The type of document this rule set is designed to parse."
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <NumberField
                  control={control}
                  name="priority"
                  label="Priority"
                  description="Higher priority rules take precedence when multiple rules match the same document."
                />
              </FormControl>
            </FormGroup>
            <Alert variant="info">
              <InfoIcon className="size-4" />
              <AlertTitle>Changing document kind</AlertTitle>
              <AlertDescription>
                Changing the document kind will affect which documents this rule
                set matches. Existing versions and fixtures will not be modified,
                but they may no longer be valid for the new document kind.
                Re-run simulations after changing this value.
              </AlertDescription>
            </Alert>
            <FormGroup cols={2}>
              <FormControl cols={2}>
                <TextareaField
                  control={control}
                  name="description"
                  label="Description"
                  placeholder="Describe when this rule set should be used, what provider or format it targets, and any special considerations..."
                  description="Helps your team understand the purpose of this rule set."
                  minRows={5}
                />
              </FormControl>
            </FormGroup>
          </FormSection>
        </div>
        <FormSaveDock saveButtonContent="Save Changes" />
      </Form>
    </FormProvider>
  );
}
