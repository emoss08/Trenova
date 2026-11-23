import type { WorkerEmploymentEventRow } from "@/lib/graphql/worker-employment";
import { ArrowRightIcon } from "@trenova/shared/components/icons";
import { translate } from "@trenova/shared/i18n/runtime";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import {
  EMPLOYMENT_VALUE_HIDDEN_KEYS,
  EMPLOYMENT_VALUE_LABELS,
} from "@trenova/shared/types/worker-employment";

export type ValuePair = { key: string; label: string; from?: string; to?: string };

/** The enum values an event carries, in words; free values (a fleet code, a rate) read as typed. */
const EMPLOYMENT_VALUE_WORDS: Record<string, string> = defineLabels({
  Active: "Active",
  Inactive: "Inactive",
  Onboarding: "Onboarding",
  OnLeave: "On leave",
  Suspended: "Suspended",
  Terminated: "Terminated",
  Local: "Local",
  Regional: "Regional",
  OTR: "OTR",
  Team: "Team",
  Employee: "Employee",
  Contractor: "Contractor",
  FMLA: "FMLA",
  Medical: "Medical",
  Military: "Military",
  Parental: "Parental",
  Personal: "Personal",
  Other: "Other",
});

const ENUM_VALUE_KEYS: ReadonlySet<string> = new Set([
  "status",
  "driverType",
  "workerType",
  "leaveType",
]);

function formatValue(key: string, value: string): string {
  if (key === "hireDate" || key === "terminationDate") {
    const unix = Number(value);
    return Number.isFinite(unix) && unix > 0 ? formatUnixDateMedium(unix) : value;
  }
  if (key === "canBeAssigned") return value === "true" ? translate("Yes") : translate("No");
  if (ENUM_VALUE_KEYS.has(key)) return EMPLOYMENT_VALUE_WORDS[value] ?? value;
  return value;
}

/**
 * Pairs from/to values by key. A raw id is hidden when its labelled twin is present, and a key
 * the client has no name for is left out rather than shown by its code name.
 */
export function pairValues(
  event: Pick<WorkerEmploymentEventRow, "fromValues" | "toValues">,
): ValuePair[] {
  const keys = new Set<string>();
  const from = new Map(event.fromValues.map((entry) => [entry.key, entry.value]));
  const to = new Map(event.toValues.map((entry) => [entry.key, entry.value]));
  for (const key of [...from.keys(), ...to.keys()]) {
    if (EMPLOYMENT_VALUE_HIDDEN_KEYS.has(key)) continue;
    if (!(key in EMPLOYMENT_VALUE_LABELS)) continue;
    keys.add(key);
  }
  const pairs: ValuePair[] = [];
  for (const key of keys) {
    const fromValue = from.get(key);
    const toValue = to.get(key);
    if ((fromValue == null || fromValue === "") && (toValue == null || toValue === "")) continue;
    pairs.push({
      key,
      label: EMPLOYMENT_VALUE_LABELS[key],
      from: fromValue != null && fromValue !== "" ? formatValue(key, fromValue) : undefined,
      to: toValue != null && toValue !== "" ? formatValue(key, toValue) : undefined,
    });
  }
  return pairs;
}

/** A from → to change as a chip: the label, the old value struck through, the new one. */
export function ValueChips({ pairs }: { pairs: readonly ValuePair[] }) {
  if (pairs.length === 0) return null;
  return (
    <ul className="flex flex-wrap gap-1">
      {pairs.map((pair) => (
        <li
          key={pair.key}
          className="flex items-center gap-1 rounded-sm bg-sunken px-1.5 text-xs leading-5"
        >
          <span className="text-muted-foreground">{pair.label}</span>
          {pair.from ? <del className="text-muted-foreground">{pair.from}</del> : null}
          {pair.from && pair.to ? (
            <ArrowRightIcon aria-hidden className="size-3 text-muted-foreground" />
          ) : null}
          {pair.to ? <ins className="no-underline">{pair.to}</ins> : null}
        </li>
      ))}
    </ul>
  );
}
