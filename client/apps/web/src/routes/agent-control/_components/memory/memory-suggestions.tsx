import {
  AGENT_MEMORY_LIST_KEY,
  AGENT_MEMORY_SUGGESTIONS_KEY,
  approveAgentMemorySuggestion,
  dismissAgentMemorySuggestion,
  fetchAgentMemorySuggestions,
  type AgentMemorySuggestion,
} from "@/lib/graphql/agent-memories";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { TextareaField } from "@/components/fields/textarea-field";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useQueryState } from "nuqs";
import { useForm, useWatch } from "react-hook-form";
import { toast } from "sonner";
import { MEMORY_SUGGESTION_PARAM, memorySuggestionParser } from "../../ai-control-tabs";
import { Ic } from "../kit/ic";
import { Modal } from "../kit/modal";
import { invalidateAIControlCounts } from "../overview/use-ai-control-stats";
import { MEMORY_CONTENT_LIMIT } from "./memory-form-schema";
import { Button } from "@trenova/shared/components/ui/button";

export const agentMemorySuggestionsQueryKey = [AGENT_MEMORY_SUGGESTIONS_KEY] as const;

/**
 * Memories drawn from what people said about an agent's answers, or learned by an
 * agent looking back over its work, waiting for an administrator. Nothing here is read
 * by an agent until it is approved: approving makes the text active as it stands or as
 * edited, dismissing keeps the same pattern from being suggested again for thirty days.
 * What people wrote is shown as quoted evidence, never as the memory itself.
 */
export function MemorySuggestions({ canDecide }: { canDecide: boolean }) {
  const t = useT();
  const suggestionsQuery = useQuery({
    queryKey: agentMemorySuggestionsQueryKey,
    queryFn: ({ signal }) => fetchAgentMemorySuggestions({ signal }),
  });
  const [editingId, setEditingId] = useQueryState(MEMORY_SUGGESTION_PARAM, memorySuggestionParser);
  const settle = useSettleSuggestion();

  const suggestions = suggestionsQuery.data ?? [];
  const editing = canDecide
    ? (suggestions.find((suggestion) => suggestion.id === editingId) ?? null)
    : null;
  if (suggestions.length === 0) {
    return null;
  }

  return (
    <section className="sec sg-s">
      <header className="sh2">
        <span className="dm" />
        <h3>{t("Nova suggests")}</h3>
        <em className="mono">{suggestions.length}</em>
        <span className="sp" />
        <span className="sh2-n">{t("Agents read none of these until you approve")}</span>
      </header>
      {suggestions.map((suggestion) => (
        <SuggestionCard
          key={suggestion.id}
          suggestion={suggestion}
          canDecide={canDecide}
          busy={settle.isPending}
          onDismiss={() => settle.mutate({ kind: "dismiss", suggestion })}
          onEdit={() => void setEditingId(suggestion.id)}
          onApprove={() =>
            settle.mutate({ kind: "approve", suggestion, content: suggestion.content })
          }
        />
      ))}
      {editing && (
        <ApproveEdited
          key={editing.id}
          suggestion={editing}
          busy={settle.isPending}
          onClose={() => void setEditingId(null)}
          onApprove={(content) =>
            settle.mutate(
              { kind: "approve", suggestion: editing, content },
              { onSuccess: () => void setEditingId(null) },
            )
          }
        />
      )}
    </section>
  );
}

type Settlement =
  | { kind: "approve"; suggestion: AgentMemorySuggestion; content: string }
  | { kind: "dismiss"; suggestion: AgentMemorySuggestion };

function useSettleSuggestion() {
  const t = useT();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (settlement: Settlement) =>
      settlement.kind === "approve"
        ? approveAgentMemorySuggestion(settlement.suggestion.id, {
            content: settlement.content,
            version: settlement.suggestion.version,
          })
        : dismissAgentMemorySuggestion(settlement.suggestion.id, settlement.suggestion.version),
    onSuccess: async (_memory, settlement) => {
      toast.success(
        settlement.kind === "approve"
          ? t("Memory approved; agents read it from their next run")
          : t("Suggestion dismissed"),
      );
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: agentMemorySuggestionsQueryKey }),
        queryClient.invalidateQueries({ queryKey: [AGENT_MEMORY_LIST_KEY] }),
        invalidateAIControlCounts(queryClient),
      ]);
    },
    onError: () => {
      toast.error(t("The suggestion could not be saved; it may have changed since it was read"));
    },
  });
}

