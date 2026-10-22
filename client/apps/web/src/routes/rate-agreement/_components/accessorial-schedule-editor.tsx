import { useT } from "@trenova/shared/i18n/use-t";
import { AccessorialChargeAutocompleteField } from "@/components/autocomplete-fields";
import { TariffEditorField } from "@/components/formula-editor/tariff-editor";
import { useTariffCheckContext } from "@/hooks/use-tariff-check-context";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { accessorialChargeMethodChoices, rateUnitChoices } from "@/lib/choices";
import { Button } from "@trenova/shared/components/ui/button";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import type { RateAgreement, RateAgreementAccessorial } from "@trenova/shared/types/rate";
import { PlusIcon } from "@trenova/shared/components/icons";
import { useFieldArray, useFormContext, useWatch } from "react-hook-form";

const NEW_ACCESSORIAL = {
  accessorialChargeId: "",
  method: "Flat",
  amount: null,
  waived: false,
  autoApply: false,
  condition: null,
} as unknown as RateAgreementAccessorial;

/** What a new condition starts as: a condition file's header and the expression to write. */
const NEW_CONDITION = `condition "Applies when"

shipment.hazmat
`;

/**
 * The contract's own prices for accessorial services.
 *
 * This is what makes a rate confirmation and an invoice agree: both read the
 * price from here rather than one reading the organization default and the
 * other whatever a clerk typed. A row marked to apply automatically becomes a
 * system charge on every shipment the contract prices, so it never depends on
 * somebody remembering.
 */
export function AccessorialScheduleEditor() {
  const t = useT();

  const { control } = useFormContext<RateAgreement>();
  const { fields, append, remove } = useFieldArray({ control, name: "accessorials" });
  const accessorials = (useWatch({ control, name: "accessorials" }) ??
    []) as RateAgreementAccessorial[];

  return (
    <div className="flex flex-col gap-3">
      {fields.length === 0 && (
        <p className="text-muted-foreground text-sm">
          {t(
            "No negotiated accessorials. Without one, every accessorial is priced at the organization default instead of at what the contract says.",
          )}
        </p>
      )}

      {fields.map((field, index) => {
        const row = accessorials[index];
        const isPerUnit = row?.method === "PerUnit";
        const autoApplies = Boolean(row?.autoApply);

        return (
          <div key={field.id} className="bg-card flex flex-col gap-4 rounded-md border p-4">
            <div className="flex items-center justify-between">
              <p className="text-sm font-medium">
                {t("Accessorial")} {index + 1}
                {row?.waived && (
                  <span className="text-muted-foreground ml-2 text-xs font-normal">
                    {t("Waived")}
                  </span>
                )}
              </p>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="h-6 text-xs"
                onClick={() => remove(index)}
              >
                {t("Remove")}
              </Button>
            </div>

            <FormGroup cols={3}>
              <FormControl>
                <AccessorialChargeAutocompleteField
                  control={control}
                  rules={{ required: true }}
                  name={`accessorials.${index}.accessorialChargeId` as never}
                  label={t("Accessorial")}
                  placeholder={t("Charge")}
                  description={t("Which service this price is for")}
                />
              </FormControl>
              <FormControl>
                <SelectField
                  control={control}
                  rules={{ required: true }}
                  name={`accessorials.${index}.method` as never}
                  label={t("Method")}
                  placeholder={t("Method")}
                  description={t("Flat, per unit, or a percentage")}
                  options={accessorialChargeMethodChoices}
                />
              </FormControl>
              {isPerUnit ? (
                <FormControl>
                  <SelectField
                    control={control}
                    rules={{ required: true }}
                    name={`accessorials.${index}.rateUnit` as never}
                    label={t("Rate unit")}
                    placeholder={t("Unit")}
                    description={t("What a unit is on this charge")}
                    options={rateUnitChoices}
                  />
                </FormControl>
              ) : (
                <FormControl>
                  <NumberField
                    control={control}
                    name={`accessorials.${index}.amount` as never}
                    label={t("Amount")}
                    placeholder="75.00"
                    sideText="$"
                    decimalScale={2}
                    thousandSeparator
                    description={t("What the contract charges")}
                  />
                </FormControl>
              )}
            </FormGroup>

            {isPerUnit && (
              <FormGroup cols={3}>
                <FormControl>
                  <NumberField
                    control={control}
                    name={`accessorials.${index}.amount` as never}
                    label={t("Amount per unit")}
                    placeholder="65.00"
                    sideText="$"
                    decimalScale={2}
                    thousandSeparator
                    description={t("What the contract charges for each unit")}
                  />
                </FormControl>
                <FormControl>
                  <NumberField
                    control={control}
                    name={`accessorials.${index}.freeUnits` as never}
                    label={t("Free units")}
                    placeholder="2"
                    description={t("Units the contract gives away before charging")}
                  />
                </FormControl>
                <FormControl>
                  <NumberField
                    control={control}
                    name={`accessorials.${index}.maxAmount` as never}
                    label={t("Maximum amount")}
                    placeholder="0.00"
                    sideText="$"
                    decimalScale={2}
                    thousandSeparator
                    description={t("A ceiling on what this charge can reach")}
                  />
                </FormControl>
              </FormGroup>
            )}

            <FormGroup cols={2}>
              <FormControl>
                <SwitchField
                  control={control}
                  name={`accessorials.${index}.autoApply` as never}
                  label={t("Auto apply")}
                  description={t(
                    "Added to every shipment this contract prices, so it never depends on somebody remembering",
                  )}
                  outlined
                />
              </FormControl>
              <FormControl>
                <SwitchField
                  control={control}
                  name={`accessorials.${index}.waived` as never}
                  label={t("Waived")}
                  description={t(
                    "The contract gives this service away, which is a stated term rather than an omission",
                  )}
                  outlined
                />
              </FormControl>
            </FormGroup>

            {autoApplies && (
              <AccessorialCondition index={index} hasCondition={Boolean(row?.condition)} />
            )}
          </div>
        );
      })}

      <Button type="button" variant="outline" size="sm" onClick={() => append(NEW_ACCESSORIAL)}>
        <PlusIcon className="mr-1 size-3.5" />
        {t("Add accessorial")}
      </Button>
    </div>
  );
}

