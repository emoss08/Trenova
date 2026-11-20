import { MoneyField } from "@/components/fields/money-field";
import { SwitchField } from "@/components/fields/switch-field";
import {
  SettingsNested,
  SettingsSection,
  SettingsSections,
} from "@/components/settings/settings-section";
import { PlanFeature } from "@/lib/plan-feature";
import type { CarrierSettlementControl } from "@/types/carrier-settlement-control";
import { usePlanFeature } from "@trenova/shared/hooks/use-plan-feature";
import { useT } from "@trenova/shared/i18n/use-t";
import { useEffect } from "react";
import { useFormContext, useWatch } from "react-hook-form";

/**
 * How a carrier's invoice is checked against the buy rate, and whether pay waits for it.
 * Matching invoices as they arrive is offered only on a plan with EDI, the only way they
 * arrive unasked; the saved settings still round-trip without it.
 */
export function MatchingTab() {
  const t = useT();
  const { control, getValues, setValue } = useFormContext<CarrierSettlementControl>();
  const edi = usePlanFeature(PlanFeature.EDI).available;
  const autoMatch = useWatch({ control, name: "autoMatchInboundInvoices" });

  useEffect(() => {
    if (!autoMatch && getValues("autoAcceptWithinTolerance")) {
      setValue("autoAcceptWithinTolerance", false, { shouldDirty: true });
    }
  }, [autoMatch, getValues, setValue]);

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Matching")}
        description={t("Each carrier invoice is compared with the buy rate agreed for the load.")}
      >
        <MoneyField
          control={control}
          name="varianceToleranceMinor"
          label={t("Variance tolerance")}
          description={t(
            "An invoice within this amount of the buy rate matches; a larger gap waits for review.",
          )}
          showZero
        />
        {edi ? (
          <>
            <SwitchField
              control={control}
              name="autoMatchInboundInvoices"
              label={t("Match EDI invoices as they arrive")}
              description={t(
                "Invoices carriers send by EDI are matched to their load without anyone starting it.",
              )}
            />
            <SettingsNested show={autoMatch}>
              <SwitchField
                control={control}
                name="autoAcceptWithinTolerance"
                label={t("Accept matches within the tolerance")}
                description={t(
                  "A match within the tolerance is accepted without review, and its cost joins the carrier's settlement.",
                )}
              />
            </SettingsNested>
          </>
        ) : null}
      </SettingsSection>

      <SettingsSection
        title={t("Paying on a matched invoice")}
        description={t("Whether a carrier is paid for a load before its invoice is matched.")}
      >
        <SwitchField
          control={control}
          name="holdUntilInvoiceMatched"
          label={t("Hold pay until the invoice is matched")}
          description={t(
            "A load's cost stays out of settlements until its invoice is matched, and a settlement holding an unmatched load cannot be approved.",
          )}
        />
      </SettingsSection>
    </SettingsSections>
  );
}
