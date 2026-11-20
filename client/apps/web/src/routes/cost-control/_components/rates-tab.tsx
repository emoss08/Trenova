import { GLAccountMultiSelectAutocompleteField } from "@/components/autocomplete-fields";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import {
  SettingsNested,
  SettingsSection,
  SettingsSections,
} from "@/components/settings/settings-section";
import { costRateSourceChoices } from "@/lib/choices";
import type { CostBehavior, CostControlFormValues } from "@/types/cost-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatPerMile } from "@trenova/shared/lib/utils";
import { useFormContext, useWatch } from "react-hook-form";
import { useCostControlAccounting } from "../cost-control-tabs";

/**
 * What each cost category charges a mile, variable and fixed apart: whether it counts at
 * all, and whether its rate is the industry benchmark, the organization's own, or read from
 * posted expenses. Reading from the ledger is offered only while GL actuals are on.
 */
export function RatesTab() {
  const t = useT();
  const { control } = useFormContext<CostControlFormValues>();
  const categories = useWatch({ control, name: "categories" });
  const glActualsEnabled = useWatch({ control, name: "glActualsEnabled" });
  const glActive = useCostControlAccounting() && glActualsEnabled;

  const rows = (behavior: CostBehavior) =>
    categories.flatMap((category, index) =>
      category.costBehavior === behavior
        ? [<CategoryRows key={category.id} index={index} glActive={glActive} />]
        : [],
    );

  const variable = rows("Variable");
  const fixed = rows("Fixed");

  return (
    <SettingsSections>
      {variable.length > 0 ? (
        <SettingsSection
          title={t("Variable costs")}
          description={t("Costs that grow with every mile driven.")}
        >
          {variable}
        </SettingsSection>
      ) : null}
      {fixed.length > 0 ? (
        <SettingsSection
          title={t("Fixed costs")}
          description={t("Ownership and overhead, spread over the miles driven.")}
        >
          {fixed}
        </SettingsSection>
      ) : null}
    </SettingsSections>
  );
}

function CategoryRows({ index, glActive }: { index: number; glActive: boolean }) {
  const t = useT();
  const { control, getValues, setValue } = useFormContext<CostControlFormValues>();
  const category = useWatch({ control, name: `categories.${index}` });
  const benchmark = Number(category.benchmarkRatePerMile);

  const options = costRateSourceChoices.map((choice) =>
    choice.value === "GLActual" && !glActive ? { ...choice, disabled: true } : choice,
  );

  // Your own rate starts from the benchmark, and leaving it takes the rate with it.
  const onRateSourceChange = (value: string) => {
    const path = `categories.${index}.overrideRatePerMile` as const;
    if (value === "Override") {
      if (getValues(path) == null) setValue(path, benchmark, { shouldDirty: true });
      return;
    }
    if (getValues(path) != null) setValue(path, null, { shouldDirty: true });
  };

  return (
    <>
      <SwitchField
        control={control}
        name={`categories.${index}.isActive`}
        label={category.name}
        description={t("Industry benchmark {0}", formatPerMile(benchmark, 3))}
      />
      <SettingsNested show={category.isActive}>
        <SelectField
          control={control}
          name={`categories.${index}.rateSource`}
          label={t("Rate")}
          description={
            category.rateSource === "GLActual" && !glActive
              ? t("GL actuals are off, so this category uses the benchmark.")
              : t("Where this category's rate per mile comes from.")
          }
          options={options}
          onValueChange={onRateSourceChange}
          required
        />
        {category.rateSource === "Override" ? (
          <NumberField
            control={control}
            name={`categories.${index}.overrideRatePerMile`}
            label={t("Your rate")}
            description={t("What a mile costs you in this category.")}
            prefix="$"
            sideText={t("/mi")}
            decimalScale={3}
            min={0}
            max={100}
            required
          />
        ) : null}
        {category.rateSource === "GLActual" && glActive ? (
          <GLAccountMultiSelectAutocompleteField
            control={control}
            name={`categories.${index}.glAccountIds`}
            label={t("GL accounts")}
            placeholder={t("Select GL accounts")}
            description={t("The expense accounts whose postings make up this rate.")}
            required
          />
        ) : null}
      </SettingsNested>
    </>
  );
}
