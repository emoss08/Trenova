import { SettingsPageSkeleton } from "@/components/settings/settings-page";
import type { SettingsSectionSpec } from "@/components/settings-form-skeleton";
import { PlanFeature } from "@/lib/plan-feature";
import { usePlanFeature } from "@trenova/shared/hooks/use-plan-feature";
import {
  useCarrierSettlementControlTabs,
  type CarrierSettlementControlTab,
} from "./carrier-settlement-control-tabs";

const choices = (count: number) => ({ kind: "input", count, narrow: false }) as const;
const figures = (count: number) => ({ kind: "input", count, narrow: true }) as const;

/**
 * Each tab's sections as the form draws them, one row to a setting. Matching invoices as
 * they arrive is drawn only on a plan with EDI, and with it off, as a new organization's
 * settings have it, so the row under it is closed.
 */
function tabSections({
  edi,
}: {
  edi: boolean;
}): Record<CarrierSettlementControlTab, readonly SettingsSectionSpec[]> {
  return {
    payPeriod: [
      { title: "sm", description: 1, fields: [choices(2), figures(1), "switch"] },
      { title: "sm", description: 1, fields: [choices(1)] },
    ],
    matching: [
      { title: "sm", description: 1, fields: edi ? [figures(1), "switch"] : [figures(1)] },
      { title: "md", description: 1, fields: ["switch"] },
    ],
    approval: [
      { title: "sm", description: 1, fields: [{ kind: "switch", count: 2 }] },
      { title: "md", description: 1, fields: [choices(2)] },
    ],
  };
}

/** The form's tab row and the open tab's sections. */
export function CarrierSettlementControlSkeleton() {
  const edi = usePlanFeature(PlanFeature.EDI).available;

  return (
    <SettingsPageSkeleton
      tabs={useCarrierSettlementControlTabs()}
      sections={tabSections({ edi })}
    />
  );
}
