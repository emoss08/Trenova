import { usePermission } from "@/hooks/use-permission";
import {
  cancelDotRandomDraw,
  DOT_RANDOM_DRAWS_KEY,
  DOT_RANDOM_POOLS_KEY,
  fetchDotRandomDraws,
  fetchDotRandomPools,
  finalizeDotRandomDraw,
  runDotRandomDraw,
  type RandomPoolRow,
} from "@/lib/graphql/worker-drug-alcohol";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDate } from "@trenova/shared/lib/date";
import {
  DOT_MINIMUM_ALCOHOL_RATE,
  DOT_MINIMUM_DRUG_RATE,
  randomPeriodLabel,
} from "@trenova/shared/lib/drug-alcohol";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useCallback, useState } from "react";
import { toast } from "sonner";
import { DrawDetailSheet } from "./draw-detail-sheet";
import { RandomPoolDialog } from "./random-pool-dialog";

export default function RandomTestingConsole() {
  const queryClient = useQueryClient();
  const { allowed: canEdit } = usePermission(Resource.DOTRandomPool, Operation.Update);
  const { allowed: canCreate } = usePermission(Resource.DOTRandomPool, Operation.Create);
  const { allowed: canDraw } = usePermission(Resource.DOTRandomPool, Operation.Manage);

  const [poolDialog, setPoolDialog] = useState<{ pool: RandomPoolRow | null } | null>(null);
  const [openDrawId, setOpenDrawId] = useState<string | null>(null);

  const poolsQuery = useQuery({
    queryKey: [DOT_RANDOM_POOLS_KEY],
    queryFn: ({ signal }) => fetchDotRandomPools({ signal }),
  });
  const drawsQuery = useQuery({
    queryKey: [DOT_RANDOM_DRAWS_KEY],
    queryFn: ({ signal }) => fetchDotRandomDraws(undefined, { signal }),
  });

  const invalidate = useCallback(
    () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: [DOT_RANDOM_POOLS_KEY] }),
        queryClient.invalidateQueries({ queryKey: [DOT_RANDOM_DRAWS_KEY] }),
      ]),
    [queryClient],
  );

  const drawMutation = useMutation({
    mutationFn: (poolId: string) => runDotRandomDraw({ poolId }),
    onSuccess: (draw) => {
      const short =
        draw.drugSelected < draw.drugTarget || draw.alcoholSelected < draw.alcoholTarget;
      toast.success(`Drew ${draw.periodKey}`, {
        description: short
          ? `The pool is smaller than the target: ${draw.drugSelected} of ${draw.drugTarget} drug and ${draw.alcoholSelected} of ${draw.alcoholTarget} alcohol.`
          : `${draw.drugSelected} for drug testing and ${draw.alcoholSelected} for alcohol, from ${draw.poolSize} drivers.`,
      });
      void invalidate();
      setOpenDrawId(draw.id);
    },
    onError: (error: Error) =>
      toast.error("Could not run the draw", { description: error.message }),
  });

  const finalizeMutation = useMutation({
    mutationFn: (id: string) => finalizeDotRandomDraw(id),
    onSuccess: () => {
      toast.success("Round finalised", {
        description: "The names are now the record. Correcting one means voiding the round.",
      });
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error("Could not finalise the round", { description: error.message }),
  });

  const cancelMutation = useMutation({
    mutationFn: (id: string) => cancelDotRandomDraw(id, "Voided from the random testing console"),
    onSuccess: () => {
      toast.success("Round voided", { description: "The period can be drawn again." });
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error("Could not void the round", { description: error.message }),
  });

  if (poolsQuery.isLoading || drawsQuery.isLoading) {
    return (
      <div className="flex flex-col gap-3">
        <Skeleton className="h-40 w-full" />
        <Skeleton className="h-40 w-full" />
      </div>
    );
  }

  const pools = poolsQuery.data ?? [];
  const draws = drawsQuery.data ?? [];

  return (
    <div className="flex flex-col gap-6">
      <section>
        <div className="mb-2 flex items-center justify-between">
          <h2 className="cc-label text-foreground">Pools</h2>
          {canCreate ? (
            <Button size="sm" onClick={() => setPoolDialog({ pool: null })}>
              New pool
            </Button>
          ) : null}
        </div>
        {pools.length === 0 ? (
          <p className="text-muted-foreground rounded-md border border-dashed p-4 text-xs">
            No pool is configured. A pool names the drivers in the hat and the annual rates the
            draws must meet.
          </p>
        ) : (
          <ul className="flex flex-col gap-2">
            {pools.map((pool) => (
              <li key={pool.id} className="rounded-md border p-3">
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="text-sm font-medium">{pool.name}</span>
                      <Badge variant="secondary">{pool.code}</Badge>
                      {pool.isDefault ? <Badge variant="info">Default</Badge> : null}
                      <Badge variant={pool.status === "Active" ? "active" : "inactive"}>
                        {pool.status}
                      </Badge>
                      {pool.meetsDotMinimums ? null : (
                        <Badge variant="warning">Below the DOT minimum</Badge>
                      )}
                    </div>
                    <p className="text-muted-foreground mt-1 text-xs">
                      {randomPeriodLabel(pool.period)} · {pool.drugRatePercent}% drug (minimum{" "}
                      {DOT_MINIMUM_DRUG_RATE}%) · {pool.alcoholRatePercent}% alcohol (minimum{" "}
                      {DOT_MINIMUM_ALCOHOL_RATE}%)
                      {pool.includedDriverTypes.length > 0
                        ? ` · ${pool.includedDriverTypes.join(", ")}`
                        : " · every driver"}
                    </p>
                  </div>
                  <div className="flex shrink-0 gap-2">
                    {canDraw ? (
                      <Button
                        size="sm"
                        isLoading={drawMutation.isPending}
                        onClick={() => drawMutation.mutate(pool.id)}
                      >
                        Run this period&apos;s draw
                      </Button>
                    ) : null}
                    {canEdit ? (
                      <Button size="sm" variant="outline" onClick={() => setPoolDialog({ pool })}>
                        Edit
                      </Button>
                    ) : null}
                  </div>
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section>
        <h2 className="cc-label text-foreground mb-2">Rounds</h2>
        {draws.length === 0 ? (
          <p className="text-muted-foreground rounded-md border border-dashed p-4 text-xs">
            No round has been drawn yet.
          </p>
        ) : (
          <ul className="flex flex-col gap-2">
            {draws.map((draw) => {
              const shortOfTarget =
                draw.drugSelected < draw.drugTarget || draw.alcoholSelected < draw.alcoholTarget;
              return (
                <li key={draw.id} className="rounded-md border p-3">
                  <div className="flex flex-wrap items-start justify-between gap-3">
                    <div className="min-w-0">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="text-sm font-medium tabular-nums">{draw.periodKey}</span>
                        {draw.pool ? <Badge variant="secondary">{draw.pool.code}</Badge> : null}
                        <Badge
                          variant={
                            draw.status === "Final"
                              ? "active"
                              : draw.status === "Cancelled"
                                ? "inactive"
                                : "warning"
                          }
                        >
                          {draw.status}
                        </Badge>
                        {shortOfTarget && draw.status !== "Cancelled" ? (
                          <Badge variant="warning">Short of target</Badge>
                        ) : null}
                      </div>
                      <p className="text-muted-foreground mt-1 text-xs tabular-nums">
                        {draw.drugSelected}/{draw.drugTarget} drug · {draw.alcoholSelected}/
                        {draw.alcoholTarget} alcohol · {draw.poolSize} in the pool · drawn{" "}
                        {formatUnixDate(draw.drawnAt)}
                      </p>
                    </div>
                    <div className="flex shrink-0 gap-2">
                      <Button size="sm" variant="outline" onClick={() => setOpenDrawId(draw.id)}>
                        Open
                      </Button>
                      {canDraw && draw.status === "Draft" ? (
                        <>
                          <Button
                            size="sm"
                            isLoading={finalizeMutation.isPending}
                            onClick={() => finalizeMutation.mutate(draw.id)}
                          >
                            Finalise
                          </Button>
                          <Button
                            size="sm"
                            variant="ghost"
                            isLoading={cancelMutation.isPending}
                            onClick={() => cancelMutation.mutate(draw.id)}
                          >
                            Void
                          </Button>
                        </>
                      ) : null}
                    </div>
                  </div>
                </li>
              );
            })}
          </ul>
        )}
      </section>

      <RandomPoolDialog
        open={poolDialog !== null}
        onOpenChange={(open) => !open && setPoolDialog(null)}
        pool={poolDialog?.pool ?? null}
      />
      <DrawDetailSheet drawId={openDrawId} onOpenChange={(open) => !open && setOpenDrawId(null)} />
    </div>
  );
}
