import {
  AGENT_MEMORY_LIST_KEY,
  fetchTaintingAgentMemories,
  reviewAgentMemory,
  setAgentMemoryStatus,
  taintingAgentMemoriesQueryKey,
  type AgentMemorySuggestion,
} from "@/lib/graphql/agent-memories";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { toast } from "sonner";
import { memorySourceLabel } from "../activity/agent-badges";
import { invalidateAIControlCounts } from "../overview/use-ai-control-stats";

type TaintDecision = { memory: AgentMemorySuggestion; action: "review" | "retire" };

/**
 * Active memories written after a run had read outside content, that nobody has
 * reviewed. Each one taints every turn that reads it, so those turns hold their
 * money, customer-visible and outside-recipient writes for a person, and
 * nothing else says why. Reviewing one keeps it as the organization's own;
 * retiring it stops agents reading it.
 */
export function MemoryTaintNotice({ canDecide }: { canDecide: boolean }) {
  const t = useT();
  const queryClient = useQueryClient();
  const taintingQuery = useQuery({
    queryKey: taintingAgentMemoriesQueryKey,
    queryFn: ({ signal }) => fetchTaintingAgentMemories({ signal }),
  });
  const decide = useMutation({
    mutationFn: async ({ memory, action }: TaintDecision): Promise<void> => {
      if (action === "review") {
        await reviewAgentMemory(memory.id, memory.version);
        return;
      }
      await setAgentMemoryStatus(memory.id, "Retired");
    },
    onSuccess: async (_, { action }) => {
      toast.success(
        action === "review"
          ? t("Memory reviewed; turns that read it can write on their own again")
          : t("Memory retired"),
      );
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: [AGENT_MEMORY_LIST_KEY] }),
        invalidateAIControlCounts(queryClient),
      ]);
    },
  });

  const tainting = taintingQuery.data ?? [];
  if (tainting.length === 0) {
    return null;
  }

  return (
    <Alert size="sm" variant="warning" data-testid="memory-taint-notice">
      <AlertTitle>
        {t(
          "{0, plural, one {# memory holds every write of the turns that read it} other {# memories hold every write of the turns that read them}}",
          tainting.length,
        )}
      </AlertTitle>
      <AlertDescription className="flex flex-col gap-2">
        <span>
          {t(
            "An agent wrote these after reading text from outside the organization. Until a person reviews one, every turn that reads it waits for approval before any money, customer-visible or outside-recipient write.",
          )}
        </span>
        {decide.error ? (
          <span className="text-danger">
            {decide.error instanceof Error ? decide.error.message : t("That didn't go through")}
          </span>
        ) : null}
        <ul aria-label={t("Memories that taint every turn")} className="flex flex-col gap-1.5">
          {tainting.map((memory) => {
            const busy = decide.isPending && decide.variables?.memory.id === memory.id;

            return (
              <li
                key={memory.id}
                className="bg-card text-foreground ring-foreground/10 flex flex-wrap items-start gap-2 rounded-md px-2 py-1.5 ring-1"
              >
                <span className="flex min-w-0 flex-1 flex-col gap-0.5">
                  <span className="text-sm leading-snug">{memory.content}</span>
                  <span className="text-foreground-muted text-xs">
                    {t(
                      "{0} · {1, plural, one {read # time} other {read # times}}",
                      memorySourceLabel(memory.source, t),
                      formatNumber(memory.useCount),
                    )}
                  </span>
                </span>
                {canDecide ? (
                  <span className="flex shrink-0 gap-1">
                    <Button
                      size="xs"
                      variant="outline"
                      isLoading={busy && decide.variables?.action === "review"}
                      disabled={decide.isPending}
                      onClick={() => decide.mutate({ memory, action: "review" })}
                    >
                      {t("Reviewed, keep it")}
                    </Button>
                    <Button
                      size="xs"
                      variant="ghost"
                      isLoading={busy && decide.variables?.action === "retire"}
                      disabled={decide.isPending}
                      onClick={() => decide.mutate({ memory, action: "retire" })}
                    >
                      {t("Retire")}
                    </Button>
                  </span>
                ) : null}
              </li>
            );
          })}
        </ul>
      </AlertDescription>
    </Alert>
  );
}
