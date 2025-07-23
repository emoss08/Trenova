/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import {
  TractorAutocompleteField,
  TrailerAutocompleteField,
  WorkerAutocompleteField,
} from "@/components/ui/autocomplete-fields";
import { FormControl, FormGroup } from "@/components/ui/form";
import { AssignmentSchema } from "@/lib/schemas/assignment-schema";
import { api } from "@/services/api";
import { useQuery } from "@tanstack/react-query";
import { useEffect } from "react";
import { useFormContext, useWatch } from "react-hook-form";

function useTractorAssignment(tractorId: string) {
  return useQuery({
    queryKey: ["tractor", tractorId, "assignment"],
    queryFn: async () => {
      if (!tractorId) throw new Error("No Tractor ID provided!");

      return await api.assignments.getTractorAssignments(tractorId);
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
        <TractorAutocompleteField<AssignmentSchema>
          name="tractorId"
          control={control}
          label="Tractor"
          rules={{ required: true }}
          placeholder="Select Tractor"
          description="Select the tractor for the assignment."
        />
      </FormControl>
      <FormControl>
        <TrailerAutocompleteField<AssignmentSchema>
          name="trailerId"
          control={control}
          label="Trailer"
          placeholder="Select Trailer"
          description="Select the trailer for the assignment."
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
