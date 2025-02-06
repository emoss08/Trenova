import { AsyncSelectField } from "@/components/fields/async-select";
import { FormControl, FormGroup } from "@/components/ui/form";
import { AssignmentSchema } from "@/lib/schemas/assignment-schema";
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
  const { control, setValue } = useFormContext<AssignmentSchema>();

  // If the tractorId is in place we need to fetch the workers assigned to it
  // and automatically populate the workerID and secondaryWorkerID(If exists)

  const tractorId = useWatch({
    control,
    name: "tractorId",
    exact: true,
    defaultValue: "",
  });

  const {
    data: assignmentData,
    isLoading: isAssignmentLoading,
    error: assignmentError,
  } = useTractorAssignment(tractorId);

  useEffect(() => {
    if (assignmentData && !isAssignmentLoading) {
      setValue("primaryWorkerId", assignmentData.primaryWorkerId, {
        shouldDirty: true,
        shouldValidate: true,
      });

      // Update the secondary worker if it exists
      if (assignmentData.secondaryWorkerId) {
        setValue("secondaryWorkerId", assignmentData.secondaryWorkerId, {
          shouldDirty: true,
          shouldValidate: true,
        });
      } else {
        setValue("secondaryWorkerId", null, {
          shouldDirty: true,
          shouldValidate: true,
        });
      }
    }
  }, [assignmentData, isAssignmentLoading, setValue]);

  return (
    <FormGroup cols={2}>
      <FormControl>
        <AsyncSelectField
          name="tractorId"
          control={control}
          link="/tractors"
          label="Tractor"
          rules={{ required: true }}
          placeholder="Select Tractor"
          description="Select the tractor for the assignment."
          // TODO(wolfred): We need to change this to include the actual user permissions
          hasPermission
          hasPopoutWindow
          popoutLink="/shipments/configurations/tractors/"
          popoutLinkLabel="Tractor"
        />
      </FormControl>
      <FormControl>
        <AsyncSelectField
          name="trailerId"
          control={control}
          link="/trailers"
          label="Trailer"
          rules={{ required: true }}
          placeholder="Select Trailer"
          description="Select the trailer for the assignment."
          // TODO(wolfred): We need to change this to include the actual user permissions
          hasPermission
          hasPopoutWindow
          popoutLink="/shipments/configurations/trailers/"
          popoutLinkLabel="Trailer"
        />
      </FormControl>
      <FormControl>
        <AsyncSelectField
          name="primaryWorkerId"
          control={control}
          link="/workers"
          label="Primary Worker"
          rules={{ required: true }}
          placeholder="Select Primary Worker"
          description="Select the primary worker for the assignment."
          // TODO(wolfred): We need to change this to include the actual user permissions
          hasPermission
          hasPopoutWindow
          popoutLink="/shipments/configurations/workers/"
          popoutLinkLabel="Primary Worker"
        />
      </FormControl>
      <FormControl>
        <AsyncSelectField
          isClearable
          name="secondaryWorkerId"
          control={control}
          link="/workers"
          label="Secondary Worker"
          placeholder="Select Secondary Worker"
          description="Select the secondary worker for the assignment."
          // TODO(wolfred): We need to change this to include the actual user permissions
          hasPermission
          hasPopoutWindow
          popoutLink="/shipments/configurations/workers/"
          popoutLinkLabel="Secondary Worker"
        />
      </FormControl>
    </FormGroup>
  );
}
