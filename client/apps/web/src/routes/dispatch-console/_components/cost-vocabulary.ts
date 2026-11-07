import type {
  DispatchBoardMove,
  DispatchCandidate,
  DispatchEndMarket,
  DispatchForecastCustomer,
  DispatchUncoveredReason,
} from "@/lib/graphql/dispatch-console";
import { formatList, formatNumber } from "@trenova/shared/i18n/format";
import { defineLabels, type LabelMap } from "@trenova/shared/i18n/labels";
import { translate, type TranslateFn } from "@trenova/shared/i18n/runtime";
import { formatClockDurationMs, formatUnixDayTimeRange } from "@trenova/shared/lib/date";
import { formatCentsWhole } from "@trenova/shared/lib/utils";

/** What each part of a pairing's cost is called, as the server's cost components name them. */
export const COST_COMPONENT_LABELS = defineLabels({
  emptyMiles: "Empty miles",
  lateness: "Lateness",
  idleWait: "Waiting",
  rest: "Rest before driving",
  homeTime: "Time away from home",
  fairness: "Miles above the fleet",
  customerExperience: "New to the customer",
  onTimeHistory: "On-time history",
  safety: "Safety history",
  acceptance: "Acceptance",
  trailerContinuity: "Trailer swap",
  haulFit: "Haul fit",
  timeOffBuffer: "Close to time off",
  endMarket: "Where they end up",
});

export type CostComponent = keyof typeof COST_COMPONENT_LABELS;

export function costComponentLabel(component: string): string {
  return component in COST_COMPONENT_LABELS
    ? COST_COMPONENT_LABELS[component as CostComponent]
    : component;
}

/**
 * Why a plan left a move uncovered, named so it reads as a heading over one move or over a
 * group of them. Typed by the schema's enum, so a reason the server adds fails to compile here.
 */
export const UNCOVERED_REASON_LABELS: LabelMap<Record<DispatchUncoveredReason, string>> =
  defineLabels({
    NoCandidate: "No eligible driver",
    Disqualified: "Nearest drivers disqualified",
    BeyondDeadhead: "Past the empty-mile limit",
    LocationUnknown: "No driver location known",
    NotWorthIt: "Not worth covering",
    Outbid: "Drivers worth more elsewhere",
  });

/**
 * What the move's last plan says the best driver for it pays in preferences. A candidate
 * costs the plan its own cost less this, which is the number the plan weighed against the
 * coverage price; zero for a move no plan has priced or that is already covered.
 */
export function preferenceFloorOf(move: DispatchBoardMove): number {
  return move.isCovered ? 0 : (move.planPrice?.preferenceFloorCents ?? 0);
}

/** The fact behind a cost line, from the candidate it belongs to, when there is one to show. */
export function costLineFact(component: string, candidate: DispatchCandidate): string | null {
  switch (component) {
    case "emptyMiles":
      return candidate.deadheadMiles == null
        ? null
        : translate("{0} mi", Math.round(candidate.deadheadMiles).toLocaleString());
    case "lateness":
      return candidate.deliveryLateMinutes > 0
        ? translate(
            "{0} late to delivery",
            formatClockDurationMs(candidate.deliveryLateMinutes * 60_000),
          )
        : candidate.minutesOfSlack < 0
          ? translate(
              "{0} late to pickup",
              formatClockDurationMs(-candidate.minutesOfSlack * 60_000),
            )
          : null;
    case "rest":
      return candidate.restMs > 0 ? formatClockDurationMs(candidate.restMs) : null;
    default:
      return null;
  }
}

const FORECAST_CUSTOMERS_SHOWN = 3;

/**
 * Who usually brings the freight a forecast expects, with the loads each usually brings:
 * "Mostly from Acme Foods (about 3) and Beta Co (about 1)." Null when no customer stands out.
 */
