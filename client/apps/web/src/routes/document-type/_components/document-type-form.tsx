import { useT } from "@trenova/shared/i18n/use-t";
import { ColorField } from "@/components/fields/color-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { documentCategoryChoices, documentClassificationChoices } from "@/lib/choices";
import type { DocumentType } from "@trenova/shared/types/document-type";
import { useFormContext } from "react-hook-form";

export function DocumentTypeForm({ disabled }: { disabled?: boolean }) {
  const t = useT();

  const { control } = useFormContext<DocumentType>();

  return (
    <FormGroup cols={2}>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="code"
          label={t("Code")}
          placeholder={t("Code")}
          description={t("A unique code for this document type")}
          maxLength={10}
          disabled={disabled}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="name"
          label={t("Name")}
          placeholder={t("Name")}
          description={t("The name of the document type")}
          maxLength={100}
          disabled={disabled}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="documentClassification"
          label={t("Classification")}
          placeholder={t("Classification")}
          description={t("The classification level of documents")}
          options={documentClassificationChoices}
          isReadOnly={disabled}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="documentCategory"
          label={t("Category")}
          placeholder={t("Category")}
          description={t("The category of documents")}
          options={documentCategoryChoices}
          isReadOnly={disabled}
        />
      </FormControl>
      <FormControl cols="full">
        <ColorField
          control={control}
          name="color"
          label={t("Color")}
          description={t("The color associated with this document type")}
          disabled={disabled}
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="description"
          label={t("Description")}
          placeholder={t("Description")}
          description={t("A description of the document type")}
          disabled={disabled}
        />
      </FormControl>
    </FormGroup>
  );
}
