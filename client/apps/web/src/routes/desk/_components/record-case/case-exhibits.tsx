import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import type { DeskCase, DeskCaseExhibit } from "@/lib/graphql/desk-case";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixTime } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { Link } from "react-router";
import { formatFactTime } from "../watchtower/lanes";
import { textTone } from "./case-tone";
import { citedExhibits } from "./case-finding";
import { CaseRoute } from "./case-route";

/**
 * The evidence beside a case, on folder tabs as the board draws them. Each
 * exhibit is one piece of evidence, numbered so the reading can cite it:
 * where the truck is, the driver's hours, what the customer wrote, the stops
 * with their projected arrivals, what is owed. An exhibit lights up while
 * its number is pointed at. The Watchtower's own line sits at the foot.
 */
export function CaseExhibits({
  data,
  lit,
  now,
  timezone,
}: {
  data: DeskCase;
  lit: number | null;
  now: number;
  timezone: string;
}) {
  const t = useT();
  // Only once there is a reading: before it, nothing has been cited yet and
  // no exhibit is marked as left out.
  const cited = citedExhibits(data);
  const reading = cited.size > 0;
  // A case with nothing to show as evidence draws no evidence column at all,
  // rather than an empty one beside the reading.
  if (data.exhibits.length === 0) {
    return null;
  }

  return (
    <ScrollArea className="dk-cz-col" maskVariant="card">
      <aside className="dk-cz-side" aria-label={t("Evidence")}>
        {data.exhibits.map((exhibit) => (
          <div
            key={exhibit.key}
            id={`dk-cz-exh-${exhibit.number}`}
            className={cn(
              "dk-cz-exh",
              lit === exhibit.number && "dk-lit",
              reading && !cited.has(exhibit.number) && "dk-uncited",
            )}
          >
            <div className="dk-cz-exh-tab">
              <span className="dk-cz-cite">{exhibit.number}</span>
              {exhibit.title}
            </div>
            <div className="dk-cz-exh-b">
              <div className="dk-cz-exh-src">
                <span>
                  {reading && !cited.has(exhibit.number)
                    ? t("{0} · not cited in the finding", exhibit.source)
                    : exhibit.source}
                </span>
                {exhibit.at != null && (
                  <span>
                    {exhibit.read
                      ? t("read {0}", formatUnixTime(exhibit.at, { timezone }))
                      : formatFactTime(t, exhibit.at, now, timezone)}
                  </span>
                )}
              </div>
              <ExhibitBody exhibit={exhibit} now={now} timezone={timezone} />
            </div>
          </div>
        ))}
        {data.note !== "" && (
          <div className="dk-cz-foot">
            <DeskIcon name="radar" size={12} />
            <span>{data.note}</span>
          </div>
        )}
      </aside>
    </ScrollArea>
  );
}

function ExhibitBody({
  exhibit,
  now,
  timezone,
}: {
  exhibit: DeskCaseExhibit;
  now: number;
  timezone: string;
}) {
  const t = useT();

  switch (exhibit.kind) {
    case "Route":
      return (
        <>
          {exhibit.route && <CaseRoute route={exhibit.route} />}
          {exhibit.note !== "" && <div className="dk-cz-exh-line">{exhibit.note}</div>}
        </>
      );
    case "Meters":
      return (
        <>
          {exhibit.meters.map((meter) => (
            <div className="dk-cz-meter" key={meter.label}>
              <div className="dk-cz-meter-row">
                <span>{meter.label}</span>
                <span className="dk-cz-num">{meter.value}</span>
              </div>
              <div className="dk-cz-meter-tr">
                <i
                  className={cn(
                    meter.tone === "Warning" && "dk-warn",
                    meter.tone === "Bad" && "dk-bad",
                    meter.tone === "Good" && "dk-ok",
                  )}
                  style={{
                    width: `${Math.round(Math.min(1, Math.max(0, meter.fraction)) * 100)}%`,
                  }}
                />
              </div>
            </div>
          ))}
          {exhibit.note !== "" && <div className="dk-cz-exh-quiet">{exhibit.note}</div>}
        </>
      );
    case "Message":
      return (
        <>
          <div className="dk-cz-quote">{exhibit.message?.quote}</div>
          {exhibit.message?.replyAgent && (
            <div className="dk-cz-exh-quiet">
              <DeskAgentTile agent={exhibit.message.replyAgent} size="xs" />
              <span>{t("{0} {1}", exhibit.message.replyAgent.name, exhibit.message.reply)}</span>
            </div>
          )}
          {exhibit.path !== "" && (
            <div className="dk-cz-exh-quiet">
              <Link className="dk-w-alnk ui-focus-ring" to={exhibit.path}>
                {t("Open in Triage")}
              </Link>
            </div>
          )}
        </>
      );
    case "Timeline":
      return (
        <ul className="dk-cz-tl">
          {exhibit.timeline.map((row, index) => (
            <li key={`${index}:${row.label}`}>
              <span
                className={cn(
                  "dk-cz-tl-n",
                  row.state === "Done" && "dk-done",
                  row.state === "Proposed" && "dk-new",
                )}
              />
              <span>
                {row.label}
                {row.state === "Proposed" && (
                  <>
                    {" "}
                    <span className="dk-cz-tl-prop">{t("proposed")}</span>
                  </>
                )}
                {row.sub !== "" && <span className="dk-cz-tl-sub">{row.sub}</span>}
              </span>
              <span className={cn("dk-cz-tl-w", textTone(row.tone))}>{row.when}</span>
            </li>
          ))}
        </ul>
      );
    case "Facts":
      return (
        <>
          {exhibit.note !== "" && <div className="dk-cz-quote">{exhibit.note}</div>}
          <dl className="dk-cz-facts">
            {exhibit.facts.map((fact, index) => (
              <div key={`${index}:${fact.label}`}>
                <dt>{fact.label}</dt>
                <dd className={cn(textTone(fact.tone))}>
                  {fact.value !== ""
                    ? fact.value
                    : fact.at != null
                      ? formatFactTime(t, fact.at, now, timezone)
                      : ""}
                </dd>
              </div>
            ))}
          </dl>
          {exhibit.path !== "" && (
            <div className="dk-cz-exh-quiet">
              <Link className="dk-w-alnk ui-focus-ring" to={exhibit.path}>
                {t("Open it")}
              </Link>
            </div>
          )}
        </>
      );
  }
}
