import { useT } from "@trenova/shared/i18n/use-t";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { accessorialChargeMethodChoices, rateUnitChoices, statusChoices } from "@/lib/choices";
import type { AccessorialCharge } from "@trenova/shared/types/accessorial-charge";
import { useEffect } from "react";
import { useFormContext, useWatch } from "react-hook-form";

const amountDescriptions: Record<AccessorialCharge["method"], string> = {
  Flat: "The fixed dollar amount charged",
  PerUnit: "The rate per unit (e.g., per hour, per mile)",
  Percentage: "The percentage applied to the linehaul rate",
};

function getAmountSideText(method: AccessorialCharge["method"], rateUnit?: string): string {
  switch (method) {
    case "Flat":
      return "$";
    case "Percentage":
      return "%";
    case "PerUnit":
      return rateUnit ? `$/${rateUnit}` : "$/Unit";
    default:
      return "$";
  }
}

export function AccessorialChargeForm() {
  const t = useT();

  const { control, setValue } = useFormContext<AccessorialCharge>();
  const method = useWatch({ name: "method" });
  const rateUnit = useWatch({ name: "rateUnit" });

  const methodIsPerUnit = method === "PerUnit";

  useEffect(() => {
    if (!methodIsPerUnit) {
      setValue("rateUnit", undefined);
    }
  }, [methodIsPerUnit, setValue]);

  return (
    <FormGroup cols={2}>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="status"
          label={t("Status")}
          placeholder={t("Status")}
          description={t("Current processing status of this accessorial charge (active, pending approval, etc.)")}
          options={statusChoices}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="code"
          label={t("Code")}
          placeholder={t("Code")}
          description={t("Standard industry or company-specific code identifying this accessorial service (e.g., LUM for lumper fee)")}
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          rules={{ required: true }}
          name="description"
          label={t("Description")}
          placeholder={t("Description")}
          description={t("Detailed explanation of the accessorial service provided, including any special conditions or requirements for FMCSA compliance")}
        />
      </FormControl>
      <FormControl cols={methodIsPerUnit ? 1 : "full"}>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="method"
          label={t("Method")}
          placeholder={t("Method")}
          description={t("Calculation method for this charge (flat rate, per mile, percentage of linehaul, etc.)")}
          options={accessorialChargeMethodChoices}
        />
      </FormControl>
      {methodIsPerUnit && (
        <FormControl>
          <SelectField
            control={control}
            rules={{ required: methodIsPerUnit }}
            name="rateUnit"
            label={t("Rate Unit")}
            placeholder={t("Rate Unit")}
            description={t("Unit of measure for this charge (mile, hour, day, stop)")}
            options={rateUnitChoices}
          />
        </FormControl>
      )}
      <FormControl cols="full">
        <NumberField
          control={control}
          rules={{ required: true }}
          name="amount"
          label={t("Amount")}
          placeholder={t("Amount")}
          sideText={getAmountSideText(method as AccessorialCharge["method"], rateUnit)}
          decimalScale={4}
          thousandSeparator
          description={
            amountDescriptions[method as AccessorialCharge["method"]] ?? amountDescriptions.Flat
          }
        />
      </FormControl>
    </FormGroup>
  );
}
