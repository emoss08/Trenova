import { useAgentProfile } from "@/hooks/use-agent-profile";
import { describeApiError } from "@/lib/api-error-message";
import {
  MAX_PROFILE_GUIDANCE_CHARS,
  type AgentProfileGuidance,
  type AgentProfileVersion,
} from "@/lib/graphql/agent-profile";
import { agentProfileHistoryQueryOptions } from "@/lib/queries/agent-profile";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { isOffered } from "@/components/desk-chat/memory/memory-format";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import type { AgentProfileTrigger } from "@trenova/graphql/generated/graphql";
import { useState, type FormEvent } from "react";
import { Ic } from "../kit/ic";
import { ScrollBox } from "../kit/layout";

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
 *
 * Anyone who can read the profile may offer a line; one who manages agent memory keeps
 * lines at once and decides on what others offered, which waits here as offered until
 * they do.
 */
export function OrganizationProfile({ canManage }: { canManage: boolean }) {
  const t = useT();
  const {
    profileQuery,
    addGuidance,
    retireGuidance,
    approveGuidance,
    dismissGuidance,
    regenerate,
  } = useAgentProfile("organization");
  const [draft, setDraft] = useState("");
  const [historyOpen, setHistoryOpen] = useState(false);

  const profile = profileQuery.data;
  if (profileQuery.isPending) {
    return <OrganizationProfileSkeleton />;
  }
  if (!profile) {
    return (
      <section className="sec" aria-label={t("Organization profile")}>
        <header className="sh2">
          <Ic n="brain" s={14} />
          <h3>{t("What agents know about the organization")}</h3>
        </header>
        <p className="sgm-e t-d" role="alert">
          {describeApiError(profileQuery.error, t("The profile could not be loaded"))}
        </p>
      </section>
    );
  }
  const current = profile.current;
  const active = profile.guidance.filter((line) => line.status === "Active");
  const offered = profile.guidance.filter(isOffered);

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
        {canManage && (
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
          <ScrollBox label={t("Profile write-up")} size="sm">
            <p className="sgm-c whitespace-pre-line">{current.content}</p>
          </ScrollBox>
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
            <ScrollBox label={t("What it was written from")} size="sm">
              <ul className="rf-l" aria-label={t("Facts from the records")}>
                {current.facts.map((fact) => (
                  <li key={fact.key} className="sgm-e">
                    {fact.text}
                  </li>
                ))}
              </ul>
            </ScrollBox>
          </details>
        )}
      </div>

      {offered.length > 0 && (
        <>
          <header className="sh2">
            <h3>{t("Offered guidance")}</h3>
            <em className="mono">{offered.length}</em>
            <span className="sp" />
            <span className="sh2-n">
              {canManage
                ? t("Waiting for you; the profile is written again with what you keep")
                : t("Waiting for an administrator; agents read none of it until then")}
            </span>
          </header>
          <ScrollBox label={t("Offered guidance")}>
            {offered.map((line) => (
              <OfferedGuidanceRow
                key={line.id}
                line={line}
                canDecide={canManage}
                busy={approveGuidance.isPending || dismissGuidance.isPending}
                onApprove={() => approveGuidance.mutate(line.id)}
                onDismiss={() => dismissGuidance.mutate(line.id)}
              />
            ))}
          </ScrollBox>
        </>
      )}

      <header className="sh2">
        <h3>{t("Guidance")}</h3>
        <em className="mono">{active.length}</em>
        <span className="sp" />
        <span className="sh2-n">
          {t("What the records can't show, such as a seasonal lane or a customer winding down")}
        </span>
      </header>
      {active.length > 0 && (
        <ScrollBox label={t("Guidance")}>
          {active.map((line) => (
            <GuidanceRow
              key={line.id}
              line={line}
              canRetire={canManage}
              busy={retireGuidance.isPending}
              onRetire={() => retireGuidance.mutate(line.id)}
            />
          ))}
        </ScrollBox>
      )}
      <form
        className="sgm"
        onSubmit={submit}
        aria-label={canManage ? t("Add guidance") : t("Offer guidance")}
      >
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
            {canManage
              ? t("{0} of {1}", draft.length, MAX_PROFILE_GUIDANCE_CHARS)
              : t(
                  "{0} of {1} · offered to an administrator, who decides whether the profile keeps it",
                  draft.length,
                  MAX_PROFILE_GUIDANCE_CHARS,
                )}
          </span>
          <Button
            type="submit"
            variant="default"
            size="sm"
            disabled={draft.trim() === "" || addGuidance.isPending}
          >
            <Ic n="plus" s={12} />
            {canManage ? t("Add guidance") : t("Offer guidance")}
          </Button>
        </div>
      </form>

      <details onToggle={(event) => setHistoryOpen(event.currentTarget.open)}>
        <summary className="sgm-e cursor-pointer">{t("Earlier versions")}</summary>
        {historyOpen && <ProfileHistory />}
      </details>
    </section>
  );
}

