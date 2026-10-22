import { formatNumber } from "@trenova/shared/i18n/format";
import { translate } from "@trenova/shared/i18n/runtime";
import { metersToMiles } from "@trenova/shared/lib/utils";
import type { LaneRuleKeys, LaneSegment } from "@trenova/shared/rules/lane";
import type { RateAgreementRule, RateScopeType } from "../types/rate";

/**
 * How a rate agreement lane reads to a person. The lane rules themselves (the key a rule
 * stores, its specificity, which lanes can never apply, and how a stored key splits into its
 * ends) run in the rules module through `@trenova/shared/rules/lane`; what is left here puts
 * translated record names and labels into what the module returns.
 */

/**
 * Resolves an id-backed scope value — a state, zone or location id — to the
 * record's name. Returning undefined means the name is not known yet, which the
 * display renders as pending rather than falling back to the id.
 */
export type LaneScopeLabelResolver = (type: RateScopeType, value: string) => string | undefined;

/**
 * A lane key's ends as a person should read them: the same shape the engine
 * matches on, with every record id replaced by the record's name. Ids must never
 * reach the screen — a value whose name has not resolved yet reads as an
 * ellipsis — and everything literal is left exactly as stored.
 */
export function laneSegmentsDisplay(
  segments: readonly LaneSegment[],
  resolveScopeValue: LaneScopeLabelResolver,
): string {
  return segments
    .map((segment) => {
      if (!segment.scope || !segment.id) return segment.text;
      const name = resolveScopeValue(segment.scope, segment.id) ?? "…";
      return `${segment.prefix ?? ""}${name}${segment.suffix ?? ""}`;
    })
    .join(">");
}

/** A rule's lane key as a person reads it, or null for a lane matched by radius. */
export function laneKeyDisplay(
  keys: LaneRuleKeys,
  resolveScopeValue: LaneScopeLabelResolver,
): string | null {
  return keys.key === null ? null : laneSegmentsDisplay(keys.segments, resolveScopeValue);
}

/** What follows a key end's prefix: "60601" of "Z5:60601". */
function endValue(key: string | null): string | null {
  return key === null ? null : key.slice(key.indexOf(":") + 1);
}

/**
 * One end of a lane as a person reads it: "Dallas, TX", "GA", "ZIP 303xx",
 * "Anywhere". A record the end names reads by the record's name, and one whose
 * name has not resolved yet reads as an ellipsis, never as its id. Postal codes
 * and countries read as the engine keys them (`keys`, from `laneKeys`), so the
 * card shows what the lane matches on.
 */
export function laneEndDisplay(
  rule: RateAgreementRule,
  end: "origin" | "destination",
  resolveScopeValue: LaneScopeLabelResolver,
  keys: LaneRuleKeys,
): string {
  const type = end === "origin" ? rule.originScopeType : rule.destinationScopeType;
  const value = (end === "origin" ? rule.originScopeValue : rule.destinationScopeValue).trim();
  const city = (end === "origin" ? rule.originCity : rule.destinationCity).trim();
  const keyed = endValue(end === "origin" ? keys.origin : keys.destination);
  const named = () => resolveScopeValue(type, value) ?? "…";

  switch (type) {
    case "Any":
      return translate("Anywhere");
    case "Location":
    case "Zone":
    case "State":
      return value ? named() : translate("Anywhere");
    case "CityState":
      return value && city ? `${city}, ${named()}` : city || translate("Anywhere");
    case "Zip5":
      return translate("ZIP {0}", keyed ?? (value || "…"));
    case "Zip3":
      return translate("ZIP {0}xx", keyed ?? (value || "…"));
    case "Country":
      return keyed ?? value;
    case "Radius": {
      const meters = Number(
        end === "origin" ? rule.originRadiusMeters : rule.destinationRadiusMeters,
      );
      const miles = Number.isFinite(meters) && meters > 0 ? metersToMiles(meters) : null;
      return miles === null
        ? city || translate("Anywhere")
        : translate(
            "Within {0} mi of {1}",
            formatNumber(miles, { maximumFractionDigits: 0 }),
            city || "…",
          );
    }
    default:
      return translate("Anywhere");
  }
}

/** The label a lane shows when nobody has named it. */
export function laneDisplayLabel(rule: RateAgreementRule, index: number): string {
  return rule.label || translate("Lane {0}", index + 1);
}
