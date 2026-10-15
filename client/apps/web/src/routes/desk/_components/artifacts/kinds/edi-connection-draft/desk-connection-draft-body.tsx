import { shortDigest } from "@/lib/digest";
import { assistant } from "@/lib/queries/assistant";
import { useDeskStore } from "@/stores/desk-store";
import type { AssistantArtifact, AssistantProposal } from "@/types/assistant";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { useMemo } from "react";
import {
  CardBar,
  CardBarHint,
  CardCheck,
  CardChecks,
  CardFacts,
  CardHead,
  CardSection,
  CardStatus,
  RecordCard,
} from "../../record-card/record-card";
import {
  connectionDraftFrom,
  draftOutline,
  partnerNameOf,
  proposeQuestion,
  resolutionQuestion,
  type ConnectionDraft,
  type DraftDocumentProfile,
  type DraftMapping,
  type DraftSample,
} from "./connection-draft";
import {
  ChangeMark,
  CredentialList,
  DecisionAsk,
  RequirementList,
  SetStamp,
  SourceTab,
  Sureness,
  ValueRows,
} from "./connection-draft-parts";

const APPLY_TOOL = "apply_edi_connection_draft";

function matchLine(draft: ConnectionDraft, t: TranslateFn): string {
  switch (draft.partner.match) {
    case "Existing":
      return t("Changes to a trading partner you already have");
    case "Ambiguous":
      return t("Matches more than one trading partner");
    case "New":
      return t("A new trading partner");
  }
}

function directionWords(draft: ConnectionDraft, t: TranslateFn): string {
  const { inbound, outbound } = draft.connection;
  if (inbound && outbound) return t("Both ways");
  if (inbound) return t("Inbound only");
  if (outbound) return t("Outbound only");
  return "";
}

/** The apply proposal made for this draft, matched by the digest the person reviewed. */
function proposalFor(
  proposals: readonly AssistantProposal[],
  draft: ConnectionDraft,
): AssistantProposal | undefined {
  return proposals
    .filter(
      (proposal) =>
        proposal.toolName === APPLY_TOOL &&
        draft.documentDigest !== "" &&
        proposal.arguments?.documentDigest === draft.documentDigest,
    )
    .at(-1);
}

function ProfileBlock({ profile }: { profile: DraftDocumentProfile }) {
  const t = useT();
  const template = profile.template;
  return (
    <div className="dk-ecd-prof">
      <div className="dk-ecd-prof-h">
        <SetStamp transactionSet={profile.transactionSet} direction={profile.direction} />
        <span className="dk-ecd-prof-n">
          <b>{profile.name}</b>
          <span>
            {template.name
              ? template.starter
                ? t("Starter template {0}", template.name)
                : t("Template {0}", template.name)
              : t("No template chosen")}
            {template.change === "Change" && template.current !== "" && (
              <s title={template.current}>{template.current}</s>
            )}
          </span>
        </span>
        <ChangeMark change={profile.change} />
      </div>
      <div className="dk-ecd-prof-m">
        <span>{t("{0, plural, one {# rule met} other {# rules met}}", template.met)}</span>
        {template.unmet.length > 0 && (
          <span className="dk-warn">
            {t("{0, plural, one {# rule unmet} other {# rules unmet}}", template.unmet.length)}
          </span>
        )}
        {template.unverified.length > 0 && (
          <span>
            {t(
              "{0, plural, one {# rule not checked} other {# rules not checked}}",
              template.unverified.length,
            )}
          </span>
        )}
        {template.considered.length > 1 && (
          <span>
            {t(
              "{0, plural, one {# template weighed} other {# templates weighed}}",
              template.considered.length,
            )}
          </span>
        )}
      </div>
      {template.note !== "" && <p className="dk-ecd-note">{template.note}</p>}
      {template.unmet.length > 0 && <RequirementList requirements={template.unmet} />}
      {profile.fields.length > 0 && <ValueRows values={profile.fields} />}
      {profile.controlNumbers.length > 0 && (
        <div className="dk-ecd-ctl">
          {profile.controlNumbers.map((control) => (
            <span key={control.kind} className={control.problem ? "dk-warn" : undefined}>
              <em>{control.kind}</em>
              <b>{control.start}</b>
              {control.current > 0 && control.change === "None" && (
                <i>{t("kept at {0}", control.current)}</i>
              )}
              {control.problem !== "" && <i>{control.problem}</i>}
              <SourceTab source={control.source} />
            </span>
          ))}
        </div>
      )}
    </div>
  );
}

