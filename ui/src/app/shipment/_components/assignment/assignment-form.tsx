import { AutocompleteField } from "@/components/fields/autocomplete";
import { WorkerAutocompleteField } from "@/components/ui/autocomplete-fields";
import { FormControl, FormGroup } from "@/components/ui/form";
import { AssignmentSchema } from "@/lib/schemas/assignment-schema";
import { TractorSchema } from "@/lib/schemas/tractor-schema";
import { TrailerSchema } from "@/lib/schemas/trailer-schema";
import { getTractorAssignments } from "@/services/tractor";
import { useQuery } from "@tanstack/react-query";
import { useEffect } from "react";
import { useFormContext, useWatch } from "react-hook-form";

function useTractorAssignment(tractorId: string) {
  return useQuery({
    queryKey: ["tractor", tractorId, "assignment"],
    queryFn: async () => {
      if (!tractorId) throw new Error("No Tractor ID provided!");

      const response = await getTractorAssignments(tractorId);
      return response.data;
    },
    enabled: !!tractorId && tractorId !== "",
    staleTime: 30000,
    gcTime: 5 * 60 * 1000,
  });
}

export function AssignmentForm() {
  const { control, setValue, getValues } = useFormContext<AssignmentSchema>();

  const tractorId = useWatch({
    control,
    name: "tractorId",
  });

  const { data: assignmentData } = useTractorAssignment(tractorId);

  // Only set worker values when tractor changes
  useEffect(() => {
    if (assignmentData && tractorId) {
      setValue("primaryWorkerId", assignmentData.primaryWorkerId || "");
      setValue("secondaryWorkerId", assignmentData.secondaryWorkerId || "");
    }
  }, [tractorId, assignmentData, setValue, getValues]);

  return (
    <FormGroup cols={2}>
      <FormControl>
        <AutocompleteField<TractorSchema, AssignmentSchema>
          name="tractorId"
          control={control}
          link="/tractors/"
          label="Tractor"
          rules={{ required: true }}
          placeholder="Select Tractor"
          description="Select the tractor for the assignment."
          getOptionValue={(option) => option.id || ""}
          getDisplayValue={(option) => `${option.code}`}
          renderOption={(option) => `${option.code}`}
        />
      </FormControl>
      <FormControl>
        <AutocompleteField<TrailerSchema, AssignmentSchema>
          name="trailerId"
          control={control}
          link="/trailers/"
          label="Trailer"
          rules={{ required: true }}
          placeholder="Select Trailer"
          description="Select the trailer for the assignment."
          getOptionValue={(option) => option.id || ""}
          getDisplayValue={(option) => `${option.code}`}
          renderOption={(option) => `${option.code}`}
        />
      </FormControl>
      <FormControl>
        <WorkerAutocompleteField<AssignmentSchema>
          name="primaryWorkerId"
          control={control}
          label="Primary Worker"
          rules={{ required: true }}
          placeholder="Select Primary Worker"
          description="Select the primary worker for the assignment."
        />
      </FormControl>
      <FormControl>
        <WorkerAutocompleteField<AssignmentSchema>
          name="secondaryWorkerId"
          control={control}
          label="Secondary Worker"
          clearable={true}
          placeholder="Select Secondary Worker"
          description="Select the secondary worker for the assignment."
        />
      </FormControl>
    </FormGroup>
  );
}
