import { ColorField } from "@/components/fields/color-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import {
  documentCategoryChoices,
  documentClassificationChoices,
} from "@/lib/choices";
import type { DocumentType } from "@/types/document-type";
import { useFormContext } from "react-hook-form";

export function DocumentTypeForm({ disabled }: { disabled?: boolean }) {
  const { control } = useFormContext<DocumentType>();

  return (
    <FormGroup cols={2}>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="code"
          label="Code"
          placeholder="Code"
          description="A unique code for this document type"
          maxLength={10}
          disabled={disabled}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="name"
          label="Name"
          placeholder="Name"
          description="The name of the document type"
          maxLength={100}
          disabled={disabled}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="documentClassification"
          label="Classification"
          placeholder="Classification"
          description="The classification level of documents"
          options={documentClassificationChoices}
          isReadOnly={disabled}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="documentCategory"
          label="Category"
          placeholder="Category"
          description="The category of documents"
          options={documentCategoryChoices}
          isReadOnly={disabled}
        />
      </FormControl>
      <FormControl cols="full">
        <ColorField
          control={control}
          name="color"
          label="Color"
          description="The color associated with this document type"
          disabled={disabled}
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="description"
          label="Description"
          placeholder="Description"
          description="A description of the document type"
          disabled={disabled}
        />
      </FormControl>
    </FormGroup>
  );
}
