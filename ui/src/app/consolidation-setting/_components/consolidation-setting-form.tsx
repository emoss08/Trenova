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
  ConsolidationSettingSchema,
  consolidationSettingSchema,
} from "@/lib/schemas/consolidation-setting-schema";
import { api } from "@/services/api";
import { zodResolver } from "@hookform/resolvers/zod";
import { useSuspenseQuery } from "@tanstack/react-query";
import { useCallback } from "react";
import { FormProvider, useForm } from "react-hook-form";

export default function ConsolidationSettingForm() {
  const consolidationSetting = useSuspenseQuery({
    ...queries.organization.getConsolidationSettings(),
  });

  const form = useForm({
    resolver: zodResolver(consolidationSettingSchema),
    defaultValues: consolidationSetting.data,
  });

  const { handleSubmit, setError, reset, control } = form;

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
        <Card>
          <CardHeader>
            <CardTitle>Shipment Consolidation Settings</CardTitle>
            <CardDescription>
              Configure intelligent consolidation parameters to optimize route
              efficiency and reduce transportation costs. These settings control
              how the system identifies and suggests shipment consolidation
              opportunities.
            </CardDescription>
          </CardHeader>
          <CardContent className="max-w-prose">
            {/* Distance Parameters Section */}
            <div className="space-y-4">
              <div className="border-b pb-2">
                <h3 className="text-lg font-semibold text-foreground">
                  Distance Parameters
                </h3>
                <p className="text-sm text-muted-foreground">
                  Control geographic proximity requirements for consolidation
                  eligibility
                </p>
              </div>
              <FormGroup cols={2}>
                <FormControl className="min-h-[3em]">
                  <NumberField
                    control={control}
                    rules={{ required: true }}
                    name="maxPickupDistance"
                    label="Max. Pickup Distance"
                    description="Maximum distance in miles between pickup locations for shipments to be considered for consolidation"
                  />
                </FormControl>
                <FormControl>
                  <NumberField
                    control={control}
                    rules={{ required: true }}
                    name="maxDeliveryDistance"
                    label="Max. Delivery Distance"
                    description="Maximum distance in miles between delivery locations for shipments to be considered for consolidation"
                  />
                </FormControl>
              </FormGroup>
            </div>
            {/* Route Optimization Section */}
            <div className="space-y-4">
              <div className="border-b pb-2">
                <h3 className="text-lg font-semibold text-foreground">
                  Route Optimization
                </h3>
                <p className="text-sm text-muted-foreground">
                  Define acceptable route modifications and efficiency
                  thresholds
                </p>
              </div>
              <FormGroup cols={1}>
                <FormControl>
                  <NumberField
                    formattedOptions={{
                      style: "decimal",
                    }}
                    control={control}
                    rules={{ required: true }}
                    name="maxRouteDetour"
                    label="Max. Route Detour (%)"
                    description="Maximum percentage increase in total route distance that's acceptable for consolidation"
                  />
                </FormControl>
              </FormGroup>
            </div>
            {/* Timing Constraints Section */}
            <div className="space-y-4">
              <div className="border-b pb-2">
                <h3 className="text-lg font-semibold text-foreground">
                  Timing Constraints
                </h3>
                <p className="text-sm text-muted-foreground">
                  Set temporal requirements for shipment scheduling
                  compatibility
                </p>
              </div>
              <FormGroup cols={2}>
                <FormControl>
                  <NumberField
                    control={control}
                    rules={{ required: true }}
                    name="maxTimeWindowGap"
                    label="Max. Time Window Gap"
                    description="Maximum time gap in minutes between shipments' planned pickup/delivery windows for consolidation"
                  />
                </FormControl>
                <FormControl>
                  <NumberField
                    control={control}
                    rules={{ required: true }}
                    name="minTimeBuffer"
                    label="Min. Time Buffer"
                    description="Minimum time buffer in minutes required between consolidated shipments for operational flexibility"
                  />
                </FormControl>
              </FormGroup>
            </div>
            {/* Consolidation Limits Section */}
            <div className="space-y-4">
              <div className="border-b pb-2">
                <h3 className="text-lg font-semibold text-foreground">
                  Consolidation Limits
                </h3>
                <p className="text-sm text-muted-foreground">
                  Define maximum consolidation group sizes and operational
                  constraints
                </p>
              </div>
              <FormGroup cols={1}>
                <FormControl>
                  <NumberField
                    control={control}
                    rules={{ required: true }}
                    name="maxShipmentsPerGroup"
                    label="Max. Shipments Per Group"
                    description="Maximum number of shipments that can be consolidated into a single group"
                  />
                </FormControl>
              </FormGroup>
            </div>
          </CardContent>
        </Card>
        <FormSaveDock />
      </Form>
    </FormProvider>
  );
}