function MappingRow({ mapping }: { mapping: DraftMapping }) {
  const t = useT();
  return (
    <div className="dk-ecd-map">
      <span className="dk-ecd-vl">{mapping.entityType}</span>
      <code title={mapping.sourceLabel || mapping.sourceId}>{mapping.sourceId}</code>
      <span className="dk-ecd-vv">
        {mapping.targetLabel || mapping.targetId ? (
          <b>{mapping.targetLabel || mapping.targetId}</b>
        ) : (
          <em>{t("Not matched")}</em>
        )}
        {mapping.reason !== "" && <span className="dk-ecd-vn">{mapping.reason}</span>}
      </span>
      <ChangeMark change={mapping.change} />
      <Sureness confidence={mapping.confidence} />
      <SourceTab source={mapping.source} />
    </div>
  );
}

function SampleCheck({ sample }: { sample: DraftSample }) {
  const t = useT();
  const mismatches = sample.checks.filter((check) => !check.matches);
  const state = !sample.parsed
    ? "bad"
    : mismatches.length > 0 ||
        sample.unmapped.length > 0 ||
        (sample.overview?.summary?.errorCount ?? 0) > 0
      ? "warn"
      : "ok";
  const title = [
    sample.transactionSet ? t("Sample {0}", sample.transactionSet) : t("Sample"),
    sample.page > 0 ? t("page {0}", sample.page) : "",
    sample.overview?.x12Version ?? "",
  ]
    .filter(Boolean)
    .join(" · ");
  const detail = !sample.parsed
    ? sample.error || t("It could not be read as X12.")
    : [
        mismatches.length === 0
          ? t("The envelope matches the drafted profile.")
          : mismatches
              .map((check) =>
                t("{0}: guide {1}, sample {2}", check.label, check.expected, check.found),
              )
              .join("; "),
        sample.unmapped.length > 0
          ? t(
              "{0, plural, one {# code has no mapping} other {# codes have no mapping}}",
              sample.unmapped.length,
            )
          : "",
      ]
        .filter(Boolean)
        .join(" ");
  return (
    <CardCheck state={state} title={title} detail={detail}>
      {sample.unmapped.length > 0 && (
        <span className="dk-ecd-codes">
          {sample.unmapped.map((code) => (
            <code key={`${code.entityType}-${code.sourceId}`} title={code.entityType}>
              {code.sourceId}
            </code>
          ))}
        </span>
      )}
    </CardCheck>
  );
}

/**
 * An EDI connection drafted from a trading partner's guide, for the person to
 * check before anything is saved. Every value carries the page it was read
 * from, and the guide's own words open from it. What the agent held back is at
 * the top, each reading with its page, and picking one has the agent draft it
 * again. The credentials are listed for the person to enter, never read from
 * the guide. Saving is a proposal like any other write.
 */
