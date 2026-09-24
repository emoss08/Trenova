import { useT } from "@trenova/shared/i18n/use-t";
import { ComponentLoader } from "@trenova/shared/components/component-loader";
import { NumberField } from "@/components/fields/number-field";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { getDataRetention, updateDataRetention } from "@/services/data-retention";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";

const AI_FEEDBACK_RETENTION_MIN_DAYS = 30;
const AI_FEEDBACK_RETENTION_DEFAULT_DAYS = 730;

const dataRetentionFormSchema = z.object({
  auditRetentionPeriod: z.number().int().min(1, "Audit retention must be at least 1 day"),
  ediInboundFileRetentionPeriod: z
    .number()
    .int()
    .min(0, "EDI inbound file retention cannot be negative"),
  ediMessageRetentionPeriod: z.number().int().min(0, "EDI message retention cannot be negative"),
  aiFeedbackRetentionPeriod: z
    .number()
    .int()
    .min(AI_FEEDBACK_RETENTION_MIN_DAYS, "AI feedback retention must be at least 30 days"),
});

type DataRetentionFormValues = z.infer<typeof dataRetentionFormSchema>;

export function DataRetentionPage() {
  const t = useT();

  const queryClient = useQueryClient();
  const canUpdate = usePermissionStore((state) =>
    state.hasPermission(Resource.Organization, Operation.Update),
  );
  const { data, isLoading, isError } = useQuery({
    queryKey: ["data-retention"],
    queryFn: getDataRetention,
  });

  const form = useForm<DataRetentionFormValues>({
    resolver: zodResolver(dataRetentionFormSchema),
    defaultValues: {
      auditRetentionPeriod: 120,
      ediInboundFileRetentionPeriod: 0,
      ediMessageRetentionPeriod: 0,
      aiFeedbackRetentionPeriod: AI_FEEDBACK_RETENTION_DEFAULT_DAYS,
    },
    mode: "onChange",
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (!data) return;
    reset({
      auditRetentionPeriod: data.auditRetentionPeriod,
      ediInboundFileRetentionPeriod: data.ediInboundFileRetentionPeriod,
      ediMessageRetentionPeriod: data.ediMessageRetentionPeriod,
      aiFeedbackRetentionPeriod:
        data.aiFeedbackRetentionPeriod > 0
          ? data.aiFeedbackRetentionPeriod
          : AI_FEEDBACK_RETENTION_DEFAULT_DAYS,
    });
  }, [data, reset]);

  const mutation = useApiMutation({
    mutationFn: (values: DataRetentionFormValues) => updateDataRetention(values),
    form,
    resourceName: "Data Retention",
    onSuccess: async () => {
      toast.success(t("Data retention settings saved"));
      await queryClient.invalidateQueries({ queryKey: ["data-retention"] });
    },
  });

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Data retention"),
        description: t(
          "Configure how long audit entries and raw EDI payloads are kept before the nightly purge jobs remove them.",
        ),
      }}
    >
      {isLoading ? (
        <ComponentLoader message={t("Loading data retention settings")} />
      ) : isError ? (
        <div className="bg-background text-muted-foreground rounded-md border p-6 text-sm">
          {t("The data retention settings could not be loaded.")}
        </div>
      ) : (
        <Form
          className="max-w-2xl"
          onSubmit={(event) => {
            void handleSubmit((values) => mutation.mutate(values))(event);
          }}
        >
          <FormSection title={t("Retention windows")} className="bg-muted/20 rounded-md border p-3">
            <FormGroup cols={1}>
              <FormControl>
                <NumberField
                  control={control}
                  name="auditRetentionPeriod"
                  label={t("Audit Retention (days)")}
                  rules={{ required: true }}
                  description={t(
                    "Audit entries older than this are deleted by the nightly audit retention purge.",
                  )}
                />
              </FormControl>
              <FormControl>
                <NumberField
                  control={control}
                  name="ediInboundFileRetentionPeriod"
                  label={t("EDI Inbound File Retention (days)")}
                  description={t(
                    "Raw inbound EDI file contents older than this are blanked while metadata is kept. 0 keeps raw payloads forever.",
                  )}
                />
              </FormControl>
              <FormControl>
                <NumberField
                  control={control}
                  name="ediMessageRetentionPeriod"
                  label={t("EDI Message Retention (days)")}
                  description={t(
                    "Raw X12 and payload snapshots for delivered/inbound messages older than this are blanked. 0 keeps raw payloads forever. Purged messages can no longer be replayed.",
                  )}
                />
              </FormControl>
              <FormControl>
                <NumberField
                  control={control}
                  name="aiFeedbackRetentionPeriod"
                  label={t("AI feedback retention (days)")}
                  rules={{ required: true }}
                  description={t(
                    "Ratings of AI output, with the question and answer each person saw, are deleted after this many days. At least 30.",
                  )}
                />
              </FormControl>
            </FormGroup>
          </FormSection>
          {canUpdate && (
            <div className="mt-3 flex justify-end">
              <Button type="submit" isLoading={mutation.isPending}>
                {t("Save settings")}
              </Button>
            </div>
          )}
        </Form>
      )}
    </PageLayout>
  );
}