function OfferedGuidanceRow({
  line,
  canDecide,
  busy,
  onApprove,
  onDismiss,
}: {
  line: AgentProfileGuidance;
  canDecide: boolean;
  busy: boolean;
  onApprove: () => void;
  onDismiss: () => void;
}) {
  const t = useT();
  const who = line.createdByName ?? t("Someone");
  const when = formatUnixDateTimeMedium(line.createdAt);

  return (
    <div className="sgm">
      <p className="sgm-c">{line.content}</p>
      <p className="sgm-e">
        {line.source === "Agent"
          ? t("Offered by {0} in a conversation, written down by an agent · {1}", who, when)
          : t("Offered by {0} · {1}", who, when)}
      </p>
      {canDecide && (
        <div className="sgm-a">
          <Button type="button" variant="ghost" size="sm" disabled={busy} onClick={onDismiss}>
            {t("Dismiss")}
          </Button>
          <Button type="button" variant="default" size="sm" disabled={busy} onClick={onApprove}>
            <Ic n="check" s={12} w={2.2} />
            {t("Keep")}
          </Button>
        </div>
      )}
    </div>
  );
}

function OrganizationProfileSkeleton() {
  const t = useT();

  return (
    <section className="sec" aria-label={t("Organization profile")} aria-busy="true">
      <header className="sh2">
        <Ic n="brain" s={14} />
        <h3>{t("What agents know about the organization")}</h3>
        <span className="sp" />
        <Skeleton className="h-3 w-28" />
      </header>
      <div className="sgm" aria-hidden="true">
        <Skeleton className="h-3.5 w-full" />
        <Skeleton className="h-3.5 w-11/12" />
        <Skeleton className="h-3.5 w-4/5" />
        <Skeleton className="h-3.5 w-2/3" />
      </div>
      <header className="sh2">
        <h3>{t("Guidance")}</h3>
      </header>
      <div className="sgm" aria-hidden="true">
        <Skeleton className="h-3.5 w-3/5" />
        <Skeleton className="h-3 w-40" />
      </div>
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
  const historyQuery = useQuery(agentProfileHistoryQueryOptions());

  if (historyQuery.isPending) {
    return (
      <ul className="rf-l" aria-busy="true" aria-label={t("Earlier versions")}>
        <li className="sgm" aria-hidden="true">
          <Skeleton className="h-3 w-48" />
          <Skeleton className="h-3.5 w-full" />
          <Skeleton className="h-3.5 w-3/4" />
        </li>
      </ul>
    );
  }
  if (historyQuery.isError) {
    return (
      <p className="sgm-e t-d" role="alert">
        {describeApiError(historyQuery.error, t("Earlier versions could not be loaded"))}
      </p>
    );
  }
  const versions = historyQuery.data ?? [];
  if (versions.length === 0) {
    return <p className="sgm-e">{t("No versions yet.")}</p>;
  }

  return (
    <ScrollBox label={t("Earlier versions")} size="lg">
      <ul className="rf-l" aria-label={t("Earlier versions")}>
        {versions.map((version) => (
          <VersionRow key={version.id} version={version} />
        ))}
      </ul>
    </ScrollBox>
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
