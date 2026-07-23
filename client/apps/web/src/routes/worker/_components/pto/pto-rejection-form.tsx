import {
  TextareaField,
  type TextareaPreset,
} from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { useFormContext } from "react-hook-form";

const REJECTION_PRESETS: TextareaPreset[] = [
  {
    id: "worker-request",
    label: "Worker Request",
    description: "PTO rejected at worker's request",
  },
  {
    id: "business-request",
    label: "Business Request",
    description: "PTO rejected at business's request",
  },
  {
    id: "other",
    label: "Other",
    description: "Other reason",
  },
];

export function PTORejectionForm() {
  const { control } = useFormContext();

  return (
    <FormGroup className="pb-2" cols={1}>
      <FormControl cols="full">
        <TextareaField
          control={control}
          rules={{ required: true }}
          name="reason"
          label="Reason"
          placeholder="Reason"
          description="Provide a reason for rejecting the PTO."
          presets={REJECTION_PRESETS}
        />
      </FormControl>
    </FormGroup>
  );
}
