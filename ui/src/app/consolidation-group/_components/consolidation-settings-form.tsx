import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Form } from "@/components/ui/form";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";
import { queries } from "@/lib/queries";
import {
  consolidationSettingSchema,
  type ConsolidationSettingSchema,
} from "@/lib/schemas/consolidation-setting-schema";
import { api } from "@/services/api";
import { zodResolver } from "@hookform/resolvers/zod";
import { useSuspenseQuery } from "@tanstack/react-query";
import { useCallback } from "react";
import { FormProvider, useForm } from "react-hook-form";

export default function ConsolidationSettingsForm() {
  const consolidationSetting = useSuspenseQuery({
    ...queries.organization.getConsolidationSettings(),
  });

  const form = useForm({
    resolver: zodResolver(consolidationSettingSchema),
    defaultValues: consolidationSetting.data,
  });

  const { handleSubmit, setError, reset } = form;
  const { mutateAsync } = useOptimisticMutation({
    queryKey: queries.organization.getConsolidationSettings._def,
    mutationFn: async (values: ConsolidationSettingSchema) =>
      api.consolidationSettings.update(values),
    successMessage: "Consolidation settings updated successfully",
    resourceName: "Consolidation Settings",
    setFormError: setError,
    resetForm: reset,
    invalidateQueries: [queries.organization.getConsolidationSettings._def],
  });

  const onSubmit = useCallback(
    async (values: ConsolidationSettingSchema) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  return (
    <FormProvider {...form}>
      <Form onSubmit={handleSubmit(onSubmit)}>
        <div className="flex flex-col gap-4 pb-14">
          <Card>
            <CardHeader>
              <CardTitle>Consolidation Settings</CardTitle>
            </CardHeader>
            <CardContent>
              <ConsolidationSettingsForm />
            </CardContent>
          </Card>
        </div>
      </Form>
    </FormProvider>
  );
}
