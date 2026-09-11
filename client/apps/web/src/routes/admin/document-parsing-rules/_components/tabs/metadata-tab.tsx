import { useT } from "@trenova/shared/i18n/use-t";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormSaveDock } from "@/components/form-save-dock";
import { Form, FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";
import { documentKindChoices } from "@/lib/choices";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { ruleSetSchema, type RuleSet } from "@/types/document-parsing-rule";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
import { PackageIcon } from "lucide-react";
import { useCallback } from "react";
import { FormProvider, useForm, useWatch } from "react-hook-form";

export default function MetadataTab({ ruleSetId }: { ruleSetId: string }) {
  const { data } = useQuery({
    ...queries.documentParsingRule.detail(ruleSetId),
  });

  if (!data) return null;

  return <MetadataForm ruleSet={data} />;
}

function PublishedVersionInfo({ ruleSet }: { ruleSet: RuleSet }) {
  const t = useT();

  const { data: versions } = useQuery({
    ...queries.documentParsingRule.versions(ruleSet.id!),
    enabled: !!ruleSet.publishedVersionId,
  });

  const publishedVersion = versions?.find((v) => v.id === ruleSet.publishedVersionId);

  return (
    <FormSection
      title={t("Published Version")}
      description={t("The currently active version used for document parsing in production.")}
    >
      {publishedVersion ? (
        <div className="grid grid-cols-2 gap-4 rounded-lg border p-4 text-sm">
          <div>
            <p className="text-muted-foreground">{t("Version")}</p>
            <p className="font-medium">
              {t("v{0} {1}", publishedVersion.versionNumber, publishedVersion.label ? ` — ${publishedVersion.label}` : "")}
            </p>
          </div>
          <div>
            <p className="text-muted-foreground">{t("Published")}</p>
            <p className="font-medium">
              {publishedVersion.publishedAt
                ? formatToUserTimezone(publishedVersion.publishedAt)
                : "N/A"}
            </p>
          </div>
          <div>
            <p className="text-muted-foreground">{t("Parser Mode")}</p>
            <p className="font-medium capitalize">
              {publishedVersion.parserMode.replace(/_/g, " ")}
            </p>
          </div>
          <div>
            <p className="text-muted-foreground">{t("Status")}</p>
            <p className="font-medium">{publishedVersion.status}</p>
          </div>
        </div>
      ) : (
        <div className="flex items-center gap-3 rounded-lg border border-dashed p-4">
          <PackageIcon className="text-muted-foreground size-5 shrink-0" />
          <p className="text-muted-foreground text-sm">
            {t("No version has been published yet. Create and publish a version from the Versions tab to start parsing documents with this rule set.")}
          </p>
        </div>
      )}
    </FormSection>
  );
}

function MetadataForm({ ruleSet }: { ruleSet: RuleSet }) {
  const t = useT();

  const form = useForm({
    resolver: zodResolver(ruleSetSchema),
    defaultValues: ruleSet,
  });

  const { handleSubmit, reset, control } = form;
  const documentKind = useWatch({ control, name: "documentKind" });
  const showDocumentKindWarning = documentKind !== ruleSet.documentKind;

  const { mutateAsync } = useOptimisticMutation({
    queryKey: queries.documentParsingRule.detail._def,
    mutationFn: async (values: RuleSet) =>
      apiService.documentParsingRuleService.update(ruleSet.id!, values),
    resourceName: "Rule Set",
    resetForm: reset,
    form,
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
          {/* {showDocumentKindWarning && (
            <Alert variant="warning">
              <InfoIcon className="size-4" />
              <AlertTitle>Changing document kind</AlertTitle>
              <AlertDescription>
                Changing the document kind will affect which documents this rule set matches.
                Existing versions and fixtures will not be modified, but they may no longer be valid
                for the new document kind. Re-run simulations after changing this value.
              </AlertDescription>
            </Alert>
          )} */}
          <FormSection
            title={t("Rule Set Details")}
            description={t("Configure the name, document kind, and priority for this rule set.")}
          >
            <FormGroup cols={2}>
              <FormControl>
                <InputField
                  control={control}
                  name="name"
                  label={t("Name")}
                  placeholder={t("e.g. CH Robinson Rate Confirmation")}
                  description={t("A descriptive name to identify this rule set.")}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <SelectField
                  control={control}
                  name="documentKind"
                  label={t("Document Kind")}
                  options={documentKindChoices}
                  description={t("The type of document this rule set is designed to parse.")}
                  rules={{ required: true }}
                  warning={{
                    show: showDocumentKindWarning,
                    message:
                      "Changing this value can invalidate existing versions and fixtures. Re-run simulations before publishing.",
                  }}
                />
              </FormControl>
              <FormControl>
                <NumberField
                  control={control}
                  name="priority"
                  label={t("Priority")}
                  description={t("Higher priority rules take precedence when multiple rules match the same document.")}
                />
              </FormControl>
            </FormGroup>

            <FormGroup cols={2}>
              <FormControl cols={2}>
                <TextareaField
                  control={control}
                  name="description"
                  label={t("Description")}
                  placeholder={t("Describe when this rule set should be used, what provider or format it targets, and any special considerations...")}
                  description={t("Helps your team understand the purpose of this rule set.")}
                  minRows={5}
                />
              </FormControl>
            </FormGroup>
          </FormSection>
        </div>
        <FormSaveDock saveButtonContent={t("Save Changes")} />
      </Form>
    </FormProvider>
  );
}
