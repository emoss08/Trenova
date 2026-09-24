import {
  CarrierAutocompleteField,
  CustomerAutocompleteField,
  LocationAutocompleteField,
  WorkerAutocompleteField,
} from "@/components/autocomplete-fields";
import { DateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { useEffect, useRef } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { memoryKindChoices } from "../activity/agent-badges";
import { MEMORY_CONTENT_LIMIT, type MemoryFormValues } from "./memory-form-schema";

/**
 * One memory, in the words a person would use: what to know, whether it is a
 * rule or a fact, and which record it is about. The record picker follows
 * the chosen kind of record, and changing the kind clears the pick, because
 * a customer's id is not a driver's.
 */
export function MemoryForm() {
  const t = useT();
  const { control, setValue } = useFormContext<MemoryFormValues>();
  const subjectType = useWatch({ control, name: "subjectType" });
  const previousSubjectType = useRef(subjectType);

  useEffect(() => {
    if (previousSubjectType.current !== subjectType) {
      previousSubjectType.current = subjectType;
      setValue("subjectId", null, { shouldDirty: true });
    }
  }, [setValue, subjectType]);

  return (
    <>
      <FormSection title={t("What to remember")} className="pb-4">
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              name="kind"
              control={control}
              label={t("Kind")}
              options={memoryKindChoices.map((choice) => ({
                value: choice.value,
                label: t(choice.label),
              }))}
              description={t("An instruction is followed; a fact is weighed.")}
            />
          </FormControl>
          <FormControl>
            <DateField
              name="expiresAt"
              control={control}
              label={t("Until")}
              placeholder={t("No end")}
              description={t("After this day agents stop reading it.")}
              clearable
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              name="content"
              control={control}
              label={t("Memory")}
              rows={4}
              maxLength={MEMORY_CONTENT_LIMIT}
              placeholder={t("Acme Freight needs the signed POD within one day of delivery.")}
              description={t(
                "One or two plain sentences a reader with no other context understands.",
              )}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("What it is about")}
        description={t("Leave both empty for something every agent should know.")}
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              name="subjectType"
              control={control}
              label={t("Kind of record")}
              placeholder={t("The whole organization")}
              isClearable
              options={[
                { value: "Customer", label: t("Customer") },
                { value: "Location", label: t("Location") },
                { value: "Worker", label: t("Driver") },
                { value: "Carrier", label: t("Carrier") },
              ]}
            />
          </FormControl>
          <FormControl>
            <SubjectPicker subjectType={subjectType} />
          </FormControl>
          <FormControl cols="full">
            <InputField
              name="toolName"
              control={control}
              label={t("Tool")}
              placeholder={t("assign_move")}
              description={t(
                "Optional. Only agents holding this tool read it, such as a correction to how it was proposed.",
              )}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </>
  );
}

function SubjectPicker({ subjectType }: { subjectType: MemoryFormValues["subjectType"] }) {
  const t = useT();
  const { control } = useFormContext<MemoryFormValues>();

  switch (subjectType) {
    case "Customer":
      return (
        <CustomerAutocompleteField
          control={control}
          name="subjectId"
          label={t("Customer")}
          placeholder={t("Pick a customer")}
          clearable
        />
      );
    case "Location":
      return (
        <LocationAutocompleteField
          control={control}
          name="subjectId"
          label={t("Location")}
          placeholder={t("Pick a location")}
          clearable
        />
      );
    case "Worker":
      return (
        <WorkerAutocompleteField
          control={control}
          name="subjectId"
          label={t("Driver")}
          placeholder={t("Pick a driver")}
          clearable
        />
      );
    case "Carrier":
      return (
        <CarrierAutocompleteField
          control={control}
          name="subjectId"
          label={t("Carrier")}
          placeholder={t("Pick a carrier")}
          clearable
        />
      );
    default:
      return (
        <InputField
          name="subjectId"
          control={control}
          label={t("Record")}
          placeholder={t("Choose a kind of record first")}
          disabled
        />
      );
  }
}
