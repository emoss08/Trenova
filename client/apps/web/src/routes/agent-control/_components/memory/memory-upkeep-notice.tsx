import {
  AGENT_MEMORY_LIST_KEY,
  agentMemoryUpkeepQueryKey,
  confirmAgentMemory,
  fetchAgentMemoryUpkeep,
  setAgentMemoryStatus,
  type AgentMemoryUpkeep,
} from "@/lib/graphql/agent-memories";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { formatRelativeTime } from "@trenova/shared/i18n/format";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { ScrollBox } from "../kit/layout";
import { invalidateAIControlCounts } from "../overview/use-ai-control-stats";
import { MemoryKindMark } from "./memory-kind";

type UpkeepDecision = { item: AgentMemoryUpkeep; action: "confirm" | "retire" };

/** Why the memory is listed, with how long it has stood, in a person's words. */
export function upkeepReasonLabel(item: AgentMemoryUpkeep, nowSeconds: number, t: TranslateFn) {
  const since = item.memory.confirmedAt ?? item.memory.createdAt;
  const ago = formatRelativeTime(since - nowSeconds);

  if (item.reason === "Aging") {
    return item.memory.confirmedAt
      ? t("A fact with no end date, last confirmed {0}", ago)
      : t("A fact with no end date, recorded {0} and never confirmed", ago);
  }
  return item.memory.confirmedAt
    ? t("No agent has used it since it was confirmed {0}", ago)
    : t("No agent has used it since it was recorded {0}", ago);
}

/**
 * Shared memories a person should read again: facts with no end date that
 * nobody has confirmed in months, and memories no turn has used. A memory
 * outlives what it describes, and agents follow it either way, so the list
 * asks the one question that keeps it honest: does this still hold? Saying so
 * takes it off the list until it ages again; retiring it stops agents reading
 * it; changing it is done in the table below.
 */
export function MemoryUpkeepNotice({ canDecide }: { canDecide: boolean }) {
  const t = useT();
  const queryClient = useQueryClient();
  const upkeepQuery = useQuery({
    queryKey: agentMemoryUpkeepQueryKey,
    queryFn: ({ signal }) => fetchAgentMemoryUpkeep({ signal }),
  });
  const decide = useMutation({
    mutationFn: async ({ item, action }: UpkeepDecision): Promise<void> => {
      if (action === "confirm") {
        await confirmAgentMemory(item.memory.id, item.memory.version);
        return;
      }
      await setAgentMemoryStatus(item.memory.id, "Retired");
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: [AGENT_MEMORY_LIST_KEY] }),
        invalidateAIControlCounts(queryClient),
      ]);
    },
  });

  const items = upkeepQuery.data ?? [];
  if (items.length === 0) {
    return null;
  }
  const nowSeconds = Math.floor(upkeepQuery.dataUpdatedAt / 1000);

  return (
    <Alert size="sm" variant="info" data-testid="memory-upkeep-notice">
      <AlertTitle>
        {t(
          "{0, plural, one {# memory is worth another look} other {# memories are worth another look}}",
          items.length,
        )}
      </AlertTitle>
      <AlertDescription className="flex flex-col gap-2">
        <span>
          {t(
            "Agents follow a memory until someone retires it, whether or not it is still true. Confirm the ones that still hold, retire the ones that don't, and change the rest in the list below.",
          )}
        </span>
        {decide.error ? (
          <span className="text-danger">
            {decide.error instanceof Error ? decide.error.message : t("That didn't go through")}
          </span>
        ) : null}
        <ScrollBox label={t("Memories worth another look")} size="sm" surface="info">
          <ul aria-label={t("Memories worth another look")} className="flex flex-col gap-1.5 p-px">
            {items.map((item) => {
              const { memory } = item;
              const busy = decide.isPending && decide.variables?.item.memory.id === memory.id;

              return (
                <li
                  key={memory.id}
                  className="bg-card text-foreground ring-foreground/10 flex flex-wrap items-start gap-2 rounded-md px-2 py-1.5 ring-1"
                >
                  <span className="flex min-w-0 flex-1 flex-col gap-0.5">
                    <span className="flex items-start gap-1.5 text-sm leading-snug">
                      <MemoryKindMark kind={memory.kind} />
                      <span className="min-w-0">
                        {memory.subjectLabel
                          ? t("{0}: {1}", memory.subjectLabel, memory.content)
                          : memory.content}
                      </span>
                    </span>
                    <span className="text-foreground-muted text-xs">
                      {upkeepReasonLabel(item, nowSeconds, t)}
                    </span>
                  </span>
                  {canDecide ? (
                    <span className="flex shrink-0 gap-1">
                      <Button
                        size="xs"
                        isLoading={busy && decide.variables?.action === "confirm"}
                        disabled={decide.isPending}
                        onClick={() => decide.mutate({ item, action: "confirm" })}
                      >
                        {t("Still true")}
                      </Button>
                      <Button
                        size="xs"
                        variant="ghost"
                        isLoading={busy && decide.variables?.action === "retire"}
                        disabled={decide.isPending}
                        onClick={() => decide.mutate({ item, action: "retire" })}
                      >
                        {t("Retire")}
                      </Button>
                    </span>
                  ) : null}
                </li>
              );
            })}
          </ul>
        </ScrollBox>
      </AlertDescription>
    </Alert>
  );
}
