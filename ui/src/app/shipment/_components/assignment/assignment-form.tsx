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
      setValue("secondaryWorkerId", assignmentData.secondaryWorkerId || null);
      console.log("Form values:", getValues());
    }
  }, [tractorId, assignmentData, setValue, getValues]);

  return (
    <FormGroup cols={2}>
      <FormControl>
        <AsyncSelectField
          name="tractorId"
          control={control}
          link="/tractors/"
          label="Tractor"
          rules={{ required: true }}
          placeholder="Select Tractor"
          description="Select the tractor for the assignment."
          // TODO(wolfred): We need to change this to include the actual user permissions
          hasPermission
          hasPopoutWindow
          popoutLink="/shipments/configurations/tractors/"
          popoutLinkLabel="Tractor"
          valueKey="code"
        />
      </FormControl>
      <FormControl>
        <AsyncSelectField
          name="trailerId"
          control={control}
          link="/trailers/"
          label="Trailer"
          rules={{ required: true }}
          placeholder="Select Trailer"
          description="Select the trailer for the assignment."
          // TODO(wolfred): We need to change this to include the actual user permissions
          hasPermission
          hasPopoutWindow
          popoutLink="/shipments/configurations/trailers/"
          popoutLinkLabel="Trailer"
          valueKey={["code"]}
        />
      </FormControl>
      <FormControl>
        <AsyncSelectField
          name="primaryWorkerId"
          control={control}
          link="/workers/"
          label="Primary Worker"
          rules={{ required: true }}
          placeholder="Select Primary Worker"
          description="Select the primary worker for the assignment."
          // TODO(wolfred): We need to change this to include the actual user permissions
          hasPermission
          hasPopoutWindow
          popoutLink="/shipments/configurations/workers/"
          popoutLinkLabel="Primary Worker"
          valueKey={["firstName", "lastName"]}
        />
      </FormControl>
      <FormControl>
        <AsyncSelectField
          isClearable
          name="secondaryWorkerId"
          control={control}
          link="/workers/"
          label="Secondary Worker"
          placeholder="Select Secondary Worker"
          description="Select the secondary worker for the assignment."
          // TODO(wolfred): We need to change this to include the actual user permissions
          hasPermission
          hasPopoutWindow
          popoutLink="/shipments/configurations/workers/"
          popoutLinkLabel="Secondary Worker"
          valueKey={["firstName", "lastName"]}
        />
      </FormControl>
    </FormGroup>
  );
}
