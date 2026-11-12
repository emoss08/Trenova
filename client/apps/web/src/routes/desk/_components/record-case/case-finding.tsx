import { DeskIcon } from "@/components/desk-chat/desk-icons";
import type { DeskCase, DeskCaseSegment } from "@/lib/graphql/desk-case";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixTime } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";

type FindingProps = {
  data: DeskCase;
  timezone: string;
  lit: number | null;
  onLight: (n: number | null) => void;
};

/**
 * What was found about the record. With Nova's reading: who found it and
 * from how many sources, the verdict in plain words with the fact that
 * decides it marked and each claim numbered to the exhibit it rests on, then
 * the four things that make it safe to act on. Without one, what the
 * Watchtower raised, said plainly; the plan below can still be decided.
 */
export function CaseFinding({ data, timezone, lit, onLight }: FindingProps) {
  const t = useT();
  const finding = data.finding;

  if (finding === null) {
    return (
      <div className="dk-cz-find">
        <div className="dk-w-banner">
          <DeskIcon name="check" size={14} />
          <span>{t("Nothing is open about this record. Ask below if something looks wrong.")}</span>
        </div>
      </div>
    );
  }

  const agentName = finding.agent?.name ?? null;
  const found = formatUnixTime(finding.foundAt, { timezone });
  const read = finding.readingStatus === "Ready" && finding.reading.length > 0;
  const writing = finding.readingStatus === "Writing";
  const cited = citedExhibits(data);
  // A record whose conditions are all closed can still have a decision
  // waiting: there is nothing to read, and saying a reading is unavailable
  // would suggest something is wrong.
  const nothingRaised = finding.raised === "" && finding.computed === "";

  return (
    <div className="dk-cz-find">
      {read || writing ? (
        <>
          <div className="dk-cz-who">
            <NovaBadge size="lg" />
            <span>
              <b>{agentName ?? t("Nova")}</b>{" "}
              {read && cited.size < finding.sources
                ? t(
                    "found this at {0}, citing {1} of {2, plural, one {# source} other {# sources}}",
                    found,
                    cited.size,
                    finding.sources,
                  )
                : t(
                    "found this at {0} from {1, plural, one {# source} other {# sources}}",
                    found,
                    finding.sources,
                  )}
            </span>
          </div>
          {read ? (
            <p className="dk-cz-verdict">
              {finding.reading.map((segment, index) => (
                <Segment key={index} segment={segment} lit={lit} onLight={onLight} />
              ))}
            </p>
          ) : (
            <div className="dk-cz-verdict-sk" aria-busy aria-label={t("Nova is reading the case")}>
              <i className="dk-sk" />
              <i className="dk-sk" />
            </div>
          )}
        </>
      ) : (
        <div className="dk-w-banner">
          <DeskIcon name={nothingRaised ? "check" : "radar"} size={14} />
          <span hidden={!nothingRaised}>
            {data.decision?.phase === "Awaiting"
              ? t("Nothing else is open about this record. A change waits on you below.")
              : t("Nothing else is open about this record.")}
          </span>
          <span hidden={nothingRaised}>
            {finding.raised !== "" && <b className="font-medium">{finding.raised}</b>}{" "}
            {finding.computed}{" "}
            {data.decision?.phase === "Awaiting"
              ? t(
                  "Nova's reading is unavailable; the plan below can still be decided, and every change can be made by hand on the record.",
                )
              : t(
                  "Nova's reading is unavailable; every change can still be made by hand on the record.",
                )}
          </span>
        </div>
      )}
      {finding.trust.length > 0 && (
        <div className="dk-cz-trust">
          {finding.trust.map((cell) => (
            <div key={cell.key}>
              <span className="dk-cz-trust-k">
                {cell.key === "Matters"
                  ? t("Why it matters")
                  : cell.key === "Unchanged"
                    ? t("If nothing changes")
                    : cell.key === "Proposes"
                      ? t("{0} proposes", agentName ?? t("The agent"))
                      : t("Needs your approval because")}
              </span>
              {cell.text}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function Segment({
  segment,
  lit,
  onLight,
}: {
  segment: DeskCaseSegment;
  lit: number | null;
  onLight: (n: number | null) => void;
}) {
  const t = useT();
  const n = segment.cite ?? null;
  if (n === null) {
    return segment.highlight ? <span className="dk-cz-hl">{segment.text}</span> : segment.text;
  }

  return (
    <button
      type="button"
      className={cn("dk-cz-cite ui-focus-ring", lit === n && "dk-on")}
      aria-label={t("Evidence {0}", n)}
      onMouseEnter={() => onLight(n)}
      onMouseLeave={() => onLight(null)}
      onFocus={() => onLight(n)}
      onBlur={() => onLight(null)}
      onClick={() => showExhibit(n)}
    >
      {n}
    </button>
  );
}

/**
 * Brings the evidence a claim cites into view in its own column, wherever
 * that column is scrolled to, without moving the reading the person is in.
 */
function showExhibit(n: number): void {
  const still = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  document
    .getElementById(`dk-cz-exh-${n}`)
    ?.scrollIntoView({ block: "start", behavior: still ? "auto" : "smooth" });
}

/**
 * The exhibits the reading rests a claim on. A reading says what is wrong in
 * a few sentences and cites only what those sentences draw on, so some of
 * the evidence beside a case is there to be looked at, not because the
 * finding used it. Empty while there is no reading.
 */
export function citedExhibits(data: DeskCase): ReadonlySet<number> {
  const cited = new Set<number>();
  const finding = data.finding;
  if (finding && finding.readingStatus === "Ready") {
    for (const segment of finding.reading) {
      if (segment.cite != null) {
        cited.add(segment.cite);
      }
    }
  }

  return cited;
}
