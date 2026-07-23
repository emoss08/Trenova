import { ComponentLoader } from "@/components/component-loader";
import { NumberField } from "@/components/fields/number-field";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { Button } from "@/components/ui/button";
import { Form, FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { getDataRetention, updateDataRetention } from "@/services/data-retention";
import { usePermissionStore } from "@/stores/permission-store";
import { Operation, Resource } from "@/types/permission";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";

const dataRetentionFormSchema = z.object({
  auditRetentionPeriod: z
    .number()
    .int()
    .min(1, "Audit retention must be at least 1 day"),
  ediInboundFileRetentionPeriod: z
    .number()
    .int()
    .min(0, "EDI inbound file retention cannot be negative"),
  ediMessageRetentionPeriod: z
    .number()
    .int()
    .min(0, "EDI message retention cannot be negative"),
});

type DataRetentionFormValues = z.infer<typeof dataRetentionFormSchema>;

export function DataRetentionPage() {
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
    },
    mode: "onChange",
  });
  const { control, handleSubmit, reset, setError } = form;

  useEffect(() => {
    if (!data) return;
    reset({
      auditRetentionPeriod: data.auditRetentionPeriod,
      ediInboundFileRetentionPeriod: data.ediInboundFileRetentionPeriod,
      ediMessageRetentionPeriod: data.ediMessageRetentionPeriod,
    });
  }, [data, reset]);

  const mutation = useApiMutation({
    mutationFn: (values: DataRetentionFormValues) => updateDataRetention(values),
    setFormError: setError,
    resourceName: "Data Retention",
    onSuccess: async () => {
      toast.success("Data retention settings saved");
      await queryClient.invalidateQueries({ queryKey: ["data-retention"] });
    },
  });

  return (
    <AdminPageLayout>
      <PageHeader
        title="Data Retention"
        description="Configure how long audit entries and raw EDI payloads are kept before the nightly purge jobs remove them."
      />
      <div className="flex flex-col gap-4 p-4">
        {isLoading ? (
          <ComponentLoader message="Loading data retention settings" />
        ) : isError ? (
          <div className="rounded-md border bg-background p-6 text-sm text-muted-foreground">
            The data retention settings could not be loaded.
          </div>
        ) : (
          <Form
            className="max-w-2xl"
            onSubmit={(event) => {
              void handleSubmit((values) => mutation.mutate(values))(event);
            }}
          >
            <FormSection
              title="Retention Windows"
              className="rounded-md border bg-muted/20 p-3"
            >
              <FormGroup cols={1}>
                <FormControl>
                  <NumberField
                    control={control}
                    name="auditRetentionPeriod"
                    label="Audit Retention (days)"
                    rules={{ required: true }}
                    description="Audit entries older than this are deleted by the nightly audit retention purge."
                  />
                </FormControl>
                <FormControl>
                  <NumberField
                    control={control}
                    name="ediInboundFileRetentionPeriod"
                    label="EDI Inbound File Retention (days)"
                    description="Raw inbound EDI file contents older than this are blanked while metadata is kept. 0 keeps raw payloads forever."
                  />
                </FormControl>
                <FormControl>
                  <NumberField
                    control={control}
                    name="ediMessageRetentionPeriod"
                    label="EDI Message Retention (days)"
                    description="Raw X12 and payload snapshots for delivered/inbound messages older than this are blanked. 0 keeps raw payloads forever. Purged messages can no longer be replayed."
                  />
                </FormControl>
              </FormGroup>
            </FormSection>
            {canUpdate && (
              <div className="mt-3 flex justify-end">
                <Button type="submit" isLoading={mutation.isPending}>
                  Save Settings
                </Button>
              </div>
            )}
          </Form>
        )}
      </div>
    </AdminPageLayout>
  );
}
