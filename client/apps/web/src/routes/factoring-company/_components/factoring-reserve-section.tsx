import { FactoringReserveEntryList } from "@/components/factoring/factoring-reserve-entry-list";
import { SectionPanel } from "@/components/section-panel";
import { usePermission } from "@/hooks/use-permission";
import { factoring as factoringQueries } from "@/lib/queries/factoring";
import { useQuery } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatRowTotal } from "@trenova/shared/lib/row-total";
import { formatMinor } from "@trenova/shared/lib/benefits";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { Operation, Resource } from "@trenova/shared/types/permission";

/**
 * What this factor holds and has paid, from the movements booked to the
 * ledger: the reserve it still holds, what is owed it under secured
 * borrowing, what it advanced and released, what was paid back for
 * chargebacks, the fees, and the latest movements themselves.
 */
export function FactoringReserveSection({ factoringCompanyId }: { factoringCompanyId: string }) {
  const t = useT();
  const { allowed: canRead } = usePermission(Resource.FactoringReserve, Operation.Read);
  const balance = useQuery({
    ...factoringQueries.reserveBalance(factoringCompanyId),
    enabled: canRead,
  });
  const entries = useQuery({
    ...factoringQueries.reserveEntries({ factoringCompanyId }),
    enabled: canRead,
  });

  if (!canRead) {
    return null;
  }

  const currencyCode = entries.data?.entries[0]?.currencyCode ?? "USD";
  const money = (minor: number) => formatMinor(minor, currencyCode);

  return (
    <SectionPanel
      title={t("Reserve")}
      count={balance.data?.entryCount}
      help={t(
        "Every movement between your organization and this factor that reached the ledger: advances, reserve releases, chargebacks and their reversals.",
      )}
    >
      {balance.isLoading || entries.isLoading ? (
        <div className="space-y-2 p-3">
          <Skeleton className="h-4 w-48" />
          <Skeleton className="h-4 w-64" />
          <Skeleton className="h-16 w-full" />
        </div>
      ) : balance.isError || entries.isError || !balance.data || !entries.data ? (
        <div className="p-3">
          <Alert size="sm" variant="destructive">
            <AlertDescription className="flex items-center justify-between gap-3">
              <span>{t("The reserve could not be loaded.")}</span>
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() => {
                  void balance.refetch();
                  void entries.refetch();
                }}
              >
                {t("Retry")}
              </Button>
            </AlertDescription>
          </Alert>
        </div>
      ) : balance.data.entryCount === 0 ? (
        <p className="text-muted-foreground px-3 py-3 text-sm">
          {t("Nothing booked with this factor yet. Funding recorded on a schedule appears here.")}
        </p>
      ) : (
        <div className="flex flex-col">
          <div className="p-3">
            <DescriptionList columns={2}>
              <DescriptionItem label={t("Reserve held")} numeric>
                {money(balance.data.reserveMinor)}
              </DescriptionItem>
              <DescriptionItem label={t("Owed to factor")} numeric>
                {money(balance.data.liabilityMinor)}
              </DescriptionItem>
              <DescriptionItem label={t("Advanced")} numeric>
                {money(balance.data.advancedMinor)}
              </DescriptionItem>
              <DescriptionItem label={t("Reserve released")} numeric>
                {money(balance.data.releasedMinor)}
              </DescriptionItem>
              <DescriptionItem label={t("Repaid for chargebacks")} numeric>
                {money(balance.data.chargedBackMinor)}
              </DescriptionItem>
              <DescriptionItem label={t("Fees")} numeric>
                {money(balance.data.feeMinor)}
              </DescriptionItem>
              <DescriptionItem label={t("Last movement")}>
                {balance.data.lastOccurredOn ? (
                  formatUnixDateMedium(balance.data.lastOccurredOn)
                ) : (
                  <DescriptionEmpty />
                )}
              </DescriptionItem>
            </DescriptionList>
          </div>
          <FactoringReserveEntryList entries={entries.data.entries} show="invoice" />
          {entries.data.totalCount > entries.data.entries.length ? (
            <p className="text-muted-foreground border-t px-3 py-2 text-xs">
              {entries.data.totalCountExact === false
                ? t(
                    "Showing the latest {0} of about {1} movements. A report on factoring reserve entries lists them all.",
                    entries.data.entries.length,
                    formatRowTotal(entries.data.totalCount, false),
                  )
                : t(
                    "Showing the latest {0} of {1} movements. A report on factoring reserve entries lists them all.",
                    entries.data.entries.length,
                    entries.data.totalCount,
                  )}
            </p>
          ) : null}
        </div>
      )}
    </SectionPanel>
  );
}
