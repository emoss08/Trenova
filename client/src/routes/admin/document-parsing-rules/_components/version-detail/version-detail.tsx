import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { FormSaveDock } from "@/components/form-save-dock";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Form, FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { Tabs, TabsList, TabsPanel, TabsTab } from "@/components/ui/tabs";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";
import { usePermission } from "@/hooks/use-permission";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import {
  ruleVersionSchema,
  type RuleVersion,
  type RuleVersionFormValues,
} from "@/types/document-parsing-rule";
import { Operation, Resource } from "@/types/permission";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertTriangleIcon, ArrowLeftIcon, LockIcon, RocketIcon } from "lucide-react";
import { useCallback } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { JsonEditor } from "./json-editor";
import { MatchConfigEditor } from "./match-config-editor";
import { RuleBuilder } from "./rule-builder";

const PARSER_MODE_OPTIONS = [
  { value: "merge_with_base", label: "Merge with Base Parser" },
  { value: "override_base", label: "Override Base Parser" },
];

const STATUS_BADGE_VARIANT = {
  Draft: "warning",
  Published: "active",
  Archived: "secondary",
} as const;

type VersionDetailProps = {
  versionId: string;
  onBack: () => void;
};

export function VersionDetail({ versionId, onBack }: VersionDetailProps) {
  const { data } = useQuery({
    ...queries.documentParsingRule.version(versionId),
  });

  if (!data) return null;

  return <VersionDetailForm version={data} onBack={onBack} />;
}