/**
 * When an automatic accessorial applies: a Tariff `condition` file whose last line is the
 * true-or-false rule, checked against each shipment the contract prices. Without one the
 * charge applies to every shipment.
 */
function AccessorialCondition({ index, hasCondition }: { index: number; hasCondition: boolean }) {
  const t = useT();
  const { control, setValue } = useFormContext<RateAgreement>();
  const { context } = useTariffCheckContext();
  const name = `accessorials.${index}.condition` as const;

  if (!hasCondition) {
    return (
      <div className="flex items-center justify-between gap-3 rounded-md border border-dashed px-3 py-2">
        <p className="text-muted-foreground text-xs">
          {t("Applies to every shipment this contract prices.")}
        </p>
        <Button
          type="button"
          variant="outline"
          size="xs"
          onClick={() => setValue(name, NEW_CONDITION, { shouldDirty: true })}
        >
          {t("Add condition")}
        </Button>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-1">
      <TariffEditorField
        control={control}
        name={name as never}
        kind="condition"
        context={context}
        variant="mini"
        label={t("Applies when")}
        description={t(
          "A Tariff condition: inputs and bindings, then one true-or-false line. The charge applies only to the shipments it holds for.",
        )}
        ariaLabel={t("Condition for accessorial {0}", index + 1)}
      />
      <Button
        type="button"
        variant="ghost"
        size="xs"
        className="text-muted-foreground self-start"
        onClick={() => setValue(name, null, { shouldDirty: true })}
      >
        {t("Remove condition")}
      </Button>
    </div>
  );
}
