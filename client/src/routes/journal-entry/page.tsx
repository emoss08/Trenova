import { AccountingStatusBadge } from "@/components/accounting/accounting-status-badge";
import { AmountDisplay } from "@/components/accounting/amount-display";
import { JournalLineItemsTable } from "@/components/accounting/journal-line-items-table";
import { SourceDrillDownLink } from "@/components/accounting/source-drill-down-link";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeftIcon } from "lucide-react";
import { Link, useNavigate, useParams } from "react-router";

export function JournalEntryDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  const { data: entry, isLoading } = useQuery({
    ...queries.journalEntry.get(id!),
    enabled: !!id,
  });

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
          Back
        </Button>
      </PageLayout>
    );
  }

  const accountingDate = new Date(entry.accountingDate * 1000).toLocaleDateString("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
  });

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
          Back
        </Button>
      </div>

      <div className="space-y-4">
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <CardTitle className="font-mono">{entry.entryNumber}</CardTitle>
                <Badge variant="outline">{entry.entryType}</Badge>
                <AccountingStatusBadge status={entry.status as any} />
              </div>
              <span className="text-sm text-muted-foreground">{accountingDate}</span>
            </div>
            {entry.description && (
              <CardDescription>{entry.description}</CardDescription>
            )}
          </CardHeader>
          <CardContent>
            <dl className="grid grid-cols-2 gap-x-6 gap-y-4 text-sm">
              <div>
                <dt className="text-muted-foreground">Reference Type</dt>
                <dd className="mt-0.5 font-medium">{entry.referenceType}</dd>
              </div>
              <div>
                <dt className="text-muted-foreground">Reference</dt>
                <dd className="mt-0.5">
                  <SourceDrillDownLink
                    sourceType={entry.referenceType}
                    sourceId={entry.referenceId}
                  />
                </dd>
              </div>
              <div>
                <dt className="text-muted-foreground">Is Reversal</dt>
                <dd className="mt-0.5">
                  <Badge variant={entry.isReversal ? "orange" : "secondary"}>
                    {entry.isReversal ? "Yes" : "No"}
                  </Badge>
                </dd>
              </div>
              {entry.reversalOfId && (
                <div>
                  <dt className="text-muted-foreground">Reversal Of</dt>
                  <dd className="mt-0.5">
                    <Link
                      to={`/accounting/journal-entries/${entry.reversalOfId}`}
                      className="font-mono text-xs text-muted-foreground hover:text-foreground hover:underline"
                    >
                      {entry.reversalOfId}
                    </Link>
                  </dd>
                </div>
              )}
              {entry.reversedById && (
                <div>
                  <dt className="text-muted-foreground">Reversed By</dt>
                  <dd className="mt-0.5">
                    <Link
                      to={`/accounting/journal-entries/${entry.reversedById}`}
                      className="font-mono text-xs text-muted-foreground hover:text-foreground hover:underline"
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
              <CardTitle>Line Items</CardTitle>
              <div className="flex items-center gap-4 text-sm">
                <span className="text-muted-foreground">
                  Total Debit:{" "}
                  <AmountDisplay
                    value={entry.totalDebit}
                    className="font-semibold"
                  />
                </span>
                <span className="text-muted-foreground">
                  Total Credit:{" "}
                  <AmountDisplay
                    value={entry.totalCredit}
                    className="font-semibold"
                  />
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
              <p className="text-sm text-muted-foreground">
                No line items available.
              </p>
            )}
          </CardContent>
        </Card>
      </div>
    </PageLayout>
  );
}
