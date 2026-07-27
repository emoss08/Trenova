import { queries } from "@/lib/queries";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { formatDetentionMinutes } from "@trenova/shared/lib/detention";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { useMemo, useState } from "react";

const WINDOW_OPTIONS = [
  { label: "30 days", days: 30 },
  { label: "90 days", days: 90 },
  { label: "180 days", days: 180 },
];

function Money({ value, invertColor }: { value: number; invertColor?: boolean }) {
  const negative = value < 0;
  const bad = invertColor ? !negative && value > 0 : negative;

  return (
    <span
      className={cn(
        "tabular-nums",
        bad && "text-red-600 dark:text-red-400",
        !bad && value !== 0 && "text-foreground",
      )}
    >
      {formatCurrency(value)}
    </span>
  );
}

function Section({
  title,
  description,
  children,
}: {
  title: string;
  description: string;
  children: React.ReactNode;
}) {
  return (
    <section className="bg-card overflow-hidden rounded-lg border">
      <div className="border-b px-4 py-3">
        <h3 className="text-sm font-medium">{title}</h3>
        <p className="text-muted-foreground mt-0.5 text-xs">{description}</p>
      </div>
      <div className="overflow-x-auto">{children}</div>
    </section>
  );
}

function EmptyRows({ message }: { message: string }) {
  return <p className="text-muted-foreground px-4 py-6 text-sm">{message}</p>;
}

