import { useT } from "@trenova/shared/i18n/use-t";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextChipsField } from "@/components/fields/text-chips-field";
import { Button } from "@trenova/shared/components/ui/button";
import { PlusIcon, Trash2Icon } from "lucide-react";
import { useFieldArray, useFormContext, useWatch } from "react-hook-form";
import {
  blankArgumentRule,
  needsValue,
  TOLERANCE_HELP,
  TOLERANCE_LABEL,
  toleranceKinds,
  type EvalCaseFormValues,
  type ToleranceKind,
} from "./eval-case-model";

type RulesPath = `tools.${number}.args` | `proposals.${number}.rules`;

/**
 * The arguments a call is checked on, one row each: the argument, how it is
 * compared, and what it is compared against. A proposal's rules take no value,
 * because the value is the parameter a person approved.
 */
export function ArgumentRulesField({ name, withValues }: { name: RulesPath; withValues: boolean }) {
  const t = useT();
  const { control } = useFormContext<EvalCaseFormValues>();
  const { fields, append, remove } = useFieldArray({ control, name });
  const kinds = withValues
    ? toleranceKinds
    : toleranceKinds.filter((kind) => kind !== "exact" && kind !== "present");

  return (
    <div className="flex flex-col gap-2">
      {fields.length === 0 ? (
        <p className="text-muted-foreground text-xs">
          {withValues
            ? t("No argument is checked; any call to the tool counts.")
            : t("Every parameter must match exactly.")}
        </p>
      ) : null}
      {fields.map((field, index) => (
        <ArgumentRuleRow
          key={field.id}
          name={`${name}.${index}`}
          kinds={kinds}
          withValues={withValues}
          onRemove={() => remove(index)}
        />
      ))}
      <div>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => append({ ...blankArgumentRule, kind: withValues ? "exact" : "numeric" })}
        >
          <PlusIcon />
          {withValues ? t("Check an argument") : t("Loosen a parameter")}
        </Button>
      </div>
    </div>
  );
}

function ArgumentRuleRow({
  name,
  kinds,
  withValues,
  onRemove,
}: {
  name: `${RulesPath}.${number}`;
  kinds: readonly ToleranceKind[];
  withValues: boolean;
  onRemove: () => void;
}) {
  const t = useT();
  const { control } = useFormContext<EvalCaseFormValues>();
  const kind = useWatch({ control, name: `${name}.kind` });

  return (
    <div className="border-border grid grid-cols-12 items-start gap-2 rounded-md border p-2">
      <div className="col-span-3">
        <InputField
          control={control}
          name={`${name}.key`}
          label={t("Argument")}
          placeholder={t("shipmentId")}
        />
      </div>
      <div className="col-span-3">
        <SelectField
          control={control}
          name={`${name}.kind`}
          label={t("Compared as")}
          description={kind ? t(TOLERANCE_HELP[kind]) : undefined}
          options={kinds.map((option) => ({ value: option, label: t(TOLERANCE_LABEL[option]) }))}
        />
      </div>
      <div className="col-span-5 flex flex-col gap-2">
        {withValues && kind && needsValue(kind) ? (
          <InputField
            control={control}
            name={`${name}.value`}
            label={kind === "setEq" ? t("Members") : t("Expected value")}
            placeholder={kind === "setEq" ? t("trc_1, trc_2") : t("1350")}
          />
        ) : null}
        {kind === "numeric" ? (
          <div className="grid grid-cols-2 gap-2">
            <InputField
              control={control}
              name={`${name}.abs`}
              label={t("Within")}
              placeholder={t("10")}
              inputMode="decimal"
            />
            <InputField
              control={control}
              name={`${name}.rel`}
              label={t("Or within share")}
              placeholder={t("0.02")}
              inputMode="decimal"
            />
          </div>
        ) : null}
        {kind === "dateWindow" ? (
          <InputField
            control={control}
            name={`${name}.windowHours`}
            label={t("Hours either side")}
            placeholder={t("24")}
            inputMode="decimal"
          />
        ) : null}
        {kind === "oneOf" ? (
          <TextChipsField
            control={control}
            name={`${name}.values`}
            label={t("Accepted values")}
            placeholder={t("Type a value and press Enter")}
          />
        ) : null}
      </div>
      <div className="col-span-1 flex justify-end pt-5">
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          aria-label={t("Remove this check")}
          onClick={onRemove}
        >
          <Trash2Icon />
        </Button>
      </div>
    </div>
  );
}
