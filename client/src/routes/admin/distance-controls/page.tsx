import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { Button } from "@/components/ui/button";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { api } from "@/lib/api";
import { distanceProfileDistanceUnitChoices } from "@/lib/choices";
import { DistanceControlService } from "@/services/distance-control";
import { distanceControlSchema } from "@/types/distance-control";
import type { DistanceControl } from "@/types/distance-control";
import type { DistanceProfile } from "@/types/distance-profile";
import type { GenericLimitOffsetResponse } from "@/types/server";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Loader2Icon, SaveIcon } from "lucide-react";
import { useEffect } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";

const distanceControlService = new DistanceControlService();

const profileFields = [
  { name: "loadedMoveDistanceProfileId", label: "Loaded Move" },
  { name: "emptyMoveDistanceProfileId", label: "Empty Move" },
  { name: "payDistanceProfileId", label: "Pay" },
  { name: "billingDistanceProfileId", label: "Billing" },
  { name: "fuelDistanceProfileId", label: "Fuel" },
  { name: "etaOutOfRouteDistanceProfileId", label: "ETA Out-of-Route" },
  { name: "distanceCalculatorPracticalDistanceProfileId", label: "Calculator Practical" },
  { name: "distanceCalculatorShortestDistanceProfileId", label: "Calculator Shortest" },
] as const;

export function DistanceControlsPage() {
  const queryClient = useQueryClient();
  const form = useForm<DistanceControl>({
    resolver: zodResolver(distanceControlSchema),
    defaultValues: {
      storeMileage: true,
      storedDistanceUnits: "Miles",
      postalCodeFallbackToCity: true,
      autoCreateStoredMileage: true,
    } as DistanceControl,
  });

  const controlQuery = useQuery({
    queryKey: ["distance-control"],
    queryFn: () => distanceControlService.get(),
  });

  const profilesQuery = useQuery({
    queryKey: ["distance-profile-options"],
    queryFn: async () =>
      api.get<GenericLimitOffsetResponse<DistanceProfile>>(
        "/distance-profiles/?limit=200&fieldFilters=%5B%7B%22field%22%3A%22status%22%2C%22operator%22%3A%22eq%22%2C%22value%22%3A%22Active%22%7D%5D",
      ),
  });

  const mutation = useMutation({
    mutationFn: (values: DistanceControl) => distanceControlService.patch(values),
    onSuccess: (updated) => {
      toast.success("Distance controls saved");
      form.reset(updated);
      void queryClient.invalidateQueries({ queryKey: ["distance-control"] });
    },
    onError: (error) => {
      toast.error("Failed to save distance controls", {
        description: error instanceof Error ? error.message : "An unexpected error occurred",
      });
    },
  });

  const isDirty = form.formState.isDirty;
  useEffect(() => {
    if (controlQuery.data && !isDirty) {
      form.reset(controlQuery.data);
    }
  }, [controlQuery.data, form, isDirty]);

  const profileOptions =
    profilesQuery.data?.results
      .filter((profile): profile is DistanceProfile & { id: string } => Boolean(profile.id))
      .map((profile) => ({
        label: profile.name,
        value: profile.id,
        description: `${profile.routingType} / ${profile.distanceUnits}`,
      })) ?? [];

  return (
    <AdminPageLayout>
      <PageHeader
        title="Distance Controls"
        description="Assign mileage profiles and stored mileage behavior for this business unit"
      />
      <FormProvider {...form}>
        <form
          className="flex max-w-5xl flex-col gap-4 p-4"
          onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
        >
          <FormSection title="Stored Mileage" description="Control cache lookup and candidate capture.">
            <FormGroup cols={2}>
              <FormControl>
                <SwitchField
                  control={form.control}
                  name="storeMileage"
                  label="Use stored mileage"
                  description="Check stored lane records before calling PC*Miler."
                  outlined
                />
              </FormControl>
              <FormControl>
                <SwitchField
                  control={form.control}
                  name="autoCreateStoredMileage"
                  label="Auto-create records"
                  description="Buffer successful PC*Miler results for scheduled upsert."
                  outlined
                />
              </FormControl>
              <FormControl>
                <SwitchField
                  control={form.control}
                  name="postalCodeFallbackToCity"
                  label="Postal fallback"
                  description="Use city/state keys when postal codes are unavailable."
                  outlined
                />
              </FormControl>
              <FormControl>
                <SelectField
                  control={form.control}
                  name="storedDistanceUnits"
                  label="Stored Units"
                  options={distanceProfileDistanceUnitChoices}
                  rules={{ required: true }}
                />
              </FormControl>
            </FormGroup>
          </FormSection>
          <FormSection
            title="Profile Assignments"
            description="Choose active PC*Miler profiles for each mileage purpose."
            className="border-t py-2"
          >
            <FormGroup cols={2}>
              {profileFields.map((field) => (
                <FormControl key={field.name}>
                  <SelectField
                    control={form.control}
                    name={field.name}
                    label={field.label}
                    options={profileOptions}
                    rules={{ required: true }}
                  />
                </FormControl>
              ))}
            </FormGroup>
          </FormSection>
          <div className="flex justify-end">
            <Button type="submit" disabled={mutation.isPending || controlQuery.isLoading}>
              {mutation.isPending ? (
                <Loader2Icon className="size-4 animate-spin" />
              ) : (
                <SaveIcon className="size-4" />
              )}
              Save
            </Button>
          </div>
        </form>
      </FormProvider>
    </AdminPageLayout>
  );
}