type SuggestionCardProps = {
  suggestion: AgentMemorySuggestion;
  canDecide: boolean;
  busy: boolean;
  onDismiss: () => void;
  onEdit: () => void;
  onApprove: () => void;
};

function SuggestionCard({
  suggestion,
  canDecide,
  busy,
  onDismiss,
  onEdit,
  onApprove,
}: SuggestionCardProps) {
  const t = useT();
  const evidence = suggestion.evidence;
  const ratings = evidence?.ratingCount ?? evidence?.feedbackIds.length ?? 0;
  const people = evidence?.distinctUsers ?? 0;
  const quotes = evidence?.quotes ?? [];
  const learned = suggestion.source === "Reflection";

  return (
    <div className="sgm">
      <p className="sgm-c">{suggestion.content}</p>
      <p className="sgm-e">
        {learned
          ? t(
              "Learned by an agent looking back over its work · {0}",
              formatUnixDateTimeMedium(suggestion.createdAt),
            )
          : evidence
            ? t(
                "{0, plural, one {Drawn from # rating} other {Drawn from # ratings}} by {1, plural, one {# person} other {# people}} · last rated {2}",
                ratings,
                people,
                formatUnixDateTimeMedium(evidence.lastRatedAt),
              )
            : t(
                "{0, plural, one {Drawn from # rating} other {Drawn from # ratings}} by {1, plural, one {# person} other {# people}}",
                ratings,
                people,
              )}
        {suggestion.tainted && <> · {t("drawn from content written outside the organization")}</>}
      </p>
      {learned && evidence?.reason && <p className="sgm-e">{evidence.reason}</p>}
      {suggestion.supersedes && (
        <p className="sgm-e">{t("Approving it retires “{0}”", suggestion.supersedes.content)}</p>
      )}
      {quotes.length > 0 && (
        <ul className="qts" aria-label={learned ? t("What taught it") : t("What people wrote")}>
          {quotes.map((quote) => (
            <li key={quote}>
              <q>{quote}</q>
            </li>
          ))}
        </ul>
      )}
      {canDecide && (
        <div className="sgm-a">
          <Button type="button" variant="ghost" size="sm" disabled={busy} onClick={onDismiss}>
            {t("Dismiss")}
          </Button>
          <Button type="button" variant="outline" size="sm" disabled={busy} onClick={onEdit}>
            <Ic n="edit" s={12} />
            {t("Edit")}
          </Button>
          <Button type="button" variant="default" size="sm" disabled={busy} onClick={onApprove}>
            <Ic n="check" s={12} w={2.2} />
            {t("Approve")}
          </Button>
        </div>
      )}
    </div>
  );
}

type ApproveEditedProps = {
  suggestion: AgentMemorySuggestion;
  busy: boolean;
  onClose: () => void;
  onApprove: (content: string) => void;
};

type ApproveEditedValues = { content: string };

function ApproveEdited({ suggestion, busy, onClose, onApprove }: ApproveEditedProps) {
  const t = useT();
  const form = useForm<ApproveEditedValues>({ defaultValues: { content: suggestion.content } });
  const content = useWatch({ control: form.control, name: "content" });
  const trimmed = content.trim();

  return (
    <Modal
      open
      onClose={onClose}
      title={t("Approve memory")}
      description={t(
        "Edit it so it reads as a rule an agent should follow. Once approved, every agent that asks for memory reads it.",
      )}
      footer={
        <>
          <span className="cmp-n mono">
            {content.length}/{MEMORY_CONTENT_LIMIT}
          </span>
          <span className="sp" />
          <Button type="button" variant="ghost" size="sm" onClick={onClose}>
            {t("Cancel")}
          </Button>
          <Button
            type="button"
            variant="default"
            size="sm"
            disabled={busy || trimmed === ""}
            onClick={() => onApprove(trimmed)}
          >
            {t("Approve")}
          </Button>
        </>
      }
    >
      <TextareaField<ApproveEditedValues>
        control={form.control}
        name="content"
        rules={{ required: true }}
        label={t("Memory")}
        autoFocus
        maxLength={MEMORY_CONTENT_LIMIT}
      />
    </Modal>
  );
}
