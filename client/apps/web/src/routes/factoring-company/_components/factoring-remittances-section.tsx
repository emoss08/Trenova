import { FactoringRemittanceStatusBadge } from "@/components/factoring/factoring-badges";
import { SectionPanel } from "@/components/section-panel";
import { recordPath } from "@/config/record-links";
import { usePermission } from "@/hooks/use-permission";
import { factoring as factoringQueries } from "@/lib/queries/factoring";
import { ImportRemittanceDialog } from "@/routes/factoring/_components/import-remittance-dialog";
import { useQuery } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { formatCurrency } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useState } from "react";
import { Link } from "react-router";

/** The most recent remittances the section lists; the factoring page holds the rest. */
const RECENT_REMITTANCES = 5;

/**
 * The funding and reserve reports this factor sent, newest first, each opening
 * its review, and the import that reads the next one.
 */
export function FactoringRemittancesSection({
  factoringCompanyId,
}: {
  factoringCompanyId: string;
}) {
  const t = useT();
  const { allowed: canRead } = usePermission(Resource.FactoringRemittance, Operation.Read);
  const { allowed: canCreate } = usePermission(Resource.FactoringRemittance, Operation.Create);
  const [importing, setImporting] = useState(false);
  const remittances = useQuery({
    ...factoringQueries.remittances({ factoringCompanyId }),
    enabled: canRead,
  });

  if (!canRead) {
    return null;
  }

  return (
    <SectionPanel
      title={t("Remittances")}
      count={remittances.data?.totalCount}
      help={t(
        "Funding and reserve reports from this factor. Each is matched to its schedules and booked only when someone applies it.",
      )}
      action={
        canCreate ? (
          <Button type="button" size="sm" variant="outline" onClick={() => setImporting(true)}>
            {t("Import remittance")}
          </Button>
        ) : undefined
      }
    >
      {remittances.isLoading ? (
        <div className="space-y-2 p-3">
          <Skeleton className="h-4 w-64" />
          <Skeleton className="h-4 w-56" />
          <Skeleton className="h-4 w-48" />
        </div>
      ) : remittances.isError || !remittances.data ? (
        <div className="p-3">
          <Alert size="sm" variant="destructive">
            <AlertDescription className="flex items-center justify-between gap-3">
              <span>{t("The remittances could not be loaded.")}</span>
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() => void remittances.refetch()}
              >
                {t("Retry")}
              </Button>
            </AlertDescription>
          </Alert>
        </div>
      ) : remittances.data.remittances.length === 0 ? (
        <p className="text-muted-foreground px-3 py-3 text-sm">
          {t(
            "No remittance from this factor yet. Import one, or forward the factor's emails to your inbox.",
          )}
        </p>
      ) : (
        <ul className="divide-border divide-y">
          {remittances.data.remittances.slice(0, RECENT_REMITTANCES).map((remittance) => (
            <li key={remittance.id} className="flex items-center justify-between gap-3 px-3 py-2">
              <span className="flex min-w-0 flex-col">
                <Link
                  to={recordPath("factoring_remittance", remittance.id)}
                  className="truncate text-sm font-medium hover:underline"
                >
                  {remittance.fileName || t("Factor remittance")}
                </Link>
                <span className="text-muted-foreground text-xs">
                  {t(
                    "{0} · {1, plural, one {# line} other {# lines}}",
                    formatUnixDateMedium(remittance.remittedOn ?? remittance.createdAt),
                    remittance.lineCount,
                  )}
                </span>
              </span>
              <span className="flex shrink-0 items-center gap-2">
                <span className="text-sm tabular-nums">
                  {formatCurrency(remittance.totalCashMinor / 100, remittance.currencyCode)}
                </span>
                <FactoringRemittanceStatusBadge status={remittance.status} />
              </span>
            </li>
          ))}
        </ul>
      )}
      <ImportRemittanceDialog
        open={importing}
        onOpenChange={setImporting}
        factoringCompanyId={factoringCompanyId}
      />
    </SectionPanel>
  );
}
