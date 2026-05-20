import { EDIDocumentTypeAutocompleteField } from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { createTemplateDraftSchema, type EDIDocumentType, ediStandardSchema } from "@/types/edi";
import { useFormContext } from "react-hook-form";
import { z } from "zod";
import { functionalGroupForTransactionSet } from "../utils/edi-designer-utils";

const createTemplateFormSchema = createTemplateDraftSchema.extend({
  documentTypeId: z.string().min(1, "Document type is required"),
  name: z.string().trim().min(1, "Name is required"),
  standard: ediStandardSchema.default("X12"),
  x12Version: z.string().trim().min(1, "X12 version is required"),
  functionalGroupId: z.string().trim().min(1, "Functional group is required"),
});

export type CreateTemplateFormValues = z.infer<typeof createTemplateFormSchema>;

export { createTemplateFormSchema };

export function CreateTemplateForm({ disabled }: { disabled?: boolean }) {
  const { control, getValues, setValue } = useFormContext<CreateTemplateFormValues>();

  const handleDocumentTypeChange = (documentType: EDIDocumentType | null) => {
    if (!documentType) return;
    setValue("direction", documentType.direction, { shouldDirty: true, shouldValidate: true });
    setValue("transactionSet", documentType.transactionSet, {
      shouldDirty: true,
      shouldValidate: true,
    });
    setValue("x12Version", documentType.defaultVersion || getValues("x12Version"), {
      shouldDirty: true,
      shouldValidate: true,
    });
    setValue("functionalGroupId", functionalGroupForTransactionSet(documentType.transactionSet), {
      shouldDirty: true,
      shouldValidate: true,
    });
  };

  return (
    <FormGroup cols={2}>
      <FormControl cols="full">
        <EDIDocumentTypeAutocompleteField<CreateTemplateFormValues>
          control={control}
          name="documentTypeId"
          label="Document Type"
          rules={{ required: true }}
          clearable
          disabled={disabled}
          description="The EDI document type that seeds the template direction and transaction set."
          placeholder="Document Type"
          onOptionChange={handleDocumentTypeChange}
        />
      </FormControl>
      <FormControl cols="full">
        <InputField
          control={control}
          rules={{ required: true }}
          name="name"
          label="Name"
          placeholder="Template name"
          description="A clear internal name for this EDI template."
          disabled={disabled}
        />
      </FormControl>
      <FormControl cols="full">
        <InputField
          control={control}
          name="description"
          label="Description"
          placeholder="Description"
          description="Optional context for where this template should be used."
          disabled={disabled}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="x12Version"
          label="X12 Version"
          placeholder="004010"
          description="The X12 version for the first draft."
          disabled={disabled}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="functionalGroupId"
          label="Group"
          placeholder="SM"
          description="The functional group identifier."
          disabled={disabled}
        />
      </FormControl>
    </FormGroup>
  );
}
