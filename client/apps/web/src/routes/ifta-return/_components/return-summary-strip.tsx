import { translate } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { KpiInfoPopover, type KpiInfoRow } from "@/components/kpi/kpi-info-popover";
import { StatTile } from "@/components/stat-tile";
import {
  formatIftaMeasure,
  formatIftaMoney,
  IFTA_GALLONS_SCALE,
  IFTA_MILES_DISPLAY_SCALE,
  IFTA_MONEY_SCALE,
  IFTA_MPG_SCALE,
  netPosition,
  type IftaReturnView,
} from "@/lib/ifta-return";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import { IFTA_FUEL_TYPE_LABELS } from "@trenova/shared/types/fuel-ifta-enums";

const MPG_EXPLANATION =
  "Total miles ÷ total gallons, per fuel type. Every jurisdiction's taxable gallons = its miles ÷ this MPG";

function mpgRows(ret: IftaReturnView): KpiInfoRow[] {
  if (ret.fleetMpgByFuelType.length === 0) {
    return [
      {
        label: translate("No fuel types"),
        value:
          "No tractor ran miles in the quarter, so there is no average to take and no taxable gallons on any line.",
      },
    ];
  }
  return ret.fleetMpgByFuelType.map((entry) => ({
    label: IFTA_FUEL_TYPE_LABELS[entry.fuelType],
    value:
      entry.mpg === null
        ? `${formatIftaMeasure(entry.totalMiles, IFTA_MILES_DISPLAY_SCALE)} miles and no gallons bought, so this fuel type's lines carry no taxable gallons.`
        : `${formatIftaMeasure(entry.totalMiles, IFTA_MILES_DISPLAY_SCALE)} miles ÷ ${formatIftaMeasure(entry.totalGallons, IFTA_GALLONS_SCALE)} gallons = ${formatIftaMeasure(entry.mpg, IFTA_MPG_SCALE)} mpg`,
  }));
}

function statusFacts(ret: IftaReturnView): string {
  const facts: string[] = [];
  facts.push(
    ret.computedAt
      ? `Computed ${formatUnixDateTime(ret.computedAt)}`
      : "Not computed yet — recompute to build the lines",
  );
  if (ret.finalizedAt) {
    facts.push(
      `Finalized ${formatUnixDateTime(ret.finalizedAt)}${
        ret.finalizedBy?.name ? ` by ${ret.finalizedBy.name}` : ""
      }`,
    );
  }
  if (ret.filedAt) {
    facts.push(
      `Filed ${formatUnixDateTime(ret.filedAt)}${
        ret.filingReference ? ` · ${ret.filingReference}` : ""
      }`,
    );
  }
  if (ret.reopenedAt) {
    facts.push(
      `Reopened ${formatUnixDateTime(ret.reopenedAt)}${
        ret.reopenReason ? `: ${ret.reopenReason}` : ""
      }`,
    );
  }
  return facts.join(" · ");
}

export function ReturnSummaryStrip({ ret }: { ret: IftaReturnView }) {
  const t = useT();

  const net = netPosition(ret.netDue);

  return (
    <div className="grid grid-cols-2 gap-2.5 xl:grid-cols-5">
      <StatTile
        label={t("Total miles")}
        value={formatIftaMeasure(ret.totalMiles, IFTA_MILES_DISPLAY_SCALE)}
        sub={`${formatIftaMeasure(ret.totalTaxableMiles, IFTA_MILES_DISPLAY_SCALE)} taxable`}
        hint={t(
          "Every mile attributed to a jurisdiction on a completed move in the quarter, plus manual entries.",
        )}
      />
      <StatTile
        label={t("Tax-paid gallons")}
        value={formatIftaMeasure(ret.totalTaxPaidGallons, IFTA_GALLONS_SCALE)}
        sub={`of ${formatIftaMeasure(ret.totalGallons, IFTA_GALLONS_SCALE)} bought`}
        hint={t(
          "Gallons bought with the fuel tax already paid at the pump. Untaxed purchases count toward the fleet total but earn no credit.",
        )}
      />
      <StatTile
        label={t("Fleet MPG")}
        value={
          <div className="flex items-start gap-1.5">
            <div className="flex flex-col gap-0.5">
              {ret.fleetMpgByFuelType.length === 0 ? (
                <span className="text-muted-foreground text-xs font-normal">
                  {t("No miles run")}
                </span>
              ) : (
                ret.fleetMpgByFuelType.map((entry) => (
                  <span key={entry.fuelType} className="tabular-nums">
                    {entry.mpg === null
                      ? `${IFTA_FUEL_TYPE_LABELS[entry.fuelType]} —`
                      : `${IFTA_FUEL_TYPE_LABELS[entry.fuelType]} ${formatIftaMeasure(entry.mpg, IFTA_MPG_SCALE)}`}
                  </span>
                ))
              )}
            </div>
            <KpiInfoPopover
              title={t("Fleet MPG")}
              description={MPG_EXPLANATION}
              rows={mpgRows(ret)}
            />
          </div>
        }
        sub={t("Miles ÷ gallons, per fuel type")}
        hint={MPG_EXPLANATION}
      />
      <StatTile
        label={t(net.label)}
        value={formatIftaMoney(net.magnitude, ret.currencyCode)}
        sub={`${formatIftaMeasure(ret.taxDue, IFTA_MONEY_SCALE)} tax + ${formatIftaMeasure(ret.surchargeDue, IFTA_MONEY_SCALE)} surcharge, in ${ret.currencyCode}`}
        tone={net.isCredit ? "info" : "warn"}
        hint={
          net.isCredit
            ? "The fleet bought more taxed fuel than it burned, so the base jurisdiction owes it back."
            : "Tax due plus surcharge due across every member jurisdiction on the return."
        }
      />
      <StatTile
        label={t("Status")}
        value={ret.status}
        sub={statusFacts(ret)}
        hint={t(
          "Draft figures move with the data. Finalized locks them. Filed is immutable; corrections open an amendment.",
        )}
      />
    </div>
  );
}
