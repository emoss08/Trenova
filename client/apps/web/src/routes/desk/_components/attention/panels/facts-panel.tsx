import type { WatchtowerItem } from "@/lib/graphql/watchtower";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatFactTime } from "../../watchtower/lanes";
import { PanelCard } from "./panel-parts";

/**
 * An item with no view made for its kind, read from what its source reported:
 * what is at stake, the facts that matter, and how it got here. Kept to the
 * same frame as the made-for-it panels, so it reads as one of them.
 */
export function FactsPanel({
  item,
  now,
  timezone,
}: {
  item: Pick<WatchtowerItem, "kindLabel" | "impact" | "facts" | "trail">;
  now: number;
  timezone: string;
}) {
  const t = useT();
  const rows = item.facts
    .map((fact) => ({
      label: fact.label,
      value: fact.value ?? (fact.at != null ? formatFactTime(t, fact.at, now, timezone) : ""),
    }))
    .filter((row) => row.value !== "");
  if (rows.length === 0 && item.trail.length === 0 && item.impact.length === 0) {
    return null;
  }

  return (
    <PanelCard
      eyebrow={item.kindLabel}
      title={item.impact.length > 0 ? item.impact.join(" · ") : t("What the Watchtower found")}
    >
      {rows.length > 0 && (
        <dl className="grid grid-cols-[minmax(6rem,auto)_minmax(0,1fr)] gap-x-4 gap-y-1.5 px-3.5 pb-3 text-dsk-base">
          {rows.map((row) => (
            <div key={row.label} className="contents">
              <dt className="text-dsk-subtle">{row.label}</dt>
              <dd className="min-w-0 text-dsk-fg">{row.value}</dd>
            </div>
          ))}
        </dl>
      )}
      {item.trail.length > 0 && (
        <ol
          aria-label={t("How it got here")}
          className="flex flex-col gap-1.5 border-t border-dsk-b-sub px-3.5 py-2.5 text-dsk-sm"
        >
          {item.trail.map((step, index) => (
            <li key={index} className="flex items-baseline gap-2.5">
              <span
                aria-hidden
                className="size-1.5 flex-none translate-y-[-1px] rounded-full bg-dsk-b-strong"
              />
              <span className="text-dsk-fg2">{step.text}</span>
              {step.at ? (
                <span className="ml-auto flex-none text-xs text-dsk-faint tabular-nums">
                  {formatFactTime(t, step.at, now, timezone)}
                </span>
              ) : null}
            </li>
          ))}
        </ol>
      )}
    </PanelCard>
  );
}
