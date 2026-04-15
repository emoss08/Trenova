import { AmountDisplay } from "@/components/accounting/amount-display";
import { SourceDrillDownLink } from "@/components/accounting/source-drill-down-link";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeftIcon } from "lucide-react";
import { Link, useNavigate, useParams } from "react-router";

export function SourceDrillDownPage() {
  const { type, id } = useParams<{ type: string; id: string }>();
  const navigate = useNavigate();

  const { data: entries, isLoading } = useQuery({
    ...queries.journalEntry.bySource(type!, id!),
    enabled: !!type && !!id,
  });

  if (isLoading) {
    return (
      <PageLayout
        pageHeaderProps={{
          title: "Journal Entries",
          description: "Loading source journal entries...",
        }}
      >
        <Skeleton className="h-64 w-full" />
      </PageLayout>
    );
  }

  return (
    <PageLayout
      pageHeaderProps={{
        title: `Journal Entries for ${type}`,
        description: `Viewing all journal entries linked to this ${type} source.`,
      }}
    >
      <div className="mb-2 flex items-center gap-3">
        <Button variant="outline" size="sm" onClick={() => void navigate(-1)}>
          <ArrowLeftIcon className="mr-1.5 size-3.5" />
          Back
        </Button>
        <SourceDrillDownLink sourceType={type!} sourceId={id!} label="View Source" />
      </div>

      <Card>
        <CardHeader>
          <CardTitle>
            {entries?.length ?? 0} Journal{" "}
            {entries?.length === 1 ? "Entry" : "Entries"}
          </CardTitle>
        </CardHeader>
        <CardContent>
          {!entries || entries.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              No journal entries found for this source.
            </p>
          ) : (
            <div className="overflow-hidden rounded-md border">
              <table className="w-full text-sm">
                <thead className="bg-muted/50 text-left text-muted-foreground">
                  <tr>
                    <th className="px-3 py-2 text-xs font-medium">
                      Entry Number
                    </th>
                    <th className="px-3 py-2 text-xs font-medium">
                      Entry Type
                    </th>
                    <th className="px-3 py-2 text-xs font-medium">
                      Accounting Date
                    </th>
                    <th className="px-3 py-2 text-right text-xs font-medium">
                      Total Debit
                    </th>
                    <th className="px-3 py-2 text-right text-xs font-medium">
                      Total Credit
                    </th>
                    <th className="px-3 py-2 text-xs font-medium">Reversal</th>
                  </tr>
                </thead>
                <tbody>
                  {entries.map((entry) => {
                    const dateStr = new Date(
                      entry.accountingDate * 1000,
                    ).toLocaleDateString("en-US", {
                      year: "numeric",
                      month: "short",
                      day: "numeric",
                    });

                    return (
                      <tr
                        key={entry.id}
                        className="border-t transition-colors hover:bg-muted/50"
                      >
                        <td className="px-3 py-2">
                          <Link
                            to={`/accounting/journal-entries/${entry.id}`}
                            className="font-mono text-xs text-muted-foreground hover:text-foreground hover:underline"
                          >
                            {entry.entryNumber}
                          </Link>
                        </td>
                        <td className="px-3 py-2">
                          <Badge variant="outline">{entry.entryType}</Badge>
                        </td>
                        <td className="px-3 py-2 text-xs text-muted-foreground">
                          {dateStr}
                        </td>
                        <td className="px-3 py-2 text-right">
                          <AmountDisplay
                            value={entry.totalDebit}
                            className="text-xs"
                          />
                        </td>
                        <td className="px-3 py-2 text-right">
                          <AmountDisplay
                            value={entry.totalCredit}
                            className="text-xs"
                          />
                        </td>
                        <td className="px-3 py-2">
                          <Badge
                            variant={entry.isReversal ? "orange" : "secondary"}
                          >
                            {entry.isReversal ? "Yes" : "No"}
                          </Badge>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>
    </PageLayout>
  );
}
