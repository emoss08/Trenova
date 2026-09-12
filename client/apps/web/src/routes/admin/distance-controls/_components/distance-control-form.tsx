import { useT } from "@trenova/shared/i18n/use-t";
import { DistanceProfileAutocompleteField } from "@/components/autocomplete-fields";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { FormSaveDock } from "@/components/form-save-dock";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@trenova/shared/components/ui/card";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";
import { distanceProfileDistanceUnitChoices } from "@/lib/choices";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import {
  distanceControlSchema,
  type DistanceControl,
  type DistanceControlInput,
} from "@/types/distance-control";
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
  const t = useT();

  const { data } = useSuspenseQuery({
    ...queries.distanceControl.get(),
  });

  const form = useForm<DistanceControlInput, unknown, DistanceControl>({
    resolver: zodResolver(distanceControlSchema),
    defaultValues: data,
  });

  const { handleSubmit, reset } = form;

  const { mutateAsync } = useOptimisticMutation<
    DistanceControl,
    DistanceControl,
    unknown,
    DistanceControlInput
  >({
    queryKey: queries.distanceControl.get._def,
    mutationFn: async (values: DistanceControl) => apiService.distanceControlService.patch(values),
    resourceName: "Distance Control",
    resetForm: reset,
    form,
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
          <JurisdictionMileageCard />
          <ProfileAssignmentsCard />
          <FormSaveDock saveButtonContent={t("Save Changes")} />
        </div>
      </Form>
    </FormProvider>
  );
}

function StoredMileageCard() {
  const t = useT();

  const { control } = useFormContext<DistanceControlInput>();

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Stored Mileage Policy")}</CardTitle>
        <CardDescription>
          {t("Configure when lane mileage is reused, how new mileage candidates are captured, and which units are stored for this business unit.")}
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="storeMileage"
              label={t("Use Stored Mileage")}
              description={t("When enabled, calculations check stored lane mileage before calling PC*Miler.")}
              position="left"
            />
          </FormControl>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="autoCreateStoredMileage"
              label={t("Auto-create Stored Mileage")}
              description={t("Successful PC*Miler results are buffered for the scheduled stored mileage upsert job.")}
              position="left"
            />
          </FormControl>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="postalCodeFallbackToCity"
              label={t("Postal Code Fallback")}
              description={t("When postal-code matching is unavailable, fall back to city and state lane keys.")}
              position="left"
            />
          </FormControl>
          <FormControl className="max-w-[400px]">
            <SelectField
              control={control}
              name="storedDistanceUnits"
              label={t("Stored Distance Units")}
              description={t("Unit used when storing reusable local mileage records.")}
              options={distanceProfileDistanceUnitChoices}
              rules={{ required: true }}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function JurisdictionMileageCard() {
  const t = useT();

  const { control } = useFormContext<DistanceControlInput>();

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Jurisdiction Mileage")}</CardTitle>
        <CardDescription>
          {t("Break each move's routed distance down by state or province so IFTA returns can attribute miles to the jurisdictions they were driven in.")}
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="captureJurisdictionMiles"
              label={t("Capture jurisdiction miles")}
              description={t("Ask PC*Miler for the state-by-state mileage report on every move route. Needed for IFTA returns.")}
              tooltip={t("May be billed by PC*Miler as an additional transaction per route. Routes calculated before this is on have no jurisdiction breakdown until they are recalculated.")}
              position="left"
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function ProfileAssignmentsCard() {
  const t = useT();

  const { control } = useFormContext<DistanceControlInput>();

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Distance Profile Assignments")}</CardTitle>
        <CardDescription>
          {t("Assign active PC*Miler profiles to each mileage purpose. These mappings determine routing behavior for shipment moves, rating workflows, and calculator requests.")}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <FormGroup cols={2}>
          {profileFields.map((field) => (
            <FormControl key={field.name} className="min-h-[3em]">
              <DistanceProfileAutocompleteField<DistanceControlInput>
                control={control}
                name={field.name}
                label={t(field.label)}
                description={t(field.description)}
                placeholder={t("Select distance profile")}
                rules={{ required: true }}
              />
            </FormControl>
          ))}
        </FormGroup>
      </CardContent>
    </Card>
  );
}