function FacilityProfiles({ from, to }: { from: number; to: number }) {
  const { data, isLoading } = useQuery(queries.detention.facilities(from, to));

  if (isLoading) {
    return <Skeleton className="m-4 h-48 w-auto" />;
  }

  const rows = data ?? [];

  if (rows.length === 0) {
    return <EmptyRows message="No settled detention in this window." />;
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Facility</TableHead>
          <TableHead className="text-right">Stops</TableHead>
          <TableHead className="text-right">Breach rate</TableHead>
          <TableHead className="text-right">Median dwell</TableHead>
          <TableHead className="text-right">p90 dwell</TableHead>
          <TableHead className="text-right">Billed</TableHead>
          <TableHead className="text-right">Margin</TableHead>
          <TableHead className="text-right">Lost</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {rows.map((row) => {
          const breachRate =
            row.stopCount > 0 ? Math.round((row.breachCount / row.stopCount) * 100) : 0;

          return (
            <TableRow key={row.locationId}>
              <TableCell className="max-w-[220px]">
                <span className="block truncate font-medium">
                  {row.locationName || row.locationId}
                </span>
              </TableCell>
              <TableCell className="text-right tabular-nums">{row.stopCount}</TableCell>
              <TableCell className="text-right tabular-nums">
                <span className={cn(breachRate >= 50 && "text-red-600 dark:text-red-400")}>
                  {breachRate}%
                </span>
              </TableCell>
              <TableCell className="text-right tabular-nums">
                {formatDetentionMinutes(Math.round(row.medianDwellMinutes))}
              </TableCell>
              <TableCell className="text-right tabular-nums">
                {formatDetentionMinutes(Math.round(row.p90DwellMinutes))}
              </TableCell>
              <TableCell className="text-right">
                <Money value={row.billedAmount} />
              </TableCell>
              <TableCell className="text-right">
                <Money value={row.netMargin} />
              </TableCell>
              <TableCell className="text-right">
                <div className="flex items-center justify-end gap-1">
                  <Money value={row.waivedAmount} invertColor />
                  {row.suppressedCount > 0 && (
                    <Badge variant="outline" className="text-[10px]">
                      {row.suppressedCount} no notice
                    </Badge>
                  )}
                </div>
              </TableCell>
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
}

function CustomerMargin({ from, to }: { from: number; to: number }) {
  const { data, isLoading } = useQuery(queries.detention.customers(from, to));

  if (isLoading) {
    return <Skeleton className="m-4 h-48 w-auto" />;
  }

  const rows = data ?? [];

  if (rows.length === 0) {
    return <EmptyRows message="No settled detention in this window." />;
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Customer</TableHead>
          <TableHead className="text-right">Stops</TableHead>
          <TableHead className="text-right">Billed</TableHead>
          <TableHead className="text-right">Driver pay</TableHead>
          <TableHead className="text-right">Net margin</TableHead>
          <TableHead className="text-right">Disputed</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {rows.map((row) => (
          <TableRow key={row.customerId}>
            <TableCell className="max-w-[220px]">
              <span className="block truncate font-medium">
                {row.customerName || row.customerId}
              </span>
            </TableCell>
            <TableCell className="text-right tabular-nums">{row.stopCount}</TableCell>
            <TableCell className="text-right">
              <Money value={row.billedAmount} />
            </TableCell>
            <TableCell className="text-right">
              <Money value={row.driverPayAmount} />
            </TableCell>
            <TableCell className="text-right">
              <Money value={row.netMargin} />
            </TableCell>
            <TableCell className="text-right tabular-nums">
              {row.disputeCount > 0 ? (
                <Badge variant="outline" className="text-[10px]">
                  {row.disputeCount}
                </Badge>
              ) : (
                "—"
              )}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

function WaiverLeakage({ from, to }: { from: number; to: number }) {
  const { data, isLoading } = useQuery(queries.detention.waivers(from, to));

  if (isLoading) {
    return <Skeleton className="m-4 h-32 w-auto" />;
  }

  const rows = data ?? [];

  if (rows.length === 0) {
    return <EmptyRows message="No detention was waived in this window." />;
  }

  const total = rows.reduce((sum, row) => sum + row.waivedAmount, 0);

  return (
    <div>
      <p className="px-4 pt-3 text-sm">
        <span className="font-semibold tabular-nums">{formatCurrency(total)}</span> of
        detention was forgiven across {rows.length}{" "}
        {rows.length === 1 ? "reason" : "reasons"}.
      </p>

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Reason</TableHead>
            <TableHead className="text-right">Waivers</TableHead>
            <TableHead className="text-right">Approvers</TableHead>
            <TableHead className="text-right">Forgiven</TableHead>
            <TableHead className="text-right">Share</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((row) => (
            <TableRow key={row.reason}>
              <TableCell className="font-medium">{row.reason}</TableCell>
              <TableCell className="text-right tabular-nums">{row.waiverCount}</TableCell>
              <TableCell className="text-right tabular-nums">
                {row.approverCount}
              </TableCell>
              <TableCell className="text-right">
                <Money value={row.waivedAmount} invertColor />
              </TableCell>
              <TableCell className="text-right tabular-nums">
                {total > 0 ? Math.round((row.waivedAmount / total) * 100) : 0}%
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

/**
 * Detention intelligence: which facilities cost the most, which customers are
 * unprofitable once driver pay is netted off, and where discretionary revenue
 * is going.
 */
export function DetentionIntelligence() {
  const [days, setDays] = useState(90);

  const { from, to } = useMemo(() => {
    const now = Math.floor(Date.now() / 1000);
    return { from: now - days * 86400, to: now };
  }, [days]);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between gap-2">
        <p className="text-muted-foreground text-xs">
          Settled detention over the selected window. The p90 dwell is the planning number —
          an average hides the afternoons that blow through free time.
        </p>
        <div className="flex shrink-0 items-center gap-1.5">
          {WINDOW_OPTIONS.map((option) => (
            <Button
              key={option.days}
              type="button"
              size="sm"
              variant={days === option.days ? "default" : "outline"}
              className="h-7 text-xs"
              onClick={() => setDays(option.days)}
            >
              {option.label}
            </Button>
          ))}
        </div>
      </div>

      <Section
        title="Facility profiles"
        description="Where detention actually accrues — the docks worth renegotiating or replanning around."
      >
        <FacilityProfiles from={from} to={to} />
      </Section>

      <Section
        title="Customer margin"
        description="Detention billed against detention paid. A negative margin means the customer's free-time concession exceeds what the driver contract grants."
      >
        <CustomerMargin from={from} to={to} />
      </Section>

      <Section
        title="Waiver leakage"
        description="Revenue forgiven at someone's discretion, grouped by the coded reason given."
      >
        <WaiverLeakage from={from} to={to} />
      </Section>
    </div>
  );
}
