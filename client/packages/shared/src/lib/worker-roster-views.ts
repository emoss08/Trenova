import type { FieldFilter } from "../types/data-table";

/**
 * The questions HR actually opens the roster to answer. Each is a saved filter
 * over the roll-ups the server denormalises onto the worker profile, so a view
 * is one click rather than three filter menus.
 *
 * The ids are stable and appear in URLs people share.
 */
export type RosterViewId =
  | "all"
  | "non-compliant"
  | "training-overdue"
  | "at-risk"
  | "prohibited"
  | "expiring-soon";

export type RosterView = {
  id: RosterViewId;
  label: string;
  /** One line saying what the view answers, for the chip's tooltip. */
  description: string;
  filters: FieldFilter[];
};

/** How far ahead "expiring soon" looks. Matches the credential renewal window. */
export const EXPIRING_SOON_DAYS = 30;

const SECONDS_PER_DAY = 86_400;

/**
 * Builds the views. `now` is passed in rather than read from the clock so the
 * expiry cutoff is deterministic and the caller controls when it moves.
 */
export function rosterViews(now: number): RosterView[] {
  return [
    {
      id: "all",
      label: "Everyone",
      description: "Every worker on the books.",
      filters: [],
    },
    {
      id: "non-compliant",
      label: "Non-compliant",
      description: "A required credential has expired or was never recorded.",
      filters: [{ field: "profile.complianceStatus", operator: "eq", value: "NonCompliant" }],
    },
    {
      id: "training-overdue",
      label: "Training overdue",
      description: "A required course is past due, expired, failed or never assigned.",
      filters: [
        {
          field: "profile.trainingHealth",
          operator: "in",
          value: ["Overdue", "Expired", "Failed", "Missing"],
        },
      ],
    },
    {
      id: "prohibited",
      label: "Prohibited",
      description:
        "A drug or alcohol violation stands unresolved. These drivers must not be dispatched.",
      filters: [{ field: "profile.drugAlcoholStatus", operator: "eq", value: "Prohibited" }],
    },
    {
      id: "at-risk",
      label: "At risk",
      description: "Safety score has fallen far enough to need attention.",
      filters: [{ field: "profile.safetyRating", operator: "eq", value: "AtRisk" }],
    },
    {
      id: "expiring-soon",
      label: `Expiring in ${EXPIRING_SOON_DAYS} days`,
      description: "A required credential lapses inside the renewal window.",
      filters: [
        {
          field: "profile.nextCredentialExpiry",
          operator: "lte",
          value: now + EXPIRING_SOON_DAYS * SECONDS_PER_DAY,
        },
      ],
    },
  ];
}

/**
 * Which view the current filters correspond to, or null when somebody has
 * filtered by hand. Compared on field and operator rather than value, because
 * the expiry cutoff moves with the clock and would otherwise never match.
 */
export function activeRosterView(
  filters: readonly FieldFilter[],
  views: readonly RosterView[],
): RosterViewId | null {
  if (filters.length === 0) return "all";
  const match = views.find(
    (view) =>
      view.filters.length === filters.length &&
      view.filters.length > 0 &&
      view.filters.every((expected, index) => {
        const actual = filters[index];
        return actual?.field === expected.field && actual?.operator === expected.operator;
      }),
  );
  return match?.id ?? null;
}
