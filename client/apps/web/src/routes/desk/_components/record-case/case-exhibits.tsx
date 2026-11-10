import { DeskIcon } from "@/components/desk-chat/desk-icons";
import type { DeskAttentionItem } from "@/lib/graphql/desk-attention";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixTime } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { Link } from "react-router";
import { lensLabel } from "../attention/lens";
import { formatFactTime } from "../watchtower/lanes";

/**
 * The evidence beside a case, one numbered exhibit per open signal, on folder
 * tabs as the board draws them: the condition the Watchtower raised with what
 * is at stake and how it got there, the message Triage read, the change an
 * agent proposes. The finding's numbers point here, and an exhibit lights up
 * while its number is pointed at.
 */
export function CaseExhibits({
  items,
  lit,
  now,
  timezone,
}: {
  items: readonly DeskAttentionItem[];
  lit: number | null;
  now: number;
  timezone: string;
}) {
  const t = useT();
  const condition = items.find((item) => item.deskLens === "Watchtower") ?? null;

  return (
    <aside className="dk-w-case-side" aria-label={t("Evidence")}>
      {items.map((item, index) => {
        const n = index + 1;
        return (
          <div key={item.id} id={`dk-w-exh-${n}`} className={cn("dk-w-exh", lit === n && "dk-lit")}>
            <div className="dk-w-exh-tab">
              <span className="dk-w-cite">{n}</span>
              {item.kindLabel}
            </div>
            <div className="dk-w-exh-b">
              <div className="dk-w-exh-src">
                <span>{lensLabel(t, item.deskLens)}</span>
                <span>{formatFactTime(t, item.occurredAt, now, timezone)}</span>
              </div>
              {item.deskLens === "Triage" ? (
                <>
                  <div className="dk-w-quote">{item.summary || item.title}</div>
                  {item.path !== "" && (
                    <Link className="dk-w-alnk dk-w-exh-link ui-focus-ring" to={item.path}>
                      {t("Open in Triage")}
                    </Link>
                  )}
                </>
              ) : (
                <>
                  <div className="dk-w-exh-ti">{item.title}</div>
                  {item.summary !== "" && <div className="dk-w-exh-sum">{item.summary}</div>}
                  {(item.impact.length > 0 || item.facts.length > 0) && (
                    <ul className="dk-w-tl">
                      {item.impact.map((stake, stakeIndex) => (
                        <li key={`i${stakeIndex}`}>
                          <span className="dk-w-tl-n dk-warn" />
                          <span>{stake}</span>
                          <span className="dk-w-tl-w">{t("At stake")}</span>
                        </li>
                      ))}
                      {item.facts.map((fact, factIndex) => (
                        <li key={`f${factIndex}`}>
                          <span className="dk-w-tl-n" />
                          <span>{fact.label}</span>
                          <span className="dk-w-tl-w">
                            {fact.value ??
                              (fact.at !== null ? formatFactTime(t, fact.at, now, timezone) : "")}
                          </span>
                        </li>
                      ))}
                    </ul>
                  )}
                  {item.trail.length > 0 && (
                    <ul className="dk-w-tl">
                      {item.trail.map((step, stepIndex) => (
                        <li key={`t${stepIndex}`}>
                          <span className="dk-w-tl-n dk-done" />
                          <span>{step.text}</span>
                          <span className="dk-w-tl-w">{formatUnixTime(step.at, { timezone })}</span>
                        </li>
                      ))}
                    </ul>
                  )}
                </>
              )}
            </div>
          </div>
        );
      })}
      {condition && (
        <div className="dk-w-exh-note">
          <DeskIcon name="radar" size={12} />
          <span>
            {t(
              "Watchtower raised this {0}. It clears on its own when the record is back on track.",
              formatFactTime(t, condition.occurredAt, now, timezone),
            )}
          </span>
        </div>
      )}
    </aside>
  );
}
