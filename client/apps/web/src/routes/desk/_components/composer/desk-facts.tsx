import { handleMutationError } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { AssistantThread } from "@/types/assistant";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { useState } from "react";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { MAX_PINNED_FACTS, MAX_PINNED_FACT_LENGTH, pinFact, unpinFact } from "./desk-facts-state";

const NO_FACTS: readonly string[] = [];

/**
 * Saves a conversation's pinned facts, showing the change at once. The list
 * is written to the open thread before the server answers and put back if it
 * refuses.
 */
function usePinnedFacts(thread: AssistantThread) {
  const queryClient = useQueryClient();
  const threadKey = queries.assistant.thread(thread.id).queryKey;

  return useMutation({
    mutationFn: (facts: readonly string[]) =>
      apiService.assistantService.updateThread(thread.id, { pinnedFacts: facts }),
    onMutate: async (facts) => {
      await queryClient.cancelQueries({ queryKey: threadKey });
      const previous = queryClient.getQueryData<AssistantThread>(threadKey);
      queryClient.setQueryData<AssistantThread>(threadKey, (current) =>
        current ? { ...current, pinnedFacts: [...facts] } : current,
      );

      return { previous };
    },
    onError: (error, _facts, context) => {
      if (context?.previous) {
        queryClient.setQueryData(threadKey, context.previous);
      }
      handleMutationError({ error, resourceName: "Conversation" });
    },
    onSuccess: (saved) => {
      queryClient.setQueryData(threadKey, saved);
    },
  });
}

/**
 * What the agents keep in mind for the whole conversation, above the
 * composer: a chip per pinned fact, × to unpin, and a "+" that opens a short
 * field to pin another. The facts ride in every turn's instructions, so
 * neither a long conversation nor its compaction loses them.
 */
export function DeskFactsBar({ thread }: { thread: AssistantThread }) {
  const t = useT();
  const mutation = usePinnedFacts(thread);
  const facts = thread.pinnedFacts ?? NO_FACTS;
  const [adding, setAdding] = useState(false);
  const [draft, setDraft] = useState("");

  const save = (next: readonly string[]) => {
    if (next !== facts) {
      mutation.mutate(next);
    }
  };
  const add = () => {
    save(pinFact(facts, draft));
    setDraft("");
    setAdding(false);
  };
  const cancel = () => {
    setDraft("");
    setAdding(false);
  };

  return (
    <div
      className="dk-fx"
      title={t("Agents keep these in mind for the whole conversation, even after it's compacted")}
    >
      <span className="dk-fx-l">
        <DeskIcon name="pin" size={12} />
        {t("Keeping in mind")}
      </span>
      {facts.map((fact) => (
        <span key={fact} className="dk-fx-c">
          {fact}
          <Button
            variant="bare"
            size="bare"
            className="size-4 justify-center rounded-full text-dsk-faint hover:bg-dsk-hover hover:text-dsk-fg"
            title={t("Unpin")}
            aria-label={t("Unpin {0}", fact)}
            onClick={() => save(unpinFact(facts, fact))}
          >
            <DeskIcon name="x" size={10} stroke={2.2} />
          </Button>
        </span>
      ))}
      {adding ? (
        <input
          className="dk-fx-in"
          // oxlint-disable-next-line jsx-a11y/no-autofocus -- the field opens from a click on "+"
          autoFocus
          value={draft}
          maxLength={MAX_PINNED_FACT_LENGTH}
          aria-label={t("Pin a fact")}
          placeholder={t("e.g. Invoice date is Oct 3")}
          onChange={(event) => setDraft(event.target.value)}
          onBlur={add}
          onKeyDown={(event) => {
            if (event.key === "Enter") {
              event.preventDefault();
              add();
            }
            if (event.key === "Escape") {
              event.preventDefault();
              event.stopPropagation();
              cancel();
            }
          }}
        />
      ) : (
        facts.length < MAX_PINNED_FACTS && (
          <Button
            variant="bare"
            size="bare"
            className="h-5.5 gap-1 rounded-full border border-dashed border-dsk-b-strong px-1.75 text-dsk-subtle transition-colors duration-150 hover:border-dsk-fg hover:text-dsk-fg"
            aria-label={t("Pin a fact")}
            onClick={() => setAdding(true)}
          >
            <DeskIcon name="plus" size={11} stroke={2.2} />
            {facts.length === 0 ? t("Pin a fact") : null}
          </Button>
        )
      )}
    </div>
  );
}
