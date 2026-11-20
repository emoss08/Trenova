import { NumberField } from "@/components/fields/number-field";
import { SwitchField } from "@/components/fields/switch-field";
import {
  SettingsNested,
  SettingsSection,
  SettingsSections,
} from "@/components/settings/settings-section";
import {
  checksNetPayVariance,
  SETTLEMENT_PERCENT_SCALE,
  type SettlementControl,
} from "@/types/settlement-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext, useWatch } from "react-hook-form";

/** Which settlements are held for a person to look at, and what approving one does. */
export function ReviewTab() {
  const t = useT();
  const { control } = useFormContext<SettlementControl>();
  const threshold = useWatch({ control, name: "varianceThresholdPct" });
  const checksVariance = checksNetPayVariance(threshold);

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Exceptions")}
        description={t(
          "A settlement whose net pay is far from the driver's recent average is flagged for review.",
        )}
      >
        <NumberField
          control={control}
          name="varianceThresholdPct"
          valueType="string"
          label={t("Flag a change in net pay over")}
          description={t(
            "Measured against the driver's average. Set it to 0 to stop flagging changes in pay.",
          )}
          placeholder="25"
          sideText="%"
          decimalScale={SETTLEMENT_PERCENT_SCALE}
          min={0}
          max={999.9999}
          required
        />
        <SettingsNested show={checksVariance}>
          <NumberField
            control={control}
            name="varianceLookbackWeeks"
            label={t("Average over")}
            description={t("How many of the driver's previous settlements make up the average.")}
            placeholder="8"
            sideText={t("settlements")}
            min={1}
            max={52}
            required
          />
        </SettingsNested>
      </SettingsSection>

      <SettingsSection
        title={t("Approval")}
        description={t("Who approves a settlement and what happens once it is approved.")}
      >
        <SwitchField
          control={control}
          name="autoApproveClean"
          label={t("Approve clean settlements automatically")}
          description={t("A settlement with no exceptions is approved without waiting for review.")}
        />
        <SwitchField
          control={control}
          name="requireSeparateApprover"
          label={t("Require a separate approver")}
          description={t(
            "Whoever submits a settlement cannot also approve it. Turn this off if one person runs payroll.",
          )}
        />
        <SwitchField
          control={control}
          name="autoPostOnApprove"
          label={t("Post to the ledger on approval")}
          description={t("Approving a settlement posts it to the general ledger in the same step.")}
        />
      </SettingsSection>
    </SettingsSections>
  );
}