export function forecastCustomersSentence(
  customers: readonly DispatchForecastCustomer[],
  t: TranslateFn,
): string | null {
  const shown = customers.slice(0, FORECAST_CUSTOMERS_SHOWN);
  if (shown.length === 0) return null;
  return t(
    "Mostly from {0}.",
    formatList(
      shown.map((customer) =>
        t(
          "{0} (about {1})",
          customer.name,
          formatNumber(customer.expectedLoads, { maximumFractionDigits: 1 }),
        ),
      ),
    ),
  );
}

/**
 * Why the endMarket line is worth what it is, one sentence per line. A forecast names the
 * area, the loads that usually appear there in the window after the driver is free, who
 * usually brings them, and what is already booked and who else is expected, then, when
 * staying put was worth something too, what the move adds or costs against that; history
 * says the credit is what drivers free there have been worth to past plans.
 */
export function endMarketReason(endMarket: DispatchEndMarket, t: TranslateFn): string[] {
  if (endMarket.source === "History") {
    return [t("Drivers free here have been worth this to past plans.")];
  }

  const lines = forecastDropSentences(endMarket, t);
  const stay = staySentence(endMarket, t);
  if (stay) {
    lines.push(stay);
  }
  return lines;
}

/**
 * The credit is relative: what the driver is worth where the move leaves them less what they
 * were worth staying where they are. When staying was worth something, this says how much and
 * whether the move adds to that or costs the plan, comparing whole dollars as the line shows
 * them. Null when staying was worth nothing, since the credit is then the drop's value alone.
 */
function staySentence(endMarket: DispatchEndMarket, t: TranslateFn): string | null {
  if (endMarket.stayValueCents <= 0) return null;

  const stay = formatCentsWhole(endMarket.stayValueCents);
  const net = formatCentsWhole(Math.abs(endMarket.valueCents));
  const netDollars = Math.round(endMarket.valueCents / 100);
  const area = endMarket.stayAreaLabel;

  if (netDollars > 0) {
    return area
      ? t("Staying near {0} was worth {1}, so the move adds {2}.", area, stay, net)
      : t("Staying where they are was worth {0}, so the move adds {1}.", stay, net);
  }
  if (netDollars < 0) {
    return area
      ? t(
          "Staying near {0} was worth {1}, more than ending here, so the move costs {2}.",
          area,
          stay,
          net,
        )
      : t(
          "Staying where they are was worth {0}, more than ending here, so the move costs {1}.",
          stay,
          net,
        );
  }
  return area
    ? t("Staying near {0} was worth {1}, as much as ending here.", area, stay)
    : t("Staying where they are was worth {0}, as much as ending here.", stay);
}

function forecastDropSentences(endMarket: DispatchEndMarket, t: TranslateFn): string[] {
  const { areaLabel, windowStart, windowEnd, typicalLow, typicalHigh } = endMarket;
  if (!areaLabel || !windowStart || !windowEnd || typicalLow == null || typicalHigh == null) {
    return [
      t(
        "Freight your history says is likely near the drop, less what is booked and the other drivers expected there.",
      ),
    ];
  }

  const hours = formatUnixDayTimeRange(windowStart, windowEnd);
  const lines: string[] = [
    typicalLow === typicalHigh
      ? t(
          "Near {0}: {1, plural, one {# load usually appears} other {# loads usually appear}} {2}.",
          areaLabel,
          typicalHigh,
          hours,
        )
      : typicalLow <= 0
        ? t(
            "Near {0}: up to {1, plural, one {# load usually appears} other {# loads usually appear}} {2}.",
            areaLabel,
            typicalHigh,
            hours,
          )
        : t(
            "Near {0}: {1}–{2} loads usually appear {3}.",
            areaLabel,
            formatNumber(typicalLow),
            formatNumber(typicalHigh),
            hours,
          ),
  ];

  const customers = forecastCustomersSentence(endMarket.customers, t);
  if (customers) {
    lines.push(customers);
  }

  if (endMarket.booked != null && endMarket.driversExpected != null) {
    lines.push(
      t(
        "{0, plural, other {# booked}}; {1, plural, =0 {no other driver expected} one {# other driver expected} other {# other drivers expected}}.",
        endMarket.booked,
        endMarket.driversExpected,
      ),
    );
  }

  return lines;
}
