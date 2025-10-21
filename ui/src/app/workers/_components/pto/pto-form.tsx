import { AutoCompleteDateField } from "@/components/fields/date-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { WorkerAutocompleteField } from "@/components/ui/autocomplete-fields";
import { FormControl, FormGroup } from "@/components/ui/form";
import { ptoStatusChoices, ptoTypeChoices } from "@/lib/choices";
import { WorkerPTOSchema } from "@/lib/schemas/worker-schema";
import { useUser } from "@/stores/user-store";
import { PTOStatus } from "@/types/worker";
import { useEffect } from "react";
import { useFormContext, useWatch } from "react-hook-form";

const cancellationStatus = [PTOStatus.Rejected, PTOStatus.Cancelled];

export function PTOForm() {
  const user = useUser();
  const { control, setValue } = useFormContext<WorkerPTOSchema>();
  const status = useWatch({ control, name: "status" });

  useEffect(() => {
    if (status === PTOStatus.Approved) {
      setValue("approverId", user?.id);
    } else {
      setValue("approverId", undefined);
    }
  }, [status, setValue, user?.id]);

  return (
    <FormGroup cols={2}>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="status"
          options={ptoStatusChoices}
          placeholder="Status"
          label="Status"
          description="The status"
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="type"
          options={ptoTypeChoices}
          placeholder="Type"
          label="Type"
          description="The type"
        />
      </FormControl>
      <FormControl cols="full">
        <WorkerAutocompleteField<WorkerPTOSchema>
          name="workerId"
          control={control}
          label="Worker"
          rules={{ required: true }}
          placeholder="Select Worker"
          description="Select the worker for the PTO."
        />
      </FormControl>
      <FormControl>
        <AutoCompleteDateField
          name="startDate"
          control={control}
          rules={{ required: true }}
          label="Start Date"
          placeholder="Select start date"
          description="Indicates the start date of the PTO."
        />
      </FormControl>
      <FormControl>
        <AutoCompleteDateField
          name="endDate"
          control={control}
          rules={{ required: true }}
          label="End Date"
          placeholder="Select end date"
          description="Indicates the end date of the PTO."
        />
      </FormControl>
      {cancellationStatus.includes(status) && (
        <FormControl cols="full">
          <TextareaField
            control={control}
            rules={{ required: true }}
            name="reason"
            label="Reason"
            placeholder="Reason"
          />{" "}
        </FormControl>
      )}
    </FormGroup>
  );
}
