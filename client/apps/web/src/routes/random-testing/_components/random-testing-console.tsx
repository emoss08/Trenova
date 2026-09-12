import { useT } from "@trenova/shared/i18n/use-t";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import { usePermission } from "@/hooks/use-permission";
import {
  cancelDotRandomDraw,
  DOT_RANDOM_DRAWS_KEY,
  DOT_RANDOM_POOLS_KEY,
  finalizeDotRandomDraw,
  runDotRandomDraw,
  type RandomPoolRow,
} from "@/lib/graphql/worker-drug-alcohol";
import {
  filterDraws,
  isRoundStatusFilter,
  poolCalendar,
  poolProgress,
  programmeOverview,
  type RoundStatusFilter,
} from "@/lib/random-testing";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { LayersIcon, ListChecksIcon, PlusIcon, XIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { ConfirmRoundDialog, type RoundAction } from "./confirm-round-dialog";
import { DrawDetailSheet } from "./draw-detail-sheet";
import { PoolRow } from "./pool-row";
import { randomDrawsQuery, randomPoolsQuery } from "./queries";
import { RandomPoolDialog } from "./random-pool-dialog";
import { RandomTestingEmpty } from "./random-testing-empty";
import { RandomTestingOverview } from "./random-testing-overview";
import { RandomTestingSkeleton } from "./random-testing-skeleton";
import { RoundsTable } from "./rounds-table";

const STATUS_ITEMS = [
  { value: "all", label: "All" },
  { value: "Draft", label: "Draft" },
  { value: "Final", label: "Final" },
  { value: "Cancelled", label: "Voided" },
] satisfies { value: RoundStatusFilter; label: string }[];

export default function RandomTestingConsole() {
  const t = useT();

  const queryClient = useQueryClient();
  const { allowed: canEdit } = usePermission(Resource.DOTRandomPool, Operation.Update);
  const { allowed: canCreate } = usePermission(Resource.DOTRandomPool, Operation.Create);
  const { allowed: canDraw } = usePermission(Resource.DOTRandomPool, Operation.Manage);

  const [poolId, setPoolId] = useState<string | null>(null);
  const [status, setStatus] = useState<RoundStatusFilter>("all");
  const [poolDialog, setPoolDialog] = useState<{ pool: RandomPoolRow | null } | null>(null);
  const [openDrawId, setOpenDrawId] = useState<string | null>(null);
  const [roundAction, setRoundAction] = useState<RoundAction | null>(null);
  // Read once on mount: a slot flipping from owed to missed mid-visit is not
  // something the page should do on its own.
  const [now] = useState(() => Math.floor(Date.now() / 1000));

  const poolsQuery = useQuery(randomPoolsQuery());
  const drawsQuery = useQuery(randomDrawsQuery());

  const invalidate = useCallback(
    () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: [DOT_RANDOM_POOLS_KEY] }),
        queryClient.invalidateQueries({ queryKey: [DOT_RANDOM_DRAWS_KEY] }),
      ]),
    [queryClient],
  );

  const drawMutation = useMutation({
    mutationFn: (id: string) => runDotRandomDraw({ poolId: id }),
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
      toast.error(t("Could not run the draw"), { description: error.message }),
  });

  const finaliseMutation = useMutation({
    mutationFn: (id: string) => finalizeDotRandomDraw(id),
    onSuccess: () => {
      toast.success(t("Round finalised"), {
        description: t("The names are now the record. Correcting one means voiding the round."),
      });
      setRoundAction(null);
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error(t("Could not finalise the round"), { description: error.message }),
  });

  const voidMutation = useMutation({
    mutationFn: (id: string) => cancelDotRandomDraw(id, "Voided from the random testing console"),
    onSuccess: () => {
      toast.success(t("Round voided"), { description: t("The period can be drawn again.") });
      setRoundAction(null);
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error(t("Could not void the round"), { description: error.message }),
  });

  const pools = poolsQuery.data;
  const draws = drawsQuery.data;
  const year = new Date(now * 1000).getUTCFullYear();

  const overview = useMemo(
    () => (pools && draws ? programmeOverview(pools, draws, now) : null),
    [pools, draws, now],
  );
  const rows = useMemo(
    () =>
      pools && draws
        ? pools.map((pool) => ({
            pool,
            calendar: poolCalendar(pool, draws, now, year),
            progress: poolProgress(pool, draws, now, year),
          }))
        : [],
    [pools, draws, now, year],
  );
  const rounds = useMemo(
    () => (draws ? filterDraws(draws, { poolId, status }) : []),
    [draws, poolId, status],
  );
  const selectedPool = useMemo(
    () => pools?.find((pool) => pool.id === poolId) ?? null,
    [pools, poolId],
  );

  if ((poolsQuery.isLoading || drawsQuery.isLoading) && !(pools && draws)) {
    return <RandomTestingSkeleton />;
  }

  const failure = poolsQuery.error ?? drawsQuery.error;
  if (failure || !pools || !draws || !overview) {
    return (
      <div className="text-destructive rounded-lg border border-dashed p-4 text-sm">
        {t("The programme could not be read. {0}", failure?.message)}
      </div>
    );
  }

  if (pools.length === 0) {
    return (
      <div className="flex flex-col gap-4">
        <RandomTestingOverview overview={overview} />
        <RandomTestingEmpty
          title={t("No pool is configured")}
          description={t(
            "A pool names the drivers in the hat and the annual rates the draws must meet. Draws cannot run until one exists.",
          )}
          onNewPool={canCreate ? () => setPoolDialog({ pool: null }) : undefined}
        />
        <RandomPoolDialog
          open={poolDialog !== null}
          onOpenChange={(open) => !open && setPoolDialog(null)}
          pool={poolDialog?.pool ?? null}
        />
      </div>
    );
  }

  const busyId =
    finaliseMutation.isPending || voidMutation.isPending ? (roundAction?.draw.id ?? null) : null;

  return (
    <div className="flex flex-col gap-4">
      <RandomTestingOverview overview={overview} />

      {overview.missed > 0 ? (
        <Alert variant="warning">
          <AlertDescription>
            {overview.missed === 1
              ? t(
                  "A round this year was never drawn. A missed period cannot be drawn after it has ended; record why in the pool's description so the gap is explained when the programme is audited.",
                )
              : t(
                  "{0} rounds this year were never drawn. A missed period cannot be drawn after it has ended; record why in the pool's description so the gap is explained when the programme is audited.",
                  overview.missed,
                )}
          </AlertDescription>
        </Alert>
      ) : null}

      <SectionPanel
        title={t("Pools")}
        icon={<LayersIcon />}
        help={t(
          "Each pool names the drivers in the hat and the annual rates its draws must meet. The strip is this year's rounds: filled is final, dashed is drawn but not final, red was never drawn.",
        )}
        count={pools.length}
        hint={`Rounds in ${year}`}
        action={
          canCreate ? (
            <Button size="xs" variant="outline" onClick={() => setPoolDialog({ pool: null })}>
              <PlusIcon className="size-3" />
              {t("New pool")}
            </Button>
          ) : null
        }
      >
        <ul aria-label={t("Pools")} className="divide-y">
          {rows.map(({ pool, calendar, progress }) => (
            <PoolRow
              key={pool.id}
              pool={pool}
              calendar={calendar}
              progress={progress}
              year={year}
              selected={pool.id === poolId}
              canDraw={canDraw}
              canEdit={canEdit}
              drawing={drawMutation.isPending && drawMutation.variables === pool.id}
              onSelect={setPoolId}
              onDraw={(id) => drawMutation.mutate(id)}
              onEdit={(target) => setPoolDialog({ pool: target })}
            />
          ))}
        </ul>
      </SectionPanel>

      <SectionPanel
        title={t("Rounds")}
        icon={<ListChecksIcon />}
        help={t(
          "Every draw, newest first. A draft can still be finalised or voided; a final round is the record and only opens.",
        )}
        hint={
          rounds.length === draws.length
            ? `${draws.length} round${draws.length === 1 ? "" : "s"}`
            : `${rounds.length} of ${draws.length}`
        }
        action={
          <>
            {selectedPool ? (
              <Button
                size="xs"
                variant="outline"
                aria-label={`Clear pool ${selectedPool.code}`}
                onClick={() => setPoolId(null)}
              >
                {selectedPool.code}
                <XIcon className="size-3" />
              </Button>
            ) : null}
            <SegmentedControl<RoundStatusFilter>
              items={STATUS_ITEMS}
              value={status}
              onValueChange={(value) => setStatus(isRoundStatusFilter(value) ? value : "all")}
              aria-label={t("Round status")}
            />
          </>
        }
      >
        {rounds.length === 0 ? (
          <SectionPanelQuiet>
            {draws.length === 0
              ? t("No round has been drawn yet.")
              : t("No round matches this pool and status.")}
          </SectionPanelQuiet>
        ) : (
          <RoundsTable
            rows={rounds}
            canDraw={canDraw}
            busyId={busyId}
            onOpen={setOpenDrawId}
            onFinalise={(draw) => setRoundAction({ kind: "finalise", draw })}
            onVoid={(draw) => setRoundAction({ kind: "void", draw })}
          />
        )}
      </SectionPanel>

      <RandomPoolDialog
        open={poolDialog !== null}
        onOpenChange={(open) => !open && setPoolDialog(null)}
        pool={poolDialog?.pool ?? null}
      />
      <DrawDetailSheet drawId={openDrawId} onOpenChange={(open) => !open && setOpenDrawId(null)} />
      <ConfirmRoundDialog
        action={roundAction}
        pending={finaliseMutation.isPending || voidMutation.isPending}
        onOpenChange={(open) => !open && setRoundAction(null)}
        onConfirm={(action) =>
          action.kind === "finalise"
            ? finaliseMutation.mutate(action.draw.id)
            : voidMutation.mutate(action.draw.id)
        }
      />
    </div>
  );
}
