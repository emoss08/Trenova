import { SectionPanel } from "@/components/section-panel";
import {
  AGENT_MEMORY_LIST_KEY,
  AGENT_MEMORY_SUGGESTIONS_KEY,
  approveAgentMemorySuggestion,
  dismissAgentMemorySuggestion,
  fetchAgentMemorySuggestions,
  type AgentMemorySuggestion,
} from "@/lib/graphql/agent-memories";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useId, useState } from "react";
import { toast } from "sonner";
import { aiControlStatsQueryKey } from "../overview/use-ai-control-stats";
import { MEMORY_CONTENT_LIMIT } from "./memory-form-schema";

export const agentMemorySuggestionsQueryKey = [AGENT_MEMORY_SUGGESTIONS_KEY] as const;

/**
 * Memories drawn from what people said about an agent's answers, waiting for
 * an administrator. Nothing here is read by an agent until it is approved:
 * approving makes the edited text active, dismissing keeps the same pattern
 * from being suggested again for thirty days. What people wrote is shown as
 * quoted evidence, never as the memory itself.
 */
export function MemorySuggestions({ canDecide }: { canDecide: boolean }) {
  const t = useT();
  const suggestionsQuery = useQuery({
    queryKey: agentMemorySuggestionsQueryKey,
    queryFn: ({ signal }) => fetchAgentMemorySuggestions({ signal }),
  });
  const [approving, setApproving] = useState<AgentMemorySuggestion | null>(null);
  const settle = useSettleSuggestion();

  const suggestions = suggestionsQuery.data ?? [];
  if (suggestions.length === 0) {
    return null;
  }

  return (
    <SectionPanel
      title={t("Suggested from feedback")}
      icon={<AssistMark />}
      count={suggestions.length}
      help={t(
        "Drawn from ratings people gave an agent's answers. Agents read none of these until an administrator approves them.",
      )}
    >
      <ul className="divide-border-subtle flex flex-col divide-y">
        {suggestions.map((suggestion) => (
          <SuggestionRow
            key={suggestion.id}
            suggestion={suggestion}
            canDecide={canDecide}
            busy={settle.isPending}
            onApprove={() => setApproving(suggestion)}
            onDismiss={() => settle.mutate({ kind: "dismiss", suggestion })}
          />
        ))}
      </ul>

      <ApproveSuggestionDialog
        suggestion={approving}
        busy={settle.isPending}
        onClose={() => setApproving(null)}
        onApprove={(content) => {
          if (approving === null) {
            return;
          }
          settle.mutate(
            { kind: "approve", suggestion: approving, content },
            { onSuccess: () => setApproving(null) },
          );
        }}
      />
    </SectionPanel>
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
        queryClient.invalidateQueries({ queryKey: aiControlStatsQueryKey }),
      ]);
    },
    onError: () => {
      toast.error(t("The suggestion could not be saved; it may have changed since it was read"));
    },
  });
}

function SuggestionRow({
  suggestion,
  canDecide,
  busy,
  onApprove,
  onDismiss,
}: {
  suggestion: AgentMemorySuggestion;
  canDecide: boolean;
  busy: boolean;
  onApprove: () => void;
  onDismiss: () => void;
}) {
  const t = useT();
  const evidence = suggestion.evidence;
  const ratings = evidence?.ratingCount ?? evidence?.feedbackIds.length ?? 0;
  const people = evidence?.distinctUsers ?? 0;
  const quotes = evidence?.quotes ?? [];

  return (
    <li className="flex flex-col gap-2 px-3 py-3">
      <p className="text-sm leading-relaxed">{suggestion.content}</p>

      <p className="text-foreground-muted text-xs">
        {t("{0, plural, one {Drawn from # rating} other {Drawn from # ratings}}", ratings)} ·{" "}
        {t("{0, plural, one {# person} other {# people}}", people)}
        {evidence ? (
          <> · {t("last rated {0}", formatUnixDateTimeMedium(evidence.lastRatedAt))}</>
        ) : null}
      </p>

      {quotes.length > 0 && (
        <ul aria-label={t("What people wrote")} className="flex flex-col gap-1">
          {quotes.map((quote) => (
            <li
              key={quote}
              className="border-border-subtle text-foreground-muted border-l-2 pl-2 text-xs italic"
            >
              <q>{quote}</q>
            </li>
          ))}
        </ul>
      )}

      {canDecide && (
        <div className="flex items-center justify-end gap-2">
          <Button variant="ghost" size="sm" onClick={onDismiss} disabled={busy}>
            {t("Dismiss")}
          </Button>
          <Button size="sm" onClick={onApprove} disabled={busy}>
            {t("Review and approve")}
          </Button>
        </div>
      )}
    </li>
  );
}

function ApproveSuggestionDialog({
  suggestion,
  busy,
  onClose,
  onApprove,
}: {
  suggestion: AgentMemorySuggestion | null;
  busy: boolean;
  onClose: () => void;
  onApprove: (content: string) => void;
}) {
  return (
    <Dialog
      open={suggestion !== null}
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
    >
      <DialogContent size="md">
        {suggestion !== null && (
          <ApproveForm
            key={suggestion.id}
            suggestion={suggestion}
            busy={busy}
            onCancel={onClose}
            onApprove={onApprove}
          />
        )}
      </DialogContent>
    </Dialog>
  );
}

function ApproveForm({
  suggestion,
  busy,
  onCancel,
  onApprove,
}: {
  suggestion: AgentMemorySuggestion;
  busy: boolean;
  onCancel: () => void;
  onApprove: (content: string) => void;
}) {
  const t = useT();
  const contentId = useId();
  const [content, setContent] = useState(suggestion.content);
  const trimmed = content.trim();

  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(event) => {
        event.preventDefault();
        if (trimmed !== "") {
          onApprove(trimmed);
        }
      }}
    >
      <DialogHeader>
        <DialogTitle>{t("Approve memory")}</DialogTitle>
        <DialogDescription>
          {t(
            "Edit the memory so it reads as a rule an agent should follow. Once approved, every agent that asks for memory reads it.",
          )}
        </DialogDescription>
      </DialogHeader>

      <div className="flex flex-col gap-1">
        <label htmlFor={contentId} className="text-xs font-medium">
          {t("Memory")}
        </label>
        <Textarea
          id={contentId}
          value={content}
          onChange={(event) => setContent(event.target.value.slice(0, MEMORY_CONTENT_LIMIT))}
          maxLength={MEMORY_CONTENT_LIMIT}
          minRows={4}
          maxRows={12}
        />
        <span className="text-foreground-subtle self-end text-2xs tabular-nums">
          {content.length}/{MEMORY_CONTENT_LIMIT}
        </span>
      </div>

      <DialogFooter>
        <Button type="button" variant="outline" onClick={onCancel} disabled={busy}>
          {t("Cancel")}
        </Button>
        <Button type="submit" disabled={busy || trimmed === ""} isLoading={busy}>
          {t("Approve")}
        </Button>
      </DialogFooter>
    </form>
  );
}
