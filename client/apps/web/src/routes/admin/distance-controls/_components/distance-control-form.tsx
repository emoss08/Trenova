import { DistanceProfileAutocompleteField } from "@/components/autocomplete-fields";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { FormSaveDock } from "@/components/form-save-dock";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@trenova/shared/components/ui/card";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";
import { distanceProfileDistanceUnitChoices } from "@/lib/choices";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { distanceControlSchema, type DistanceControl } from "@/types/distance-control";
import { zodResolver } from "@hookform/resolvers/zod";
import { useSuspenseQuery } from "@tanstack/react-query";
import { useCallback } from "react";
import { FormProvider, useForm, useFormContext } from "react-hook-form";

const profileFields = [
  {
    name: "loadedMoveDistanceProfileId",
    label: "Loaded Move",
    description: "Profile used when calculating mileage for loaded shipment moves.",
  },
  {
    name: "emptyMoveDistanceProfileId",
    label: "Empty Move",
    description: "Profile used when calculating mileage for empty repositioning moves.",
  },
  {
    name: "payDistanceProfileId",
    label: "Pay",
    description: "Profile used by driver pay mileage workflows.",
  },
  {
    name: "billingDistanceProfileId",
    label: "Billing",
    description: "Profile used by customer billing mileage workflows.",
  },
  {
    name: "fuelDistanceProfileId",
    label: "Fuel",
    description: "Profile used by fuel mileage workflows.",
  },
  {
    name: "etaOutOfRouteDistanceProfileId",
    label: "ETA Out-of-Route",
    description: "Profile used when measuring ETA out-of-route variance.",
  },
  {
    name: "distanceCalculatorPracticalDistanceProfileId",
    label: "Calculator Practical",
    description: "Default practical routing profile for the distance calculator.",
  },
  {
    name: "distanceCalculatorShortestDistanceProfileId",
    label: "Calculator Shortest",
    description: "Shortest-route profile for shortest-path calculator requests.",
  },
] as const;

export default function DistanceControlForm() {
  const { data } = useSuspenseQuery({
    ...queries.distanceControl.get(),
  });

  const form = useForm<DistanceControl>({
    resolver: zodResolver(distanceControlSchema),
    defaultValues: data,
  });

  const { handleSubmit, reset, setError } = form;

  const { mutateAsync } = useOptimisticMutation<
    DistanceControl,
    DistanceControl,
    unknown,
    DistanceControl
  >({
    queryKey: queries.distanceControl.get._def,
    mutationFn: async (values: DistanceControl) => apiService.distanceControlService.patch(values),
    resourceName: "Distance Control",
    resetForm: reset,
    setFormError: setError,
    invalidateQueries: [queries.distanceControl.get._def],
  });

  const onSubmit = useCallback(
    async (values: DistanceControl) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  return (
    <FormProvider {...form}>
      <Form onSubmit={handleSubmit(onSubmit)}>
        <div className="flex flex-col gap-4 pb-14">
          <StoredMileageCard />
          <ProfileAssignmentsCard />
          <FormSaveDock saveButtonContent="Save Changes" />
        </div>
      </Form>
    </FormProvider>
  );
}

function StoredMileageCard() {
  const { control } = useFormContext<DistanceControl>();

  return (
    <Card>
      <CardHeader>
        <CardTitle>Stored Mileage Policy</CardTitle>
        <CardDescription>
          Configure when lane mileage is reused, how new mileage candidates are captured, and which
          units are stored for this business unit.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="storeMileage"
              label="Use Stored Mileage"
              description="When enabled, calculations check stored lane mileage before calling PC*Miler."
              position="left"
            />
          </FormControl>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="autoCreateStoredMileage"
              label="Auto-create Stored Mileage"
              description="Successful PC*Miler results are buffered for the scheduled stored mileage upsert job."
              position="left"
            />
          </FormControl>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="postalCodeFallbackToCity"
              label="Postal Code Fallback"
              description="When postal-code matching is unavailable, fall back to city and state lane keys."
              position="left"
            />
          </FormControl>
          <FormControl className="max-w-[400px]">
            <SelectField
              control={control}
              name="storedDistanceUnits"
              label="Stored Distance Units"
              description="Unit used when storing reusable local mileage records."
              options={distanceProfileDistanceUnitChoices}
              rules={{ required: true }}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function ProfileAssignmentsCard() {
  const { control } = useFormContext<DistanceControl>();

  return (
    <Card>
      <CardHeader>
        <CardTitle>Distance Profile Assignments</CardTitle>
        <CardDescription>
          Assign active PC*Miler profiles to each mileage purpose. These mappings determine routing
          behavior for shipment moves, rating workflows, and calculator requests.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <FormGroup cols={2}>
          {profileFields.map((field) => (
            <FormControl key={field.name} className="min-h-[3em]">
              <DistanceProfileAutocompleteField<DistanceControl>
                control={control}
                name={field.name}
                label={field.label}
                description={field.description}
                placeholder="Select distance profile"
                rules={{ required: true }}
              />
            </FormControl>
          ))}
        </FormGroup>
      </CardContent>
    </Card>
  );
}

