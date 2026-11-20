import { GLAccountAutocompleteField } from "@/components/autocomplete-fields";
import { SettingsSection, SettingsSections } from "@/components/settings/settings-section";
import type { AccountingControl } from "@/types/accounting-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext } from "react-hook-form";

/**
 * The accounts a driver settlement posts to: what drivers earn, what they are owed and what
 * is held back from them. Driver pay costs in Cost Control come from these postings.
 */
export function DriverPayTab() {
  const t = useT();
  const { control } = useFormContext<AccountingControl>();

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Earnings")}
        description={t(
          "Where a settlement's pay is expensed. Cost per mile in Cost Control reads these postings, so it reflects what drivers are actually paid.",
        )}
      >
        <GLAccountAutocompleteField
          control={control}
          name="defaultDriverPayExpenseAccountId"
          label={t("Driver pay expense account")}
          placeholder={t("Select driver pay expense account")}
          description={t(
            "Debited for company drivers' earnings when a settlement posts. Settlements cannot post without it.",
          )}
        />
        <GLAccountAutocompleteField
          control={control}
          name="defaultPurchasedTransportationAccountId"
          label={t("Purchased transportation account")}
          placeholder={t("Select purchased transportation account")}
          description={t(
            "Debited for owner-operators' earnings instead of driver pay expense. Owner-operator settlements cannot post without it.",
          )}
        />
        <GLAccountAutocompleteField
          control={control}
          name="defaultDriverReimbursementAccountId"
          label={t("Driver reimbursement account")}
          placeholder={t("Select reimbursement account")}
          description={t(
            "Debited for non-taxable reimbursements such as per diem. Left empty, they post to driver pay expense.",
          )}
        />
      </SettingsSection>

      <SettingsSection
        title={t("Owed to and by drivers")}
        description={t("What a settlement leaves owing until it is paid, recovered or released.")}
      >
        <GLAccountAutocompleteField
          control={control}
          name="defaultSettlementsPayableAccountId"
          label={t("Settlements payable account")}
          placeholder={t("Select settlements payable account")}
          description={t(
            "Credited for a driver's net pay until the settlement is paid. Settlements cannot post without it.",
          )}
        />
        <GLAccountAutocompleteField
          control={control}
          name="defaultDriverAdvanceAccountId"
          label={t("Driver advance receivable account")}
          placeholder={t("Select advance receivable account")}
          description={t(
            "Tracks advances drivers have not yet repaid, and negative balances carried to the next settlement.",
          )}
        />
        <GLAccountAutocompleteField
          control={control}
          name="defaultEscrowLiabilityAccountId"
          label={t("Escrow liability account")}
          placeholder={t("Select escrow liability account")}
          description={t("Credited for escrow withheld from drivers' settlements.")}
        />
        <GLAccountAutocompleteField
          control={control}
          name="defaultEscrowInterestExpenseAccountId"
          label={t("Escrow interest expense account")}
          placeholder={t("Select escrow interest expense account")}
          description={t(
            "Expense account debited for interest credited to driver escrow balances.",
          )}
        />
        <GLAccountAutocompleteField
          control={control}
          name="defaultEscrowAdjustmentAccountId"
          label={t("Escrow adjustment account")}
          placeholder={t("Select escrow adjustment account")}
          description={t(
            "Offsets escrow liability for manual escrow adjustments, kept apart from purchased transportation so corrections stay visible and reconcilable; adjustments cannot be recorded without it.",
          )}
        />
      </SettingsSection>
    </SettingsSections>
  );
}
