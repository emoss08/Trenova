import type { IftaReturnLine } from "@/lib/graphql/ifta-return";
import {
  formatIftaMeasure,
  groupLinesByFuelType,
  IFTA_GALLONS_SCALE,
  IFTA_MILES_DISPLAY_SCALE,
  IFTA_MONEY_SCALE,
  isNegativeDecimal,
  sumLines,
  type IftaLineTotals,
  type IftaReturnView,
} from "@/lib/ifta-return";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@trenova/shared/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableFooter,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { cn } from "@trenova/shared/lib/utils";
import { IFTA_FUEL_TYPE_LABELS } from "@trenova/shared/types/fuel-ifta-enums";
import { Fragment } from "react";
import { Link } from "react-router";

export const ADD_RATE_PATH = "/fuel/configuration-files/ifta-tax-rates?panelType=create";

const RATE_SCALE = 4;
const COLUMN_COUNT = 10;

function moneyClass(value: string): string {
  return isNegativeDecimal(value) ? "text-emerald-600 dark:text-emerald-400" : "";
}

function milesBreakdown(line: IftaReturnLine): string {
  return [
    `${formatIftaMeasure(line.routeMiles, IFTA_MILES_DISPLAY_SCALE)} routed`,
    `${formatIftaMeasure(line.manualMiles, IFTA_MILES_DISPLAY_SCALE)} entered by hand`,
    `${formatIftaMeasure(line.loadedMiles, IFTA_MILES_DISPLAY_SCALE)} loaded`,
    `${formatIftaMeasure(line.emptyMiles, IFTA_MILES_DISPLAY_SCALE)} empty`,
  ].join(" · ");
}

function TotalsCells({ totals }: { totals: IftaLineTotals }) {
  return (
    <>
      <TableCell className="text-right tabular-nums">
        {formatIftaMeasure(totals.totalMiles, IFTA_MILES_DISPLAY_SCALE)}
      </TableCell>
      <TableCell className="text-right tabular-nums">
        {formatIftaMeasure(totals.taxableMiles, IFTA_MILES_DISPLAY_SCALE)}
      </TableCell>
      <TableCell className="text-right tabular-nums">
        {formatIftaMeasure(totals.taxPaidGallons, IFTA_GALLONS_SCALE)}
      </TableCell>
      <TableCell className="text-right tabular-nums">
        {formatIftaMeasure(totals.taxableGallons, IFTA_GALLONS_SCALE)}
      </TableCell>
      <TableCell className={cn("text-right tabular-nums", moneyClass(totals.netTaxableGallons))}>
        {formatIftaMeasure(totals.netTaxableGallons, IFTA_GALLONS_SCALE)}
      </TableCell>
      <TableCell />
      <TableCell className={cn("text-right tabular-nums", moneyClass(totals.taxDue))}>
        {formatIftaMeasure(totals.taxDue, IFTA_MONEY_SCALE)}
      </TableCell>
      <TableCell className="text-right tabular-nums">
        {formatIftaMeasure(totals.surchargeDue, IFTA_MONEY_SCALE)}
      </TableCell>
      <TableCell
        className={cn("text-right font-semibold tabular-nums", moneyClass(totals.lineTotal))}
      >
        {formatIftaMeasure(totals.lineTotal, IFTA_MONEY_SCALE)}
      </TableCell>
    </>
  );
}

