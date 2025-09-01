/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { FormSaveDock } from "@/components/form";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { NumberField } from "@/components/ui/number-input";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";
import { queries } from "@/lib/queries";
import {
  DataRetentionSchema,
  dataRetentionSchema,
} from "@/lib/schemas/data-retention-schema";
import { api } from "@/services/api";
import { zodResolver } from "@hookform/resolvers/zod";
import { useSuspenseQuery } from "@tanstack/react-query";
import { useCallback } from "react";
import { FormProvider, useForm, useFormContext } from "react-hook-form";

export default function DataRetentionPolicies() {
  const dataRetention = useSuspenseQuery({
    ...queries.organization.getDataRetention(),
  });

  const form = useForm({
    resolver: zodResolver(dataRetentionSchema),
    defaultValues: dataRetention.data,
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle>Retention Policies</CardTitle>
        <CardDescription>
          Configure data retention policies for your organization.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormProvider {...form}>
          <AuditRetentionPolicyForm />
        </FormProvider>
      </CardContent>
    </Card>
  );
}

function AuditRetentionPolicyForm() {
  const { handleSubmit, setError, reset, control } =
    useFormContext<DataRetentionSchema>();

  const { mutateAsync } = useOptimisticMutation({
    queryKey: queries.organization.getDataRetention._def,
    mutationFn: async (values: DataRetentionSchema) =>
      api.dataRetention.update(values),
    successMessage: "Data retention updated successfully",
    resourceName: "Data Retention",
    setFormError: setError,
    resetForm: reset,
    invalidateQueries: [queries.organization.getDataRetention._def],
  });

  const onSubmit = useCallback(
    async (values: DataRetentionSchema) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  return (
    <Form onSubmit={handleSubmit(onSubmit)}>
      <FormGroup cols={1}>
        <FormControl>
          <NumberField
            rules={{ required: true }}
            control={control}
            name="auditRetentionPeriod"
            label="Audit Retention Period"
            placeholder="Enter audit retention period"
            description="Defines the number of days to retain audit logs."
            sideText="days"
          />
        </FormControl>
      </FormGroup>
      <FormSaveDock />
    </Form>
  );
}
