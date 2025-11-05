import { SwitchField } from "@/components/fields/switch-field";
import { FormSaveDock } from "@/components/form";
import { BetaTag } from "@/components/ui/beta-tag";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { NumberField } from "@/components/ui/number-input";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { queries } from "@/lib/queries";
import {
  patternConfigSchema,
  type PatternConfigSchema,
} from "@/lib/schemas/pattern-config-schema";
import { api } from "@/services/api";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { useCallback } from "react";
import {
  FormProvider,
  useForm,
  useFormContext,
  useWatch,
} from "react-hook-form";
import { toast } from "sonner";

export default function PatternConfigForm() {
  const queryClient = useQueryClient();
  const patternConfig = useSuspenseQuery({
    ...queries.patternConfig.get(),
  });

  const form = useForm({
    resolver: zodResolver(patternConfigSchema),
    defaultValues: patternConfig.data,
  });

  const { handleSubmit, setError, reset } = form;

  const { mutateAsync } = useApiMutation({
    mutationFn: async (values: PatternConfigSchema) => {
      return await api.patternConfig.update(values);
    },
    onMutate: async (newValues) => {
      await queryClient.cancelQueries({
        queryKey: queries.patternConfig.get._def,
      });

      const previousPatternConfig = queryClient.getQueryData([
        queries.patternConfig.get._def,
      ]);

      queryClient.setQueryData([queries.patternConfig.get._def], newValues);

      return { previousPatternConfig, newValues };
    },
    onSuccess: (newValues) => {
      toast.success("Pattern config updated successfully");

      broadcastQueryInvalidation({
        queryKey: queries.patternConfig.get._def as unknown as string[],
        options: {
          correlationId: `update-pattern-config-${Date.now()}`,
        },
        config: {
          predicate: true,
          refetchType: "all",
        },
      });

      reset(newValues);
    },
    onSettled: () => {
      queryClient.invalidateQueries({
        queryKey: queries.patternConfig.get._def,
      });
    },
    setFormError: setError,
    resourceName: "Pattern Config",
  });

  const onSubmit = useCallback(
    async (values: PatternConfigSchema) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  return (
    <FormProvider {...form}>
      <Form onSubmit={handleSubmit(onSubmit)}>
        <div className="flex flex-col gap-4 pb-14">
          <PatternDetectionForm />
          <FormSaveDock />
        </div>
      </Form>
    </FormProvider>
  );
}

function PatternDetectionForm() {
  const { control } = useFormContext<PatternConfigSchema>();

  const enabled = useWatch({
    control,
    name: "enabled",
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle>
          Dedicated Lane Pattern Detection <BetaTag />
        </CardTitle>
        <CardDescription>
          Configure how the system detects patterns in shipment data to
          automatically suggest dedicated lanes for shipments.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={2}>
          <FormControl cols="full">
            <SwitchField
              control={control}
              name="enabled"
              position="left"
              label="Enabled"
              description="When enabled, the system will automatically detect patterns in shipment data and suggest dedicated lanes for shipments."
            />
          </FormControl>
          {enabled && (
            <>
              <FormControl>
                <NumberField
                  control={control}
                  name="minFrequency"
                  label="Minimum Frequency"
                  placeholder="Enter minimum frequency"
                  rules={{ required: true }}
                  description="Sets the minimum frequency of a pattern to be considered for suggestion."
                />
              </FormControl>
              <FormControl>
                <NumberField
                  control={control}
                  name="analysisWindowDays"
                  label="Analysis Window Days"
                  rules={{ required: true }}
                  placeholder="Enter analysis window days"
                  description="Sets the number of days to analyze for pattern detection."
                />
              </FormControl>
              <FormControl>
                <NumberField
                  control={control}
                  name="minConfidenceScore"
                  label="Minimum Confidence Score"
                  rules={{ required: true }}
                  placeholder="Enter minimum confidence score"
                  description="Sets the minimum confidence score for a pattern to be considered for suggestion."
                />
              </FormControl>
              <FormControl>
                <NumberField
                  control={control}
                  name="suggestionTtlDays"
                  label="Suggestion TTL Days"
                  rules={{ required: true }}
                  placeholder="Enter suggestion TTL days"
                  description="Sets the number of days to keep suggested patterns."
                />
              </FormControl>
              <FormControl cols="full">
                <SwitchField
                  control={control}
                  name="requireExactMatch"
                  position="left"
                  label="Require Exact Match"
                  description="When enabled, the system will only suggest patterns that match exactly with the shipment data. This is useful when you want to ensure that the suggested patterns are very specific to the shipment data."
                />
              </FormControl>
              <FormControl cols="full">
                <SwitchField
                  control={control}
                  name="weightRecentShipments"
                  position="left"
                  label="Weight Recent Shipments"
                  description="When enabled, the system will weight recent shipments more heavily in pattern detection."
                />
              </FormControl>
            </>
          )}
        </FormGroup>
      </CardContent>
    </Card>
  );
}
