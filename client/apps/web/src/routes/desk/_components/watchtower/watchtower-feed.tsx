import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  dismissWatchtowerItem,
  handOffWatchtowerItem,
  markWatchtowerSeen,
  type WatchtowerItem,
  type WatchtowerSeverity,
  type WatchtowerSourceKind,
} from "@/lib/graphql/watchtower";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { useAssistantStore } from "@/stores/assistant-store";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router";
import { toast } from "sonner";
import { WatchtowerItemRow } from "./item-row";
import { ALL_SEVERITIES, isUnseen, toggleFilter } from "./severity";

const nowInSeconds = () => Math.floor(Date.now() / 1000);

/**
 * The watchtower: everything the system noticed, newest first.
 *
 * It is a projection rather than a union of eleven queries — the sources stay
 * authoritative and an item resolves when its record does — so this reads one
 * list, pages it by time, and carries the reader's own cursor for the unseen
 * line. Filtering by kind or severity is a different question, not the same
 * one asked again, so each filter is part of the query key.
 *
 * Opening the feed marks it seen. That is deliberate: the cursor answers
 * "what is new since I last looked", and the act of looking is what moves it.
 */
export function WatchtowerFeed() {
  const t = useT();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [now] = useState(nowInSeconds);
  const [kinds, setKinds] = useState<WatchtowerSourceKind[]>([]);
  const [severities, setSeverities] = useState<WatchtowerSeverity[]>([]);

  const countsQuery = useQuery(queries.watchtower.counts());
  const filter = useMemo(() => ({ kinds, severities, unresolvedOnly: true }), [kinds, severities]);
  const feedQuery = useQuery(queries.watchtower.feed(filter));

  const refresh = useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: queries.watchtower._def }),
      queryClient.invalidateQueries({ queryKey: queries.attention._def }),
    ]);
  }, [queryClient]);

  // The cursor the feed draws its line against. Held here rather than read
  // from the page so that marking the feed seen stops drawing the line
  // without waiting for every row to come back.
  const [seenAt, setSeenAt] = useState<number | null>(null);
  const servedSeenAt = feedQuery.data?.seenAt ?? 0;
  const line = seenAt ?? servedSeenAt;

  useEffect(() => {
    if (feedQuery.data === undefined) {
      return;
    }
    let cancelled = false;
    void markWatchtowerSeen().then((counts) => {
      if (!cancelled) {
        queryClient.setQueryData(queries.watchtower.counts().queryKey, counts);
      }
    });

    return () => {
      cancelled = true;
    };
    // Once per visit: the cursor answers "since I last looked", and looking
    // twice at the same page is one look.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const dismissMutation = useApiMutation({
    mutationFn: (item: WatchtowerItem) => dismissWatchtowerItem(item.id),
    onSuccess: refresh,
    resourceName: "Watchtower item",
  });

  const handOffMutation = useApiMutation({
    mutationFn: (item: WatchtowerItem) => handOffWatchtowerItem(item.id),
    onSuccess: async (result) => {
      if (result.runId !== null) {
        toast.success(t("An agent is working on it"));
      } else if (result.subscribers.length > 0) {
        toast.success(t("Handed to {0}", result.subscribers.map((agent) => agent.name).join(", ")));
      } else if (result.candidates.length > 0) {
        toast.info(t("No agent subscribes to this yet"), {
          description: t(
            "{0} could take it — start one from AI Control.",
            result.candidates[0].name,
          ),
        });
      } else {
        toast.info(t("No agent can take this one yet"));
      }
      await refresh();
    },
    resourceName: "Watchtower item",
  });

  // Asking about an item opens a conversation on the record it stands for, so
  // the agent starts with the subject rather than with the question "which
  // shipment?". The agent is the one the person last talked to, on the same
  // rule the ask box uses.
  const agentsQuery = useQuery(queries.assistant.agents(true, true));
  const lastAgentId = useAssistantStore((state) => state.lastAgentId);
  const askAgent =
    agentsQuery.data?.find((agent) => agent.id === lastAgentId) ?? agentsQuery.data?.[0] ?? null;

  const askMutation = useApiMutation({
    mutationFn: (item: WatchtowerItem) => {
      if (askAgent === null) {
        throw new Error(t("No agents are available to ask"));
      }

      return apiService.assistantService.startThread(askAgent.id, {
        origin: "Watchtower",
        subjectType: item.subjectType ?? undefined,
        subjectId: item.subjectId ?? undefined,
      });
    },
    onSuccess: (thread) => navigate(`/desk/t/${thread.id}`),
    resourceName: "Conversation",
  });

  const items = feedQuery.data?.items ?? [];
  const counts = countsQuery.data;
  const busy = dismissMutation.isPending || handOffMutation.isPending || askMutation.isPending;
  const canAsk = askAgent !== null;

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="border-desk-hairline flex flex-wrap items-center gap-1.5 border-b px-4 py-3">
        {ALL_SEVERITIES.map((severity) => (
          <FilterChip
            key={severity}
            label={severityLabel(t, severity)}
            active={severities.includes(severity)}
            onClick={() => setSeverities((current) => toggleFilter(current, severity))}
          />
        ))}

        {counts && counts.byKind.length > 0 && (
          <span aria-hidden className="bg-desk-hairline mx-1 h-5 w-px" />
        )}

        {counts?.byKind.map((row) => (
          <FilterChip
            key={row.kind}
            label={row.label}
            count={row.count}
            active={kinds.includes(row.kind)}
            onClick={() => setKinds((current) => toggleFilter(current, row.kind))}
          />
        ))}
      </div>

      <ScrollArea className="min-h-0 flex-1">
        {feedQuery.isLoading ? (
          <div className="flex flex-col gap-2 p-4">
            <Skeleton className="h-14" />
            <Skeleton className="h-14" />
            <Skeleton className="h-14" />
          </div>
        ) : items.length === 0 ? (
          <EmptySheet
            className="my-10"
            title={t("Nothing is asking for you")}
            description={t(
              "Failed runs, service failures, expiring credentials, quarantined EDI and anything else worth a look will appear here as it happens.",
            )}
            sketch={
              <div className="flex flex-col gap-3">
                <GhostLine className="w-2/3" />
                <GhostLine className="w-1/2" />
                <GhostLine className="w-3/5" />
              </div>
            }
          />
        ) : (
          <ul className="flex flex-col">
            {items.map((item) => (
              <WatchtowerItemRow
                key={item.id}
                item={item}
                unseen={isUnseen(item.occurredAt, line)}
                now={now}
                busy={busy}
                onAsk={canAsk ? (value) => askMutation.mutate(value) : undefined}
                onHandOff={(value) => handOffMutation.mutate(value)}
                onDismiss={(value) => dismissMutation.mutate(value)}
              />
            ))}
          </ul>
        )}
      </ScrollArea>

      {feedQuery.data?.hasNextPage && (
        <div className="border-desk-hairline flex justify-center border-t p-2">
          <Button
            size="sm"
            variant="ghost"
            onClick={() => setSeenAt(line)}
            disabled={feedQuery.isFetching}
          >
            {t("There is more below")}
          </Button>
        </div>
      )}
    </div>
  );
}

function FilterChip({
  label,
  count,
  active,
  onClick,
}: {
  label: string;
  count?: number;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      aria-pressed={active}
      onClick={onClick}
      className={cn(
        "ui-focus-ring flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs transition-colors",
        active
          ? "bg-foreground text-background"
          : "text-muted-foreground hover:text-foreground ring-foreground/10 ring-1",
      )}
    >
      {label}
      {count !== undefined && (
        <Badge
          variant="neutral"
          className={cn("h-4 px-1 tabular-nums", active && "bg-background/20 text-background")}
        >
          {count}
        </Badge>
      )}
    </button>
  );
}

function severityLabel(t: (value: string) => string, severity: WatchtowerSeverity): string {
  switch (severity) {
    case "Critical":
      return t("Critical");
    case "Warning":
      return t("Warning");
    default:
      return t("Info");
  }
}
