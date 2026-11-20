import { GLAccountAutocompleteField } from "@/components/autocomplete-fields";
import { SwitchField } from "@/components/fields/switch-field";
import { SettingsSection, SettingsSections } from "@/components/settings/settings-section";
import type { CarrierSettlementControl } from "@/types/carrier-settlement-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext } from "react-hook-form";

/** Who approves a carrier settlement, what approving it does, and where it posts. */
export function ApprovalTab() {
  const t = useT();
  const { control } = useFormContext<CarrierSettlementControl>();

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Approval")}
        description={t("Who approves a carrier settlement and what happens once it is approved.")}
      >
        <SwitchField
          control={control}
          name="requireSeparateApprover"
          label={t("Require a separate approver")}
          description={t(
            "Whoever submits a carrier settlement cannot also approve it. Turn this off if one person runs carrier pay.",
          )}
        />
        <SwitchField
          control={control}
          name="autoPostOnApprove"
          label={t("Post to the ledger on approval")}
          description={t("Approving a settlement posts it to the general ledger in the same step.")}
        />
      </SettingsSection>

      <SettingsSection
        title={t("Ledger accounts")}
        description={t(
          "Where a carrier settlement posts. Left blank, the accounts on accounting control are used.",
        )}
      >
        <GLAccountAutocompleteField
          control={control}
          name="defaultApAccountId"
          label={t("Accounts payable account")}
          placeholder={t("Select AP account")}
          description={t("Credited when a settlement posts and debited when it is paid.")}
        />
        <GLAccountAutocompleteField
          control={control}
          name="defaultPurchasedTransportationAccountId"
          label={t("Purchased transportation account")}
          placeholder={t("Select expense account")}
          description={t("Debited for the carrier's cost when a settlement posts.")}
        />
      </SettingsSection>
    </SettingsSections>
  );
}