export function DeskConnectionDraftBody({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const draft = useMemo(() => connectionDraftFrom(artifact), [artifact]);
  const outline = useMemo(() => draftOutline(draft), [draft]);
  const askAgent = useDeskStore((state) => state.askAgent);
  const asked = useDeskStore((state) => state.asks[artifact.threadId] ?? null);
  const proposalsQuery = useQuery(assistant.proposals(artifact.threadId));
  const proposal = proposalFor(proposalsQuery.data?.results ?? [], draft);
  const ask = (question: string) => askAgent(artifact.threadId, question);
  const busy = asked !== null;
  const partnerName = partnerNameOf(draft);
  const credentials = draft.transport?.credentials ?? [];

  return (
    <RecordCard className="dk-ecd">
      <CardHead
        kicker={[
          t("EDI connection draft"),
          draft.fileName,
          t("{0, plural, one {# page} other {# pages}}", Math.max(1, draft.pageCount)),
        ]}
        title={partnerName}
        subtitle={matchLine(draft, t)}
        figure={
          <>
            <b>{draft.changes}</b>
            <span>{t("{0, plural, one {change} other {changes}}", draft.changes)}</span>
          </>
        }
        meta={
          <>
            {draft.ready ? (
              <CardStatus tone="success" mark="check">
                {t("Ready to approve")}
              </CardStatus>
            ) : (
              <CardStatus tone="warn">
                {t(
                  "{0, plural, one {# thing to settle} other {# things to settle}}",
                  Math.max(draft.blockers.length, outline.decisions),
                )}
              </CardStatus>
            )}
            <CardFacts>
              {[
                draft.connection.method,
                directionWords(draft, t),
                draft.connection.sets
                  .map((set) => `${set.transactionSet} ${set.direction === "Inbound" ? "↓" : "↑"}`)
                  .join(" "),
                <code key="digest" title={draft.documentDigest}>
                  {shortDigest(draft.documentDigest)}
                </code>,
              ]}
            </CardFacts>
          </>
        }
      />

      {draft.partner.match === "Ambiguous" && draft.partner.candidates.length > 0 && (
        <CardSection title={t("Which partner is this?")}>
          <CardChecks>
            {draft.partner.candidates.map((candidate) => (
              <CardCheck
                key={candidate.partnerId}
                state="warn"
                title={`${candidate.name} · ${candidate.code}`}
                detail={candidate.reasons.join("; ")}
              >
                <Button
                  variant="quiet"
                  size="sm"
                  disabled={busy}
                  onClick={() =>
                    ask(
                      t(
                        "The guide {0} is for the existing trading partner {1} ({2}, id {3}). Draft it again for that partner.",
                        draft.fileName,
                        candidate.name,
                        candidate.code,
                        candidate.partnerId,
                      ),
                    )
                  }
                >
                  {t("It is {0}", candidate.code)}
                </Button>
              </CardCheck>
            ))}
          </CardChecks>
        </CardSection>
      )}

      {draft.decisions.length > 0 && (
        <CardSection
          title={t("To decide")}
          aside={<span className="dk-ecd-aside">{t("Never chosen for you")}</span>}
        >
          {draft.decisions.map((decision) => (
            <DecisionAsk
              key={decision.field}
              decision={decision}
              disabled={busy}
              onChoose={(candidate) => ask(resolutionQuestion(draft, decision, candidate, t))}
            />
          ))}
        </CardSection>
      )}

      {draft.failures.length > 0 && (
        <CardSection title={t("Would not save")}>
          <CardChecks>
            {draft.failures.map((failure, index) => (
              <CardCheck
                // oxlint-disable-next-line react/no-array-index-key -- one record may fail several times on one field
                key={`${failure.record}-${failure.field}-${index}`}
                state="bad"
                title={failure.field ? `${failure.record} · ${failure.field}` : failure.record}
                detail={failure.message}
              />
            ))}
          </CardChecks>
        </CardSection>
      )}

      {credentials.length > 0 && (
        <CardSection
          title={t("Credentials you enter")}
          aside={
            <span className="dk-ecd-aside">
              {outline.secretsToEnter.length > 0
                ? t(
                    "{0, plural, one {# required} other {# required}}",
                    outline.secretsToEnter.length,
                  )
                : t("None required")}
            </span>
          }
        >
          <p className="dk-ecd-note">
            {t(
              "Nothing secret is taken from the guide. After you approve, enter these on the communication profile, then turn it on.",
            )}
          </p>
          <CredentialList credentials={credentials} />
        </CardSection>
      )}

      {draft.partner.fields.length > 0 && (
        <CardSection
          title={t("Trading partner")}
          aside={<ChangeMark change={draft.partner.change} />}
        >
          <ValueRows values={draft.partner.fields} />
        </CardSection>
      )}

      {draft.interchange.length > 0 && (
        <CardSection title={t("Interchange")}>
          <ValueRows values={draft.interchange} />
        </CardSection>
      )}

      {draft.transport && draft.transport.fields.length > 0 && (
        <CardSection
          title={draft.transport.name || t("Communication")}
          aside={<ChangeMark change={draft.transport.change} />}
        >
          <ValueRows values={draft.transport.fields} />
        </CardSection>
      )}

      {draft.documentProfiles.length > 0 && (
        <CardSection
          title={t("Document types")}
          aside={
            outline.unmet > 0 ? (
              <span className="dk-ecd-aside dk-warn">
                {t("{0, plural, one {# rule unmet} other {# rules unmet}}", outline.unmet)}
              </span>
            ) : undefined
          }
        >
          {draft.documentProfiles.map((profile) => (
            <ProfileBlock
              key={`${profile.transactionSet}-${profile.direction}`}
              profile={profile}
            />
          ))}
        </CardSection>
      )}

      {draft.mappings.length > 0 && (
        <CardSection
          title={t("Code mappings")}
          aside={
            draft.mappingCount > draft.mappings.length ? (
              <span className="dk-ecd-aside">
                {t("{0} of {1} shown", draft.mappings.length, draft.mappingCount)}
              </span>
            ) : undefined
          }
        >
          <div className="dk-ecd-vs">
            {draft.mappings.map((mapping) => (
              <MappingRow key={`${mapping.entityType}-${mapping.sourceId}`} mapping={mapping} />
            ))}
          </div>
        </CardSection>
      )}

      {draft.samples.length > 0 && (
        <CardSection title={t("The guide's samples")}>
          <CardChecks>
            {draft.samples.map((sample) => (
              <SampleCheck key={`${sample.page}-${sample.transactionSet}`} sample={sample} />
            ))}
          </CardChecks>
        </CardSection>
      )}

      {draft.gaps.length > 0 && (
        <CardSection title={t("Gaps in the guide")}>
          <CardChecks>
            {draft.gaps.map((gap, index) => (
              <CardCheck
                // oxlint-disable-next-line react/no-array-index-key -- two gaps may share a summary
                key={`${gap.summary}-${index}`}
                state={gap.kind === "Unsupported" ? "warn" : "info"}
                title={gap.kind === "Unsupported" ? t("Not supported") : t("Unclear")}
                detail={[gap.summary, gap.problem].filter(Boolean).join(" ")}
              >
                <SourceTab source={gap.source} />
              </CardCheck>
            ))}
          </CardChecks>
        </CardSection>
      )}

      {draft.kept.length > 0 && (
        <CardSection title={t("Left as they are")}>
          <ul className="dk-ecd-kept">
            {draft.kept.map((kept) => (
              <li key={`${kept.record}-${kept.label}`}>
                <span>{kept.record}</span>
                <b>{kept.label}</b>
              </li>
            ))}
          </ul>
        </CardSection>
      )}

      <CardBar>
        <CardBarHint>
          {draft.ready
            ? t("Nothing is saved until you approve the proposal.")
            : draft.blockers.join(" · ")}
        </CardBarHint>
        {proposal?.status === "Executed" ? (
          <span className="dk-ax-sent">{t("Saved")}</span>
        ) : proposal?.status === "Pending" ? (
          <span className="dk-ax-sent">{t("Waiting on your approval")}</span>
        ) : (
          <Button
            disabled={!draft.ready || busy || draft.documentDigest === ""}
            onClick={() => ask(proposeQuestion(draft, t))}
          >
            {t("Propose saving it")}
          </Button>
        )}
      </CardBar>
    </RecordCard>
  );
}
