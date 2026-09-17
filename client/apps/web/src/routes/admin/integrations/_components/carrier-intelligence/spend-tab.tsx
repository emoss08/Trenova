import { NumberField } from "@/components/fields/number-field";
import { CarrierIntelUsageSummaryView } from "@/components/carrier-intelligence/usage-summary";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext } from "react-hook-form";
import type { CarrierIntelSettingsFormValues } from "./carrier-intel-settings-schema";
import { SettingsSection } from "./settings-section";

export function CarrierIntelSpendTab({
  open,
  readOnly,
  canManage,
}: {
  open: boolean;
  readOnly: boolean;
  canManage: boolean;
}) {
  const t = useT();
  const { control } = useFormContext<CarrierIntelSettingsFormValues>();

  return (
    <div className="space-y-6">
      <fieldset disabled={readOnly} className="m-0 min-w-0 space-y-6 border-0 p-0">
        <SettingsSection
          title={t("Spend limits")}
          description={t(
            "Monitoring pauses when the monthly cap is reached. Lookups that would exceed the daily full profile cap fall back to a lighter lookup.",
          )}
        >
          <FormGroup cols={3}>
            <FormControl>
              <NumberField
                control={control}
                name="monthlySpendCap"
                valueType="string"
                label={t("Monthly spend cap")}
                description={t("Leave blank for no cap.")}
                placeholder={t("No cap")}
                prefix="$"
                decimalScale={2}
                thousandSeparator
                readOnly={readOnly}
              />
            </FormControl>
            <FormControl>
              <NumberField
                control={control}
                name="softCapPercent"
                label={t("Soft cap warning")}
                description={t("Warn administrators at this share of the cap.")}
                sideText="%"
                min={1}
                max={100}
                readOnly={readOnly}
              />
            </FormControl>
            <FormControl>
              <NumberField
                control={control}
                name="dailyFullProfileCap"
                label={t("Daily full profile cap")}
                description={t("Leave blank for no daily limit.")}
                placeholder={t("No limit")}
                min={0}
                readOnly={readOnly}
              />
            </FormControl>
          </FormGroup>
        </SettingsSection>
        <SettingsSection
          title={t("Retention")}
          description={t("How long raw provider responses and snapshot history are kept.")}
        >
          <FormGroup cols={3}>
            <FormControl>
              <NumberField
                control={control}
                name="rawRetentionDays"
                label={t("Raw payload retention")}
                sideText={t("days")}
                min={7}
                max={730}
                readOnly={readOnly}
              />
            </FormControl>
            <FormControl>
              <NumberField
                control={control}
                name="snapshotHistoryLimit"
                label={t("Snapshot history")}
                description={t("Snapshots kept per carrier.")}
                min={1}
                max={100}
                readOnly={readOnly}
              />
            </FormControl>
          </FormGroup>
        </SettingsSection>
      </fieldset>
      {canManage ? (
        <UsageMeter open={open} />
      ) : (
        <div className="border-border bg-muted/20 text-muted-foreground rounded-md border p-3 text-sm">
          {t("Usage and spend are visible to users who can manage carrier intelligence.")}
        </div>
      )}
    </div>
  );
}

function UsageMeter({ open }: { open: boolean }) {
  const t = useT();
  const usageQuery = useQuery({
    ...queries.carrierIntelSettings.usage(),
    enabled: open,
  });

  return (
    <SettingsSection
      title={t("Month to date")}
      description={t("Estimated provider charges for the current month, in UTC.")}
    >
      {usageQuery.isLoading ? (
        <Skeleton className="h-40 w-full" />
      ) : usageQuery.isError || !usageQuery.data ? (
        <div className="border-border text-destructive rounded-md border p-3 text-sm">
          {t("Usage could not be loaded.")}
        </div>
      ) : (
        <CarrierIntelUsageSummaryView usage={usageQuery.data} />
      )}
    </SettingsSection>
  );
}
