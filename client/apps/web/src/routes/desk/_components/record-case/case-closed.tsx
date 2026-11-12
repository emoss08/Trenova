import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { fetchWatchtowerFeed, type WatchtowerItem } from "@/lib/graphql/watchtower";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import type { WatchtowerResolutionCode } from "@trenova/graphql/generated/graphql";
import { useNavigate } from "react-router";
import { CaseFooter } from "./case-footer";

/** How many of the things last closed about the record the page names. */
const CLOSED_SHOWN = 4;

/** How a thing was closed, in the words a person would use for it. */
const HOW: Record<WatchtowerResolutionCode, string> = defineLabels({
  CarrierDelay: "Put down to a carrier delay",
  Cleared: "Cleared on its own once the record was back on track",
  CustomerChange: "Put down to a change the customer made",
  DataCorrected: "The record was corrected",
  DocumentReceived: "The document was filed",
  DriverIssue: "Put down to a driver issue",
  Duplicate: "Closed as a duplicate",
  Equipment: "Put down to equipment",
  Fixed: "Fixed",
  NoActionNeeded: "Closed as needing nothing",
  Other: "Closed",
  ReceiverDelay: "Put down to a receiver delay",
  Rescheduled: "Rescheduled",
  ShipperDelay: "Put down to a shipper delay",
  Weather: "Put down to weather",
});

function when(at: number | null | undefined): string {
  return at
    ? formatUnixInUserTimezone(
        at,
        { month: "short", day: "numeric", hour: "numeric", minute: "2-digit" },
        "",
      )
    : "";
}

/** How one thing was closed, by whom and when, in a line. */
export function closedLine(t: TranslateFn, item: WatchtowerItem): string {
  const how = item.resolutionCode ? t(HOW[item.resolutionCode]) : t("Closed");
  const by = item.resolvedBy?.name ?? "";
  const at = when(item.resolvedAt);
  if (by !== "" && at !== "") {
    return t("{0} · {1} · {2}", how, by, at);
  }

  return at !== "" ? t("{0} · {1}", how, at) : how;
}

/**
 * A case with nothing open. A person who has just closed one (filed the
 * paper, approved the plan) used to land on one grey line that said nothing
 * was open, and nothing about what they had done. Here the page says it is
 * closed, names what was closed and how, and offers where to go next.
 */
export function CaseClosed({
  recordId,
  label,
  recordPath,
}: {
  recordId: string;
  label: string;
  recordPath: string | null;
}) {
  const t = useT();
  const navigate = useNavigate();
  const closed = useQuery({
    queryKey: ["watchtower", "closed-about", recordId],
    queryFn: ({ signal }) =>
      fetchWatchtowerFeed(
        { subjectId: recordId, statuses: ["Resolved"], unresolvedOnly: false, first: CLOSED_SHOWN },
        { signal },
      ),
  });
  const items = closed.data?.items ?? [];

  return (
    <section className="dk-cz-plan" aria-label={t("Closed")} data-slot="case-closed">
      <div className="flex items-start gap-3 px-4 py-4">
        <span className="mt-0.5 grid size-6 flex-none place-items-center rounded-full bg-success-subtle text-success">
          <DeskIcon name="check" size={13} stroke={2.4} />
        </span>
        <div className="min-w-0">
          <h3 className="text-dsk-base font-semibold text-dsk-fg">
            {label !== ""
              ? t("Nothing is open about {0}.", label)
              : t("Nothing is open about this record.")}
          </h3>
          <p className="mt-0.5 text-dsk-sm text-dsk-subtle">
            {items.length > 0
              ? t(
                  "It is off Needs you for everyone. If anything new is raised, the case opens again.",
                )
              : t("Nothing has been raised about it. If anything is, the case opens here.")}
          </p>
        </div>
      </div>
      {closed.isPending ? (
        <div className="flex flex-col gap-2 border-t border-dsk-b-sub px-4 py-3" aria-busy>
          <i className="h-3.5 w-2/3 ui-shimmer rounded-[4px]" />
          <i className="h-3 w-1/2 ui-shimmer rounded-[4px]" />
        </div>
      ) : (
        items.length > 0 && (
          <ul className="flex flex-col border-t border-dsk-b-sub">
            {items.map((item) => (
              <li
                key={item.id}
                className="flex min-w-0 flex-col gap-0.5 border-b border-dsk-b-sub px-4 py-2.5 last:border-b-0"
              >
                <span className="truncate text-dsk-sm text-dsk-fg">{item.title}</span>
                <span className="truncate text-dsk-sm text-dsk-subtle">{closedLine(t, item)}</span>
              </li>
            ))}
          </ul>
        )
      )}
      <CaseFooter note={t("Something still look wrong? Ask about it below.")}>
        {recordPath && (
          <Button variant="secondary" onClick={() => void navigate(recordPath)}>
            {t("Open the record")}
          </Button>
        )}
        <Button onClick={() => void navigate("/desk/needs", { viewTransition: true })}>
          {t("Back to Needs you")}
        </Button>
      </CaseFooter>
    </section>
  );
}
