import { useT } from "@trenova/shared/i18n/use-t";
import { usePermission } from "@/hooks/use-permission";
import {
  DOT_RANDOM_DRAW_KEY,
  DOT_RANDOM_DRAWS_KEY,
  fetchDotRandomDraw,
  updateDotRandomDrawEntry,
} from "@/lib/graphql/worker-drug-alcohol";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@trenova/shared/components/ui/sheet";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { entryTally } from "@/lib/random-testing";
import { randomEntryStatusLabel } from "@trenova/shared/lib/drug-alcohol";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";
import { toast } from "sonner";

export type DrawDetailSheetProps = {
  drawId: string | null;
  onOpenChange: (open: boolean) => void;
};

export function DrawDetailSheet({ drawId, onOpenChange }: DrawDetailSheetProps) {
  const t = useT();

  const queryClient = useQueryClient();
  const { allowed: canManage } = usePermission(Resource.DOTRandomPool, Operation.Manage);

  const drawQuery = useQuery({
    queryKey: [DOT_RANDOM_DRAW_KEY, drawId],
    queryFn: ({ signal }) => fetchDotRandomDraw(drawId as string, { signal }),
    enabled: Boolean(drawId),
  });

  const entryMutation = useMutation({
    mutationFn: ({
      entryId,
      status,
      excuseReason,
    }: {
      entryId: string;
      status: "Notified" | "Excused" | "Missed";
      excuseReason?: string;
    }) => updateDotRandomDrawEntry({ entryId, status, excuseReason }),
    onSuccess: () => {
      void Promise.all([
        queryClient.invalidateQueries({ queryKey: [DOT_RANDOM_DRAW_KEY, drawId] }),
        queryClient.invalidateQueries({ queryKey: [DOT_RANDOM_DRAWS_KEY] }),
      ]);
    },
    onError: (error: Error) =>
      toast.error(t("Could not update the selection"), { description: error.message }),
  });

  const draw = drawQuery.data;
  const tallies = useMemo(() => entryTally(draw?.entries ?? []), [draw]);

  return (
    <Sheet open={Boolean(drawId)} onOpenChange={onOpenChange}>
      <SheetContent className="w-full sm:max-w-xl">
        <SheetHeader>
          <SheetTitle>{draw ? `Round ${draw.periodKey}` : "Round"}</SheetTitle>
          <SheetDescription>
            {draw
              ? `${draw.pool?.name ?? "Pool"} · ${draw.poolSize} drivers in the pool · drawn ${formatUnixDate(draw.drawnAt)}`
              : null}
          </SheetDescription>
        </SheetHeader>

        {drawQuery.isLoading || !draw ? (
          <div className="flex flex-col gap-2 p-4">
            {[0, 1, 2].map((row) => (
              <Skeleton key={row} className="h-10 w-full" />
            ))}
          </div>
        ) : (
          <div className="flex flex-col gap-4 overflow-y-auto p-4">
            <section className="rounded-md border p-3 text-xs">
              <h3 className="cc-label text-foreground mb-2">{t("How this round was drawn")}</h3>
              <dl className="grid grid-cols-2 gap-2">
                <div>
                  <dt className="text-muted-foreground text-[11px]">{t("Method")}</dt>
                  <dd className="font-mono text-[11px]">{draw.method}</dd>
                </div>
                <div>
                  <dt className="text-muted-foreground text-[11px]">{t("Period")}</dt>
                  <dd className="tabular-nums">
                    {formatUnixDate(draw.periodStart)} – {formatUnixDate(draw.periodEnd)}
                  </dd>
                </div>
                <div className="col-span-2">
                  <dt className="text-muted-foreground text-[11px]">{t("Seed")}</dt>
                  <dd className="font-mono text-[11px] break-all">{draw.seed}</dd>
                </div>
              </dl>
              <p className="text-muted-foreground mt-2 text-[11px]">
                {t("The same seed over the same roster reproduces exactly these names, in this order.")}
              </p>
            </section>

            <section aria-label={t("Collections")} className="grid grid-cols-2 gap-2">
              {tallies.map((tally) => (
                <div key={tally.substance} className="rounded-md border p-3 text-xs">
                  <div className="flex items-center justify-between gap-2">
                    <span className="cc-label text-foreground">{tally.substance}</span>
                    <span className="font-mono tabular-nums">
                      {tally.collected}
                      <span className="text-muted-foreground">/{tally.total} collected</span>
                    </span>
                  </div>
                  <p className="text-muted-foreground mt-1 tabular-nums">
                    {tally.total === 0
                      ? "Nobody selected"
                      : [
                          tally.outstanding > 0 ? `${tally.outstanding} to collect` : null,
                          tally.notified > 0 ? `${tally.notified} notified` : null,
                          tally.excused > 0 ? `${tally.excused} excused` : null,
                          tally.missed > 0 ? `${tally.missed} missed` : null,
                        ]
                          .filter(Boolean)
                          .join(" · ") || "All collected"}
                  </p>
                </div>
              ))}
            </section>

            <section>
              <h3 className="cc-label text-foreground mb-2">{t("Selected ({0})", draw.entries.length)}</h3>
              <ul className="flex flex-col gap-1.5">
                {draw.entries.map((entry) => (
                  <li
                    key={entry.id}
                    className="flex flex-wrap items-center justify-between gap-2 rounded-md border px-3 py-2 text-xs"
                  >
                    <span className="flex flex-wrap items-center gap-2">
                      <span className="text-muted-foreground tabular-nums">#{entry.rank}</span>
                      <span className="font-medium">
                        {entry.worker
                          ? `${entry.worker.firstName} ${entry.worker.lastName}`
                          : entry.workerId}
                      </span>
                      <Badge variant="secondary">
                        {entry.substance === "Alcohol" ? "Alcohol" : "Drug"}
                      </Badge>
                      <Badge
                        variant={
                          entry.status === "Completed"
                            ? "active"
                            : entry.status === "Missed"
                              ? "inactive"
                              : "warning"
                        }
                      >
                        {randomEntryStatusLabel(entry.status)}
                      </Badge>
                    </span>
                    {canManage && (entry.status === "Selected" || entry.status === "Notified") ? (
                      <span className="flex gap-1">
                        {entry.status === "Selected" ? (
                          <Button
                            size="xs"
                            variant="outline"
                            onClick={() =>
                              entryMutation.mutate({ entryId: entry.id, status: "Notified" })
                            }
                          >
                            {t("Mark notified")}
                          </Button>
                        ) : null}
                        <Button
                          size="xs"
                          variant="ghost"
                          onClick={() =>
                            entryMutation.mutate({
                              entryId: entry.id,
                              status: "Excused",
                              excuseReason: "Excused from the random testing console",
                            })
                          }
                        >
                          {t("Excuse")}
                        </Button>
                        <Button
                          size="xs"
                          variant="ghost"
                          onClick={() =>
                            entryMutation.mutate({ entryId: entry.id, status: "Missed" })
                          }
                        >
                          {t("Missed")}
                        </Button>
                      </span>
                    ) : null}
                  </li>
                ))}
              </ul>
              <p className="text-muted-foreground mt-2 text-[11px]">
                {t("A selection is marked collected by recording its test on the driver's Testing tab, not from here — that keeps the entry and the test from ever disagreeing.")}
              </p>
            </section>
          </div>
        )}
      </SheetContent>
    </Sheet>
  );
}
