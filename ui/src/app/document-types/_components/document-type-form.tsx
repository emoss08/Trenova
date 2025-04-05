import { ColorField } from "@/components/fields/color-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import {
  documentCategoryChoices,
  documentClassificationChoices,
} from "@/lib/choices";
import { DocumentTypeSchema } from "@/lib/schemas/document-type-schema";
import { useFormContext } from "react-hook-form";

export function DocumentTypeForm() {
  const { control } = useFormContext<DocumentTypeSchema>();
  return (
    <FormGroup cols={2}>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="code"
          label="Code"
          placeholder="Code"
          description="Unique identifier used for system reference and integration"
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="name"
          label="Name"
          placeholder="Name"
          description="Display name shown throughout the application interface"
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="description"
          label="Description"
          placeholder="Description"
          description="Detailed explanation of document purpose and usage context"
        />
      </FormControl>
      <FormControl cols="full">
        <ColorField
          control={control}
          name="color"
          label="Color"
          description="Visual identifier for quick recognition in document listings and reports"
        />
      </FormControl>
      <FormControl cols="full">
        <SelectField
          control={control}
          name="documentClassification"
          label="Document Classification"
          description="Security and access level designation for compliance requirements"
          options={documentClassificationChoices}
        />
      </FormControl>
      <FormControl cols="full">
        <SelectField
          control={control}
          name="documentCategory"
          label="Document Category"
          description="Functional grouping for organizational and reporting purposes"
          options={documentCategoryChoices}
        />
      </FormControl>
    </FormGroup>
  );
}
