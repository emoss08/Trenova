import { SwitchField } from "@/components/fields/switch-field";
import {
  SettingsNested,
  SettingsSection,
  SettingsSections,
} from "@/components/settings/settings-section";
import type { DashControl } from "@/types/dash-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext, useWatch } from "react-hook-form";

/** What drivers see of their pay before settlement, and the money they can claim or question. */
export function PayTab() {
  const t = useT();
  const { control } = useFormContext<DashControl>();
  const showsLoadPay = useWatch({ control, name: "showLoadPay" });
  const takesExpenses = useWatch({ control, name: "allowExpenseSubmission" });

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Pay")}
        description={t(
          "Drivers always see their settlement statements; these show pay before a load is settled.",
        )}
      >
        <SwitchField
          control={control}
          name="showLoadPay"
          label={t("Pay per load")}
          description={t("Drivers see what each load pays, and their pay as it is earned.")}
        />
        <SettingsNested show={showsLoadPay}>
          <SwitchField
            control={control}
            name="showPayEstimates"
            label={t("Pay estimates")}
            description={t(
              "Drivers see an estimate from their pay plan on a load before its pay is earned.",
            )}
          />
        </SettingsNested>
      </SettingsSection>

      <SettingsSection
        title={t("Expenses and disputes")}
        description={t("Money drivers claim back, and settlements they question.")}
      >
        <SwitchField
          control={control}
          name="allowExpenseSubmission"
          label={t("Expense claims")}
          description={t(
            "Drivers claim what they paid out of pocket, such as lumpers and tolls, for reimbursement.",
          )}
        />
        <SettingsNested show={takesExpenses}>
          <SwitchField
            control={control}
            name="requireExpenseReceipt"
            label={t("Require a receipt")}
            description={t("A claim cannot be approved until a photo of its receipt is attached.")}
          />
        </SettingsNested>
        <SwitchField
          control={control}
          name="allowSettlementDisputes"
          label={t("Settlement disputes")}
          description={t("Drivers can question a statement, or one line of it, from Dash.")}
        />
      </SettingsSection>
    </SettingsSections>
  );
}
