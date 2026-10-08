import {
  CarrierAutocompleteField,
  CustomerAutocompleteField,
  LocationAutocompleteField,
  WorkerAutocompleteField,
} from "@/components/autocomplete-fields";
import { DateField } from "@/components/fields/date-field/date-field";
import type { AgentMemorySubjectType } from "@trenova/graphql/generated/graphql";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { useController, useFormContext } from "react-hook-form";
import { F, Sel, Txt } from "../edit/fields";
import { Seg } from "../kit/layout";
import { MEMORY_CONTENT_LIMIT, type MemoryFormValues } from "./memory-form-schema";
import { MEMORY_KIND_LABELS, MEMORY_KINDS } from "./memory-kind";

const SUBJECT_LABELS: Record<AgentMemorySubjectType, string> = defineLabels({
  Customer: "A customer",
  Location: "A location",
  Worker: "A driver",
  Carrier: "A carrier",
});

const EVERY_AGENT = "";

/** What to remember, in one or two plain sentences, and what kind of memory it is. */
export function MemoryText() {
  const t = useT();
  const { control } = useFormContext<MemoryFormValues>();
  const content = useController({ control, name: "content" });
  const kind = useController({ control, name: "kind" });

  return (
    <div className="cmp-m">
      <textarea
        className="mta"
        aria-label={t("Memory")}
        placeholder={t("What should the agents know?")}
        autoFocus
        rows={4}
        value={content.field.value}
        onBlur={content.field.onBlur}
        onChange={(event) =>
          content.field.onChange(event.target.value.slice(0, MEMORY_CONTENT_LIMIT))
        }
      />
      <div className="cmp-r">
        <Seg
          className="sm"
          label={t("Kind")}
          v={kind.field.value}
          opts={MEMORY_KINDS.map((value) => [value, t(MEMORY_KIND_LABELS[value])] as const)}
          onChange={kind.field.onChange}
        />
        <span className="sp" />
        <span className="cmp-n mono">
          {content.field.value.length}/{MEMORY_CONTENT_LIMIT}
        </span>
      </div>
      {content.fieldState.error && <p className="f-h t-w">{content.fieldState.error.message}</p>}
    </div>
  );
}

/** Which record the memory is about, and the tool it is limited to; neither for every agent. */
export function MemoryAbout() {
  const t = useT();
  const { control, setValue } = useFormContext<MemoryFormValues>();
  const subjectType = useController({ control, name: "subjectType" });
  const subjectId = useController({ control, name: "subjectId" });
  const toolName = useController({ control, name: "toolName" });

  const options = [
    [EVERY_AGENT, t("Every agent")] as const,
    ...(Object.keys(SUBJECT_LABELS) as AgentMemorySubjectType[]).map(
      (value) => [value, t(SUBJECT_LABELS[value])] as const,
    ),
  ];

  return (
    <>
      <F label={t("About")} hint={t("A customer's rule, a dock's hours, a driver's preference.")}>
        <Sel
          label={t("About")}
          value={subjectType.field.value ?? EVERY_AGENT}
          options={options}
          onChange={(value) => {
            subjectType.field.onChange(value === EVERY_AGENT ? null : value);
            setValue("subjectId", null, { shouldDirty: true, shouldValidate: true });
          }}
        />
      </F>
      {subjectType.field.value && (
        <F label={t("Record")} error={subjectId.fieldState.error?.message}>
          <SubjectPicker subjectType={subjectType.field.value} />
        </F>
      )}
      <F
        label={t("Tool")}
        hint={t(
          "Optional. Only agents holding this tool read it, such as a correction to how it is used.",
        )}
      >
        <Txt
          mono
          label={t("Tool")}
          value={toolName.field.value}
          placeholder="assign_move"
          onChange={toolName.field.onChange}
        />
      </F>
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
          placeholder={t("Which customer?")}
          clearable
        />
      );
    case "Location":
      return (
        <LocationAutocompleteField
          control={control}
          name="subjectId"
          placeholder={t("Which location?")}
          clearable
        />
      );
    case "Worker":
      return (
        <WorkerAutocompleteField
          control={control}
          name="subjectId"
          placeholder={t("Which driver?")}
          clearable
        />
      );
    case "Carrier":
      return (
        <CarrierAutocompleteField
          control={control}
          name="subjectId"
          placeholder={t("Which carrier?")}
          clearable
        />
      );
  }
}