function LineRow({ line }: { line: IftaReturnLine }) {
  const missingRate = line.rateMissing && line.isIftaMember;

  return (
    <TableRow
      className={cn(
        missingRate && "bg-red-50/60 dark:bg-red-950/20",
        !line.isIftaMember && "text-muted-foreground",
      )}
    >
      <TableCell>
        <div className="flex items-center gap-1.5">
          <span className="font-medium">{line.jurisdiction.code}</span>
          <span className="text-muted-foreground text-xs">{line.jurisdiction.name}</span>
          {missingRate ? <Badge variant="inactive">No rate</Badge> : null}
          {line.isIftaMember ? null : <Badge variant="outline">Non-member</Badge>}
          {line.jurisdiction.hasSurcharge ? <Badge variant="outline">Surcharge</Badge> : null}
        </div>
      </TableCell>
      <TableCell className="text-right tabular-nums" title={milesBreakdown(line)}>
        {formatIftaMeasure(line.totalMiles, IFTA_MILES_DISPLAY_SCALE)}
      </TableCell>
      <TableCell className="text-right tabular-nums">
        {formatIftaMeasure(line.taxableMiles, IFTA_MILES_DISPLAY_SCALE)}
      </TableCell>
      <TableCell
        className="text-right tabular-nums"
        title={`${line.purchaseCount} purchases · ${line.taxPaidGallonsRaw} gallons before rounding`}
      >
        {formatIftaMeasure(line.taxPaidGallons, IFTA_GALLONS_SCALE)}
      </TableCell>
      <TableCell className="text-right tabular-nums">
        {formatIftaMeasure(line.taxableGallons, IFTA_GALLONS_SCALE)}
      </TableCell>
      <TableCell className={cn("text-right tabular-nums", moneyClass(line.netTaxableGallons))}>
        {formatIftaMeasure(line.netTaxableGallons, IFTA_GALLONS_SCALE)}
      </TableCell>
      <TableCell className="text-right tabular-nums">
        {missingRate ? (
          <Link to={ADD_RATE_PATH} className="text-brand text-xs font-medium hover:underline">
            Add rate
          </Link>
        ) : line.ratePerGallon ? (
          formatIftaMeasure(line.ratePerGallon, RATE_SCALE)
        ) : (
          "—"
        )}
      </TableCell>
      <TableCell className={cn("text-right tabular-nums", moneyClass(line.taxDue))}>
        {formatIftaMeasure(line.taxDue, IFTA_MONEY_SCALE)}
      </TableCell>
      <TableCell className="text-right tabular-nums">
        {formatIftaMeasure(line.surchargeDue, IFTA_MONEY_SCALE)}
      </TableCell>
      <TableCell className={cn("text-right font-medium tabular-nums", moneyClass(line.lineTotal))}>
        {formatIftaMeasure(line.lineTotal, IFTA_MONEY_SCALE)}
      </TableCell>
    </TableRow>
  );
}

export function ReturnLinesTable({ ret }: { ret: IftaReturnView }) {
  const groups = groupLinesByFuelType(ret.lines);
  const grandTotal = sumLines(ret.lines);

  return (
    <Card className="rounded-md">
      <CardHeader>
        <CardTitle className="text-sm font-semibold">Jurisdiction lines</CardTitle>
        <p className="text-muted-foreground text-xs">
          One line per jurisdiction and fuel type. Gallons and miles are whole as the form prints
          them; money is in {ret.currencyCode}, and a negative figure is a credit.
        </p>
      </CardHeader>
      <CardContent>
        <div className="overflow-x-auto rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Jurisdiction</TableHead>
                <TableHead className="text-right">Total miles</TableHead>
                <TableHead className="text-right">Taxable miles</TableHead>
                <TableHead className="text-right">Tax-paid gal</TableHead>
                <TableHead className="text-right">Taxable gal</TableHead>
                <TableHead className="text-right">Net taxable gal</TableHead>
                <TableHead className="text-right">Rate</TableHead>
                <TableHead className="text-right">Tax due</TableHead>
                <TableHead className="text-right">Surcharge</TableHead>
                <TableHead className="text-right">Line total</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {groups.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={COLUMN_COUNT}
                    className="text-muted-foreground py-6 text-center text-xs"
                  >
                    No jurisdiction carried miles or fuel in this quarter. Attribute the
                    quarter&apos;s moves, or record the miles by hand, then recompute.
                  </TableCell>
                </TableRow>
              ) : null}
              {groups.map((group) => (
                <Fragment key={group.fuelType}>
                  <TableRow className="bg-muted/40">
                    <TableCell colSpan={COLUMN_COUNT} className="text-xs font-semibold">
                      {IFTA_FUEL_TYPE_LABELS[group.fuelType]}
                    </TableCell>
                  </TableRow>
                  {group.lines.map((line) => (
                    <LineRow key={line.id} line={line} />
                  ))}
                  <TableRow className="border-t-2">
                    <TableCell className="text-xs font-semibold">
                      {IFTA_FUEL_TYPE_LABELS[group.fuelType]} subtotal
                    </TableCell>
                    <TotalsCells totals={group.subtotal} />
                  </TableRow>
                </Fragment>
              ))}
            </TableBody>
            <TableFooter>
              <TableRow>
                <TableCell className="text-xs font-semibold">Total</TableCell>
                <TotalsCells totals={grandTotal} />
              </TableRow>
            </TableFooter>
          </Table>
        </div>
      </CardContent>
    </Card>
  );
}
