import { useT } from "@trenova/shared/i18n/use-t";
import { AccountingStatusBadge } from "@/components/accounting/accounting-status-badge";
import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { JournalLineItemsTable } from "@/components/accounting/journal-line-items-table";
import { SourceDrillDownLink } from "@/components/accounting/source-drill-down-link";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@trenova/shared/components/ui/card";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { queries } from "@/lib/queries";
import { useBreadcrumbLabel } from "@/hooks/use-breadcrumb-label";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeftIcon } from "lucide-react";
import { Link, useNavigate, useParams } from "react-router";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";

export function JournalEntryDetailPage() {
  const t = useT();

  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  const { data: entry, isLoading } = useQuery({
    ...queries.journalEntry.get(id!),
    enabled: !!id,
  });
  useBreadcrumbLabel(entry?.entryNumber);

  if (isLoading) {
    return (
      <PageLayout
        pageHeaderProps={{
          title: "Journal Entry",
          description: "Loading entry details...",
        }}
      >
        <div className="space-y-4">
          <Skeleton className="h-48 w-full" />
          <Skeleton className="h-64 w-full" />
        </div>
      </PageLayout>
    );
  }

  if (!entry) {
    return (
      <PageLayout
        pageHeaderProps={{
          title: "Journal Entry",
          description: "Entry not found.",
        }}
      >
        <Button variant="outline" onClick={() => void navigate(-1)}>
          <ArrowLeftIcon className="mr-1.5 size-3.5" />
          {t("Back")}
        </Button>
      </PageLayout>
    );
  }

  const accountingDate = formatUnixDateMedium(entry.accountingDate);

  return (
    <PageLayout
      pageHeaderProps={{
        title: `Journal Entry ${entry.entryNumber}`,
        description: entry.description || `${entry.entryType} journal entry`,
      }}
    >
      <div className="mb-2">
        <Button variant="outline" size="sm" onClick={() => void navigate(-1)}>
          <ArrowLeftIcon className="mr-1.5 size-3.5" />
          {t("Back")}
        </Button>
      </div>

      <div className="space-y-4">
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <CardTitle className="font-mono">{entry.entryNumber}</CardTitle>
                <Badge variant="outline">{entry.entryType}</Badge>
                <AccountingStatusBadge status={entry.status} />
              </div>
              <span className="text-muted-foreground text-sm">{accountingDate}</span>
            </div>
            {entry.description && <CardDescription>{entry.description}</CardDescription>}
          </CardHeader>
          <CardContent>
            <dl className="grid grid-cols-2 gap-x-6 gap-y-4 text-sm">
              <div>
                <dt className="text-muted-foreground">{t("Reference Type")}</dt>
                <dd className="mt-0.5 font-medium">{entry.referenceType}</dd>
              </div>
              <div>
                <dt className="text-muted-foreground">{t("Reference")}</dt>
                <dd className="mt-0.5">
                  <SourceDrillDownLink
                    sourceType={entry.referenceType}
                    sourceId={entry.referenceId}
                  />
                </dd>
              </div>
              <div>
                <dt className="text-muted-foreground">{t("Is Reversal")}</dt>
                <dd className="mt-0.5">
                  <Badge variant={entry.isReversal ? "orange" : "secondary"}>
                    {entry.isReversal ? "Yes" : "No"}
                  </Badge>
                </dd>
              </div>
              {entry.reversalOfId && (
                <div>
                  <dt className="text-muted-foreground">{t("Reversal Of")}</dt>
                  <dd className="mt-0.5">
                    <Link
                      to={`/accounting/journal-entries/${entry.reversalOfId}`}
                      className="text-muted-foreground hover:text-foreground font-mono text-xs hover:underline"
                    >
                      {entry.reversalOfId}
                    </Link>
                  </dd>
                </div>
              )}
              {entry.reversedById && (
                <div>
                  <dt className="text-muted-foreground">{t("Reversed By")}</dt>
                  <dd className="mt-0.5">
                    <Link
                      to={`/accounting/journal-entries/${entry.reversedById}`}
                      className="text-muted-foreground hover:text-foreground font-mono text-xs hover:underline"
                    >
                      {entry.reversedById}
                    </Link>
                  </dd>
                </div>
              )}
            </dl>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle>{t("Line Items")}</CardTitle>
              <div className="flex items-center gap-4 text-sm">
                <span className="text-muted-foreground">
                  {t("Total Debit:")} <AmountDisplay value={entry.totalDebit} className="font-semibold" />
                </span>
                <span className="text-muted-foreground">
                  {t("Total Credit:")}{" "}
                  <AmountDisplay value={entry.totalCredit} className="font-semibold" />
                </span>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            {entry.lines && entry.lines.length > 0 ? (
              <JournalLineItemsTable
                lines={entry.lines}
                totalDebit={entry.totalDebit}
                totalCredit={entry.totalCredit}
              />
            ) : (
              <p className="text-muted-foreground text-sm">{t("No line items available.")}</p>
            )}
          </CardContent>
        </Card>
      </div>
    </PageLayout>
  );
}
