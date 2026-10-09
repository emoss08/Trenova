import {
  CarrierAutocompleteField,
  CustomerAutocompleteField,
  LocationAutocompleteField,
  WorkerAutocompleteField,
} from "@/components/autocomplete-fields";
import { DateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { SegmentedField } from "@/components/fields/segmented-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import type { AgentMemorySubjectType } from "@trenova/graphql/generated/graphql";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { useController, useFormContext, useWatch } from "react-hook-form";
import { aicFieldTrigger } from "../edit/field-trigger";
import { MEMORY_CONTENT_LIMIT, type MemoryFormValues } from "./memory-form-schema";
import { MEMORY_KIND_LABELS, MEMORY_KINDS } from "./memory-kind";

const SUBJECT_LABELS: Record<AgentMemorySubjectType, string> = defineLabels({
  Customer: "A customer",
  Location: "A location",
  Worker: "A driver",
  Carrier: "A carrier",
});

/** "Every agent" is no subject at all; the select spells it as the empty choice. */
const EVERY_AGENT = "";

const SUBJECT_OPTIONS = [
  { value: EVERY_AGENT, label: "Every agent" },
  ...(Object.keys(SUBJECT_LABELS) as AgentMemorySubjectType[]).map((value) => ({
    value,
    label: SUBJECT_LABELS[value],
  })),
];

/** What to remember, in one or two plain sentences, and what kind of memory it is. */
export function MemoryText() {
  const t = useT();
  const { control } = useFormContext<MemoryFormValues>();
  const content = useWatch({ control, name: "content" });

  return (
    <div className="flex flex-col gap-3">
      <TextareaField<MemoryFormValues>
        control={control}
        name="content"
        rules={{ required: true }}
        label={t("Memory")}
        placeholder={t("What should the agents know?")}
        autoFocus
        maxLength={MEMORY_CONTENT_LIMIT}
        description={`${content.length}/${MEMORY_CONTENT_LIMIT}`}
      />
      <SegmentedField<MemoryFormValues, (typeof MEMORY_KINDS)[number]>
        control={control}
        name="kind"
        label={t("Kind")}
        options={MEMORY_KINDS.map((value) => ({
          value,
          label: t(MEMORY_KIND_LABELS[value]),
        }))}
      />
    </div>
  );
}

/** Which record the memory is about, and the tool it is limited to; neither for every agent. */
export function MemoryAbout() {
  const t = useT();
  const { control, setValue } = useFormContext<MemoryFormValues>();
  const subjectType = useController({ control, name: "subjectType" });

  return (
    <>
      <SelectField<MemoryFormValues>
        control={control}
        name="subjectType"
        label={t("About")}
        description={t("A customer's rule, a dock's hours, a driver's preference.")}
        options={SUBJECT_OPTIONS}
        placeholder={t("Every agent")}
        triggerClassName={aicFieldTrigger}
        onValueChange={(value) => {
          if (value === EVERY_AGENT) {
            setValue("subjectType", null, { shouldDirty: true, shouldValidate: true });
          }
          setValue("subjectId", null, { shouldDirty: true, shouldValidate: true });
        }}
      />
      {subjectType.field.value && <SubjectPicker subjectType={subjectType.field.value} />}
      <InputField<MemoryFormValues>
        control={control}
        name="toolName"
        label={t("Tool")}
        description={t(
          "Optional. Only agents holding this tool read it, such as a correction to how it is used.",
        )}
        placeholder="assign_move"
        inputClassProps={aicFieldTrigger}
      />
    </>
  );
}

/** The day agents stop reading the memory, or none. */
export function MemoryUntil() {
  const t = useT();
  const { control } = useFormContext<MemoryFormValues>();

  return (
    <DateField
      name="expiresAt"
      control={control}
      label={t("Until")}
      placeholder={t("No end")}
      clearable
    />
  );
}

function SubjectPicker({ subjectType }: { subjectType: AgentMemorySubjectType }) {
  const t = useT();
  const { control } = useFormContext<MemoryFormValues>();

  switch (subjectType) {
    case "Customer":
      return (
        <CustomerAutocompleteField
          control={control}
          name="subjectId"
          rules={{ required: true }}
          label={t("Record")}
          placeholder={t("Which customer?")}
          triggerClassName={aicFieldTrigger}
          clearable
        />
      );
    case "Location":
      return (
        <LocationAutocompleteField
          control={control}
          name="subjectId"
          rules={{ required: true }}
          label={t("Record")}
          placeholder={t("Which location?")}
          triggerClassName={aicFieldTrigger}
          clearable
        />
      );
    case "Worker":
      return (
        <WorkerAutocompleteField
          control={control}
          name="subjectId"
          rules={{ required: true }}
          label={t("Record")}
          placeholder={t("Which driver?")}
          triggerClassName={aicFieldTrigger}
          clearable
        />
      );
    case "Carrier":
      return (
        <CarrierAutocompleteField
          control={control}
          name="subjectId"
          rules={{ required: true }}
          label={t("Record")}
          placeholder={t("Which carrier?")}
          triggerClassName={aicFieldTrigger}
          clearable
        />
      );
  }
}
