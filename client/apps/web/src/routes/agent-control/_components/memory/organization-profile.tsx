import { useAgentProfile } from "@/hooks/use-agent-profile";
import {
  MAX_PROFILE_GUIDANCE_CHARS,
  agentProfileHistoryQueryKey,
  fetchAgentProfileHistory,
  type AgentProfileGuidance,
  type AgentProfileVersion,
} from "@/lib/graphql/agent-profile";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import type { AgentProfileTrigger } from "@trenova/graphql/generated/graphql";
import { useState, type FormEvent } from "react";
import { Ic } from "../kit/ic";

const TRIGGER_LABEL: Record<AgentProfileTrigger, string> = defineLabels({
  Scheduled: "Nightly",
  Requested: "Asked for",
  Guidance: "After guidance",
});

/**
 * What every agent is told about the organization's business before it answers: a
 * write-up of facts worked out from the records each night, and of the guidance people
 * gave. The write-up is never edited by hand; a line of guidance is how a person
 * corrects it, and the next version honours every line. Only facts and guidance are
 * shown as the source, so nobody mistakes a figure in the write-up for a live one.
 */
export function OrganizationProfile({
  canCreate,
  canUpdate,
}: {
  canCreate: boolean;
  canUpdate: boolean;
}) {
  const t = useT();
  const { profileQuery, addGuidance, retireGuidance, regenerate } = useAgentProfile("organization");
  const [draft, setDraft] = useState("");
  const [historyOpen, setHistoryOpen] = useState(false);

  const profile = profileQuery.data;
  if (!profile) {
    return null;
  }
  const current = profile.current;

  const submit = (event: FormEvent) => {
    event.preventDefault();
    const content = draft.trim();
    if (content === "") {
      return;
    }
    addGuidance.mutate(content, { onSuccess: () => setDraft("") });
  };

  return (
    <section className="sec" aria-label={t("Organization profile")}>
      <header className="sh2">
        <Ic n="brain" s={14} />
        <h3>{t("What agents know about the organization")}</h3>
        <span className="sp" />
        {current && (
          <span className="sh2-n">
            {t("Checked {0}", formatUnixDateTimeMedium(current.checkedAt))}
          </span>
        )}
        {canUpdate && (
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={regenerate.isPending}
            onClick={() => regenerate.mutate()}
          >
            <Ic n="refresh" s={12} />
            {t("Write it again")}
          </Button>
        )}
      </header>

      <div className="sgm">
        {current?.content ? (
          <p className="sgm-c whitespace-pre-line">{current.content}</p>
        ) : (
          <p className="sgm-c">
            {t(
              "No profile yet. One is written each night once there is work on record, from your shipments, fleet, customers, lanes, integrations and settings.",
            )}
          </p>
        )}
        {profile.latestFailure && (
          <p className="sgm-e t-d">
            {t(
              "The last attempt did not finish, so agents are still told the version above: {0}",
              profile.latestFailure.errorMessage ?? "",
            )}
          </p>
        )}
        {current && current.facts.length > 0 && (
          <details>
            <summary className="sgm-e cursor-pointer">{t("What it was written from")}</summary>
            <ul className="rf-l" aria-label={t("Facts from the records")}>
              {current.facts.map((fact) => (
                <li key={fact.key} className="sgm-e">
                  {fact.text}
                </li>
              ))}
            </ul>
          </details>
        )}
      </div>

      <header className="sh2">
        <h3>{t("Guidance")}</h3>
        <em className="mono">{profile.guidance.length}</em>
        <span className="sp" />
        <span className="sh2-n">
          {t("What the records can't show, such as a seasonal lane or a customer winding down")}
        </span>
      </header>
      {profile.guidance.map((line) => (
        <GuidanceRow
          key={line.id}
          line={line}
          canRetire={canUpdate}
          busy={retireGuidance.isPending}
          onRetire={() => retireGuidance.mutate(line.id)}
        />
      ))}
      {canCreate && (
        <form className="sgm" onSubmit={submit} aria-label={t("Add guidance")}>
          <Textarea
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
            maxLength={MAX_PROFILE_GUIDANCE_CHARS}
            minRows={2}
            placeholder={t("e.g. “The Laredo lane is seasonal; it runs from October to March”")}
            aria-label={t("Guidance")}
          />
          <div className="sgm-a">
            <span className="sgm-e">
              {t("{0} of {1}", draft.length, MAX_PROFILE_GUIDANCE_CHARS)}
            </span>
            <Button
              type="submit"
              variant="default"
              size="sm"
              disabled={draft.trim() === "" || addGuidance.isPending}
            >
              <Ic n="plus" s={12} />
              {t("Add guidance")}
            </Button>
          </div>
        </form>
      )}

      <details onToggle={(event) => setHistoryOpen(event.currentTarget.open)}>
        <summary className="sgm-e cursor-pointer">{t("Earlier versions")}</summary>
        {historyOpen && <ProfileHistory />}
      </details>
    </section>
  );
}

function GuidanceRow({
  line,
  canRetire,
  busy,
  onRetire,
}: {
  line: AgentProfileGuidance;
  canRetire: boolean;
  busy: boolean;
  onRetire: () => void;
}) {
  const t = useT();
  const who = line.createdByName ?? t("Someone");
  const when = formatUnixDateTimeMedium(line.createdAt);

  return (
    <div className="sgm">
      <p className="sgm-c">{line.content}</p>
      <p className="sgm-e">
        {line.source === "Agent"
          ? t("From {0} in a conversation, written down by an agent · {1}", who, when)
          : t("From {0} · {1}", who, when)}
      </p>
      {canRetire && (
        <div className="sgm-a">
          <Button type="button" variant="ghost" size="sm" disabled={busy} onClick={onRetire}>
            {t("Retire")}
          </Button>
        </div>
      )}
    </div>
  );
}

function ProfileHistory() {
  const t = useT();
  const historyQuery = useQuery({
    queryKey: agentProfileHistoryQueryKey,
    queryFn: ({ signal }) => fetchAgentProfileHistory({ signal }),
  });

  if (historyQuery.isPending) {
    return <p className="sgm-e">{t("Loading…")}</p>;
  }
  const versions = historyQuery.data ?? [];
  if (versions.length === 0) {
    return <p className="sgm-e">{t("No versions yet.")}</p>;
  }

  return (
    <ul className="rf-l" aria-label={t("Earlier versions")}>
      {versions.map((version) => (
        <VersionRow key={version.id} version={version} />
      ))}
    </ul>
  );
}

function VersionRow({ version }: { version: AgentProfileVersion }) {
  const t = useT();
  const status =
    version.status === "Current"
      ? t("Current")
      : version.status === "Failed"
        ? t("Did not finish")
        : t("Replaced");

  return (
    <li className="sgm">
      <p className="sgm-e">
        {formatUnixDateTimeMedium(version.createdAt)} · {t(TRIGGER_LABEL[version.trigger])} ·{" "}
        {status}
      </p>
      {version.status === "Failed" ? (
        <p className="sgm-e t-d">{version.errorMessage}</p>
      ) : (
        <p className="sgm-c whitespace-pre-line">{version.content}</p>
      )}
    </li>
  );
}
