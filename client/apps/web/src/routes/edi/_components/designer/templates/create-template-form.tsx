import { EDIDocumentTypeAutocompleteField } from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { type EDIDocumentType, type TemplateFormValues } from "@trenova/shared/types/edi";
import { useFormContext } from "react-hook-form";
import { functionalGroupForTransactionSet } from "../utils/edi-designer-utils";
import { templateStatusOptions } from "../utils/edi-designer-options";

export function CreateTemplateForm({
  disabled,
  mode = "create",
}: {
  disabled?: boolean;
  mode?: "create" | "edit";
}) {
  const { control, getValues, setValue } = useFormContext<TemplateFormValues>();

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
    <FormGroup cols={2} className="pb-2">
      {mode === "create" ? (
        <FormControl cols="full">
          <EDIDocumentTypeAutocompleteField<TemplateFormValues>
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
      ) : null}
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
      {mode === "create" ? (
        <>
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
        </>
      ) : (
        <FormControl cols="full">
          <SelectField
            control={control}
            rules={{ required: true }}
            name="status"
            label="Status"
            placeholder="Status"
            description="The template status."
            options={templateStatusOptions}
            isReadOnly={disabled}
          />
        </FormControl>
      )}
    </FormGroup>
  );
}
