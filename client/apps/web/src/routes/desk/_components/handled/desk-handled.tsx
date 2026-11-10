import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { usePermission } from "@/hooks/use-permission";
import type { DeskActivityDetail } from "@/lib/graphql/desk-attention";
import { deskActivity } from "@/lib/queries/desk-activity";
import { useDeskMarginStore } from "@/stores/desk-margin-store";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import {
  formatUnixDateTimeShort,
  formatUnixTime,
  resolveUserTimezone,
} from "@trenova/shared/lib/date";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useEffect } from "react";
import { Link, useNavigate } from "react-router";
import { casePath } from "../attention/attention-record";
import { LifeBadge, outcomeLife } from "../attention/life-badge";
import { recordLine } from "./handled-lines";

function whyTitle(t: TranslateFn, detail: DeskActivityDetail): string {
  switch (detail.entry.outcome) {
    case "OnItsOwn":
      return t("Why it could do this on its own");
    case "Approved":
      return t("Why it ran");
    case "Previewed":
      return t("Why it was only previewed");
    default:
      return t("Why it didn't go through");
  }
}

function ruleLine(t: TranslateFn, detail: DeskActivityDetail): string {
  const agent = detail.entry.agent?.name ?? t("The agent");
  switch (detail.entry.outcome) {
    case "OnItsOwn":
      return t("AI control lets {0} {1} without asking.", agent, detail.tool.toLowerCase());
    case "Approved":
      return detail.entry.decidedBy
        ? t("{0} asked first, and {1} approved it.", agent, detail.entry.decidedBy.name)
        : t("{0} asked first, and a person approved it.", agent);
    case "Previewed":
      return t("{0} runs this in shadow: it is previewed and never made.", agent);
    default:
      return detail.entry.detail || t("The server refused it, so nothing changed.");
  }
}

/**
 * One line of Nova handled read in full: what the write was, why the agent
 * could make it (the rule it ran under, or who approved it), how often this
 * agent has made it before, and every record it changed. Every figure is the
 * server's record of the write; nothing here is written by a model.
 */
export function DeskHandled({ id }: { id: string }) {
  const t = useT();
  const navigate = useNavigate();
  const timezone = resolveUserTimezone(useAuthStore((state) => state.user?.timezone));
  const { allowed: canSeeControl } = usePermission(Resource.AgentDefinition, Operation.Read);
  const query = useQuery(deskActivity.entry(id));
  const detail = query.data ?? null;
  const setPageSubject = useDeskMarginStore((state) => state.setPageSubject);
  const first = detail?.entry.records.find((record) => record.readable) ?? null;
  useEffect(() => {
    setPageSubject(first ? { kind: first.kind, id: first.id, label: first.label } : null);
  }, [first, setPageSubject]);
  useEffect(() => () => setPageSubject(null), [setPageSubject]);

  if (query.isPending) {
    return (
      <div className="dk-w-case-scroll" aria-busy aria-label={t("Reading what the agent did")}>
        <div className="dk-w-det dk-w-led">
          <span className="dk-w-sk-line dk-w-sk-short ui-shimmer" />
          <span className="dk-w-sk-row ui-shimmer" />
          <span className="dk-w-sk-row ui-shimmer" />
        </div>
      </div>
    );
  }
  if (query.isError || detail === null) {
    return (
      <div className="dk-w-empty">
        <div>
          <b>{t("This line of the ledger could not be read")}</b>
          <Button size="sm" variant="outline" onClick={() => void query.refetch()}>
            {t("Try again")}
          </Button>
        </div>
      </div>
    );
  }

  const { entry } = detail;
  const records = recordLine(t, entry);

  return (
    <div className="dk-w-case-scroll">
      <div className="dk-w-det dk-w-led">
        <div className="dk-w-det-k">
          {entry.agent && <DeskAgentTile agent={entry.agent} size="xs" />}
          <span>{entry.agent?.name ?? t("An agent")}</span>
          <span>·</span>
          <span>{formatUnixDateTimeShort(entry.at, { timezone })}</span>
          <span className="dk-w-sp" />
          <LifeBadge
            life={outcomeLife(entry.outcome)}
            extra={
              entry.outcome === "Failed"
                ? undefined
                : t("· confirmed {0}", formatUnixTime(entry.at, { timezone }))
            }
          />
        </div>
        <h1 className="dk-w-det-h">{entry.title}</h1>
        {entry.detail !== "" && entry.outcome !== "Failed" && (
          <p className="dk-w-det-sum">{entry.detail}</p>
        )}

        <section className="dk-w-blk">
          <div className="dk-w-blk-h">{whyTitle(t, detail)}</div>
          <div className="dk-w-kvrows">
            <div>
              <span className="dk-w-kvrows-k">{t("The rule")}</span>
              <span>{ruleLine(t, detail)}</span>
            </div>
            <div>
              <span className="dk-w-kvrows-k">{t("Its record")}</span>
              <span>
                {t(
                  "{0, plural, =0 {The first time it has made this change in 30 days.} one {Made this change once in the last 30 days.} other {Made this change # times in the last 30 days.}}",
                  detail.ranBefore,
                )}
              </span>
            </div>
            <div>
              <span className="dk-w-kvrows-k">{t("Who can change this")}</span>
              <span>
                {t("Anyone with AI control access.")}{" "}
                {canSeeControl && (
                  <Link className="dk-w-alnk ui-focus-ring" to="/admin/agent-control">
                    {t("Open AI control")}
                  </Link>
                )}
              </span>
            </div>
          </div>
        </section>

        <section className="dk-w-blk">
          <div className="dk-w-blk-h">
            {t("What changed")}
            {records !== "" && <span className="dk-w-blk-src">{records}</span>}
          </div>
          <div className="dk-w-tbl">
            <table>
              <thead>
                <tr>
                  <th>{t("What it changed")}</th>
                  <th>{t("When")}</th>
                  <th>{t("How it ended")}</th>
                </tr>
              </thead>
              <tbody>
                {detail.changes.map((change) => (
                  <tr key={change.proposalId}>
                    <td>
                      {change.record?.readable ? (
                        <Link
                          className="dk-w-alnk dk-w-mono ui-focus-ring"
                          to={casePath(change.record.kind, change.record.id)}
                        >
                          {change.record.label}
                        </Link>
                      ) : change.record ? (
                        <span className="dk-w-mono">{t("A record you can't open")}</span>
                      ) : (
                        <span>{t("No record")}</span>
                      )}
                    </td>
                    <td className="dk-w-mono">{formatUnixTime(change.at, { timezone })}</td>
                    <td>
                      <LifeBadge life={outcomeLife(change.outcome)} />
                      {change.error !== "" && <span className="dk-w-tbl-err">{change.error}</span>}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>

        <div className="dk-w-blk dk-w-led-acts">
          {first && (
            <Button
              variant="outline"
              onClick={() =>
                useDeskMarginStore
                  .getState()
                  .openMargin({ kind: first.kind, id: first.id, label: first.label })
              }
            >
              <DeskIcon name="chat" size={13} />
              {t("Ask about this")}
            </Button>
          )}
          <Button variant="ghost" onClick={() => void navigate("/desk")}>
            {t("Back to Today")}
          </Button>
        </div>
      </div>
    </div>
  );
}
