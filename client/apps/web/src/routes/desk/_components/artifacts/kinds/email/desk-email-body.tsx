import { Button } from "@trenova/shared/components/ui/button";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { markProposalEdits } from "@/lib/proposal-cache";
import { apiService } from "@/services/api";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { assistant } from "@/lib/queries/assistant";
import type { AssistantArtifact } from "@/types/assistant";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { useMemo, useState } from "react";
import {
  emailDraftEdits,
  emailDraftFrom,
  savedEmailWording,
  type EmailWording,
} from "../../artifact-payloads";
import { ArtIcon } from "../../desk-art-kinds";

/**
 * A message waiting to go: who it goes to, what it says and why it is worded
 * so. The subject and body can be changed here. "Send for approval" does not
 * decide anything: it saves the wording on the proposal behind the draft, so
 * it survives a reload, and the decision card on the composer approves the
 * proposal with it.
 */
export function DeskEmailBody({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const queryClient = useQueryClient();
  const draft = useMemo(() => emailDraftFrom(artifact), [artifact]);
  const proposalsQuery = useQuery({
    ...assistant.proposals(artifact.threadId),
    enabled: artifact.proposalId !== "",
  });
  const proposal = proposalsQuery.data?.results.find(
    (candidate) => candidate.id === artifact.proposalId,
  );
  const { copy, isCopied: copied } = useCopyToClipboard();
  const subjectKey = typeof artifact.payload.subject === "string" ? "subject" : "";
  const bodyKey = typeof artifact.payload.body === "string" ? "body" : "";
  const state =
    artifact.status === "Sent" || proposal?.status === "Executed"
      ? "sent"
      : proposal?.status === "Pending"
        ? "waiting"
        : proposal
          ? "decided"
          : "unknown";
  const editable = state === "waiting";

  // What the person saved is read from the proposal, which may arrive after
  // the draft; what they are typing now sits over it until it is saved.
  const saved = state === "waiting" ? proposal?.pendingModifications : null;
  const savedWording = savedEmailWording(draft, saved);
  const [typed, setTyped] = useState<Partial<EmailWording>>({});
  const subject = typed.subject ?? savedWording.subject;
  const body = typed.body ?? savedWording.body;
  const dirty = typed.subject !== undefined || typed.body !== undefined;
  // Sending the agent's wording unchanged saves nothing, so the hand-over is
  // remembered here until the page is left; a change is remembered by the server.
  const [handedOver, setHandedOver] = useState(false);
  const sentForApproval =
    !dirty && (handedOver || (saved != null && Object.keys(saved).length > 0));

  const save = useApiMutation({
    mutationFn: (modifications: Record<string, unknown>) =>
      apiService.assistantService.saveProposalEdits(
        artifact.threadId,
        artifact.proposalId,
        modifications,
      ),
    onSuccess: (edits) => {
      markProposalEdits(queryClient, edits.proposalId, edits.pendingModifications ?? null);
      setTyped({});
      setHandedOver(true);
      void queryClient.invalidateQueries({
        queryKey: assistant.proposals(artifact.threadId).queryKey,
      });
    },
    resourceName: "Draft",
  });
  const sendForApproval = () =>
    save.mutate(
      emailDraftEdits(
        draft,
        { subject, body },
        { subject: subjectKey !== "", body: bodyKey !== "" },
      ),
    );

  return (
    <div className="dk-ax-mail">
      {draft.to.length > 0 && (
        <div className="dk-ax-mrow">
          <span>{t("To")}</span>
          <div className="dk-ax-to">
            {draft.to.map((to) => (
              <span key={to}>{to}</span>
            ))}
          </div>
        </div>
      )}
      <div className="dk-ax-mrow">
        <span>{t("Subject")}</span>
        <input
          value={subject}
          readOnly={!editable || subjectKey === ""}
          onChange={(event) => {
            const value = event.target.value;
            setTyped((current) => ({ ...current, subject: value }));
          }}
          aria-label={t("Subject")}
        />
      </div>
      <textarea
        value={
          bodyKey === ""
            ? t("The message is written from the organization's template when it is sent.")
            : body
        }
        readOnly={!editable || bodyKey === ""}
        onChange={(event) => {
          const value = event.target.value;
          setTyped((current) => ({ ...current, body: value }));
        }}
        spellCheck={false}
        aria-label={t("Message")}
      />
      {draft.rationale !== "" && (
        <div className="dk-ax-why">
          <b>{t("Why this wording")}</b>
          {draft.rationale}
        </div>
      )}
      <div className="dk-ax-acts">
        <Button variant="quiet" onClick={() => void copy(`${subject}\n\n${body}`)}>
          <ArtIcon name={copied ? "check" : "copy"} size={13} />
          {copied ? t("Copied") : t("Copy")}
        </Button>
        <span className="flex-1" />
        {state === "sent" ? (
          <span className="dk-ax-sent">
            <ArtIcon name="check" size={13} stroke={2.4} />
            {t("Sent")}
          </span>
        ) : (state === "waiting" && sentForApproval) ||
          (state === "decided" && proposal?.status !== "Rejected") ? (
          <span className="dk-ax-sent">
            <ArtIcon name="check" size={13} stroke={2.4} />
            {t("Sent for approval")}
          </span>
        ) : state === "waiting" ? (
          <Button disabled={artifact.proposalId === "" || save.isPending} onClick={sendForApproval}>
            {t("Send for approval")}
          </Button>
        ) : state === "decided" ? (
          <span className="dk-ax-sent dk-wait">{t("Set aside")}</span>
        ) : null}
      </div>
    </div>
  );
}
