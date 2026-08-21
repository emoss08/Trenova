import type { Shipment } from "@trenova/shared/types/shipment";

/**
 * The ids every rating mutation parses as non-null.
 *
 * GraphQL declares them `ID!`, which only guarantees the key is present — a
 * half-filled form posts an empty string and the server rejects the whole
 * mutation. The preview is speculative by nature, so it waits instead.
 */
const REQUIRED_IDS = [
  "customerId",
  "serviceTypeId",
  "shipmentTypeId",
] as const satisfies readonly (keyof Shipment)[];

/**
 * A move needs both ends of the lane before its mileage means anything, and
 * distance resolution reads them as a pair.
 */
const MIN_STOPS_PER_MOVE = 2;

function isFilled(value: unknown): value is string {
  return typeof value === "string" && value.length > 0;
}

/**
 * Whether the form holds enough to compute a total.
 *
 * Totals are produced by the shipment's own rating method, so that method has
 * to have been chosen — by the rater or by the contract — before there is
 * anything to calculate.
 */
export function isRatable(values: Shipment): boolean {
  return (
    REQUIRED_IDS.every((field) => isFilled(values[field])) && isFilled(values.formulaTemplateId)
  );
}

/**
 * Whether the form holds enough for the contracts to be asked what they charge.
 *
 * Deliberately not `isRatable`: the rating method is what a contract hands
 * over, so waiting for one before asking is waiting for the answer before
 * asking the question, and no new shipment ever gets auto-rated. What a
 * contract is resolved by is the party and the lane, and those are what this
 * requires.
 */
export function canResolveContract(values: Shipment): boolean {
  return REQUIRED_IDS.every((field) => isFilled(values[field])) && hasCompleteLane(values);
}

/**
 * Whether both ends of the lane are known.
 *
 * Which rate agreement covers a load is resolved from its origin and its
 * destination, so a contract chosen from half a lane is the wrong contract.
 * Pricing tolerates a partial lane — the mileage is simply lower — but choosing
 * a contract does not.
 */
export function hasCompleteLane(values: Shipment): boolean {
  return (values.moves ?? []).some(
    (move) =>
      (move.stops ?? []).filter((stop) => isFilled(stop.locationId)).length >= MIN_STOPS_PER_MOVE,
  );
}

/**
 * Strip a shipment down to the rows the rating mutations can actually parse.
 *
 * A row the user has started but not finished — a stop with no location, a
 * commodity line with no commodity — carries an empty id that fails server-side
 * parsing and takes the whole preview down with it. Dropping those rows prices
 * what has been entered so far and lets the total fill in as the rest is typed.
 *
 * Relation objects are dropped too: the inputs take ids, and echoing the
 * hydrated relation back inflates every request for nothing.
 */
export function toRatingPreviewPayload(values: Shipment): Shipment {
  const moves = (values.moves ?? [])
    .map((move) => ({
      ...move,
      stops: (move.stops ?? []).filter((stop) => isFilled(stop.locationId)),
    }))
    .filter((move) => move.stops.length >= MIN_STOPS_PER_MOVE);

  const additionalCharges = (values.additionalCharges ?? [])
    .filter((charge) => isFilled(charge.accessorialChargeId))
    // oxlint-disable-next-line no-unused-vars
    .map(({ accessorialCharge, ...rest }) => rest);

  const commodities = (values.commodities ?? [])
    .filter((line) => isFilled(line.commodityId))
    // oxlint-disable-next-line no-unused-vars
    .map(({ commodity, ...rest }) => rest);

  return { ...values, moves, additionalCharges, commodities } as Shipment;
}