function VersionDetailForm({ version, onBack }: { version: RuleVersion; onBack: () => void }) {
  const queryClient = useQueryClient();
  const { allowed: canActivate } = usePermission(Resource.DocumentParsingRule, Operation.Activate);

  const isDraft = version.status === "Draft";
  const isReadOnly = !isDraft;

  const form = useForm<RuleVersionFormValues, unknown, RuleVersion>({
    resolver: zodResolver(ruleVersionSchema),
    defaultValues: version,
  });

  const { handleSubmit, reset, setError, control } = form;

  const { mutateAsync } = useOptimisticMutation({
    queryKey: queries.documentParsingRule.version._def,
    mutationFn: async (values: RuleVersion) =>
      apiService.documentParsingRuleService.updateVersion(version.id!, values),
    resourceName: "Rule Version",
    resetForm: reset,
    setFormError: setError,
    invalidateQueries: [
      queries.documentParsingRule.version._def,
      queries.documentParsingRule.versions._def,
    ],
  });

  const publishMutation = useMutation({
    mutationFn: () => apiService.documentParsingRuleService.publishVersion(version.id!),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: queries.documentParsingRule.version._def,
      });
      void queryClient.invalidateQueries({
        queryKey: queries.documentParsingRule.versions._def,
      });
      void queryClient.invalidateQueries({
        queryKey: queries.documentParsingRule.detail._def,
      });
      void queryClient.invalidateQueries({
        queryKey: queries.documentParsingRule.list._def,
      });
      toast.success("Version published successfully");
    },
    onError: () => {
      toast.error("Failed to publish version");
    },
  });

  const onSubmit = useCallback(
    async (values: RuleVersion) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  const fixtureFailures = Array.isArray(version.validationSummary?.failures)
    ? version.validationSummary.failures
        .map((failure) => {
          if (!failure || typeof failure !== "object") return null;
          const name =
            "name" in failure && typeof failure.name === "string"
              ? failure.name
              : "Unknown fixture";
          const error =
            "error" in failure && typeof failure.error === "string"
              ? failure.error
              : "Validation failed";
          return { name, error };
        })
        .filter((value): value is { name: string; error: string } => value !== null)
    : [];

  const fixtureCount =
    typeof version.validationSummary?.fixtureCount === "number"
      ? version.validationSummary.fixtureCount
      : null;

  const hasValidationIssues = fixtureFailures.length > 0 || fixtureCount !== null;

  return (
    <FormProvider {...form}>
      <Form onSubmit={handleSubmit(onSubmit)}>
        <div className="space-y-2">
          {/* Breadcrumb back link */}
          <button
            type="button"
            onClick={onBack}
            className="group mt-2 flex items-center gap-1 text-xs text-muted-foreground transition-colors hover:text-foreground"
          >
            <ArrowLeftIcon className="size-3 transition-transform group-hover:-translate-x-0.5" />
            Back to versions
          </button>

          {/* Page title row */}
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2.5">
              <h3 className="font-table text-base leading-none tracking-tight">
                Version {version.versionNumber}
              </h3>
              <Badge
                variant={
                  STATUS_BADGE_VARIANT[version.status as keyof typeof STATUS_BADGE_VARIANT] ??
                  "secondary"
                }
              >
                {version.status}
              </Badge>
              {version.label && (
                <span className="text-sm text-muted-foreground">{version.label}</span>
              )}
            </div>
            {isDraft && canActivate && (
              <AlertDialog>
                <AlertDialogTrigger
                  render={
                    <Button
                      type="button"
                      size="sm"
                      className="gap-1.5"
                      disabled={publishMutation.isPending}
                    >
                      <RocketIcon className="size-3.5" />
                      Publish
                    </Button>
                  }
                />
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>Publish Version {version.versionNumber}</AlertDialogTitle>
                    <AlertDialogDescription className="space-y-2">
                      <span>
                        Publishing will make this version the active rule used for document parsing.
                        This action:
                      </span>
                      <ul className="ml-4 list-disc text-sm text-muted-foreground">
                        <li>
                          Activates this version for all incoming documents matching its criteria
                        </li>
                        <li>Archives the currently published version (if any)</li>
                        <li>Makes this version read-only — no further edits will be possible</li>
                      </ul>
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>Cancel</AlertDialogCancel>
                    <AlertDialogAction onClick={() => publishMutation.mutate()}>
                      {publishMutation.isPending ? "Publishing..." : "Publish Version"}
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            )}
          </div>

          {isReadOnly && (
            <div className="flex items-start gap-2.5 rounded-md border border-info/50 bg-info/10 p-3">
              <LockIcon className="mt-0.5 size-4 shrink-0 text-info" />
              <div className="text-sm text-info">
                <p className="font-medium">Read-only version</p>
                <p className="mt-0.5 text-xs opacity-80">
                  {version.status === "Published"
                    ? "Published versions cannot be modified. Create a new version to make changes."
                    : "Archived versions are frozen snapshots of previously published rules."}
                </p>
              </div>
            </div>
          )}

          {hasValidationIssues && (
            <div className="rounded-md border border-destructive/50 bg-destructive/10 p-3">
              <div className="mb-1.5 flex items-center gap-1.5">
                <AlertTriangleIcon className="size-4 text-destructive" />
                <p className="text-sm font-medium text-destructive">Validation Issues</p>
              </div>
              {fixtureCount !== null && (
                <p className="mb-1 text-xs text-destructive/80">
                  {fixtureCount} fixture{fixtureCount !== 1 ? "s" : ""} tested
                </p>
              )}
              {fixtureFailures.length > 0 && (
                <ul className="space-y-1">
                  {fixtureFailures.map((failure, idx) => (
                    <li
                      key={idx}
                      className="rounded bg-destructive/5 px-2 py-1 text-xs text-destructive"
                    >
                      <span className="font-medium">{failure.name}</span>
                      <span className="mx-1.5 text-destructive/50">&mdash;</span>
                      <span>{failure.error}</span>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          )}

          {/* Version settings */}
          <FormSection
            title="Version Settings"
            description="Label this version and choose how it interacts with the base parser."
          >
            <FormGroup cols={2}>
              <FormControl>
                <InputField
                  control={control}
                  name="label"
                  label="Label"
                  placeholder="e.g. Initial draft, Added stops"
                  description="A short description to identify this version."
                  readOnly={isReadOnly}
                />
              </FormControl>
              <FormControl>
                <SelectField
                  control={control}
                  name="parserMode"
                  label="Parser Mode"
                  description="Merge extends the base parser results. Override replaces them entirely."
                  options={PARSER_MODE_OPTIONS}
                  isReadOnly={isReadOnly}
                />
              </FormControl>
            </FormGroup>
          </FormSection>

          {/* Rule configuration tabs */}
          <Tabs defaultValue="match-config">
            <TabsList variant="underline">
              <TabsTab value="match-config">Match Config</TabsTab>
              <TabsTab value="rule-builder">Rule Builder</TabsTab>
              <TabsTab value="json">JSON</TabsTab>
            </TabsList>
            <TabsPanel value="match-config" className="mt-4">
              <MatchConfigEditor />
            </TabsPanel>
            <TabsPanel value="rule-builder" className="mt-4">
              <RuleBuilder />
            </TabsPanel>
            <TabsPanel value="json" className="mt-4">
              <JsonEditor />
            </TabsPanel>
          </Tabs>

          {isDraft && <FormSaveDock saveButtonContent="Save Changes" />}
        </div>
      </Form>
    </FormProvider>
  );
}
