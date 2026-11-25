import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useMemo, useRef } from "react";
import { usePermission } from "@/hooks/use-permission";
import { assignDispatchMovesGraphQL, type DispatchBoardMove } from "@/lib/graphql/dispatch-console";
import { DISPATCH_BOARD_KEY, dispatchConsoleQueries } from "@/lib/queries/dispatch-console";
import { withoutRecordIds } from "@/lib/record-ids";
import type { AssistantArtifact } from "@/types/assistant";
import { dispatchPlanFrom, type DispatchPlanArtifact } from "../../artifact-payloads";
import { CardActionButton } from "../../record-card/card-action";
import {
  CardBar,
  CardBarHint,
  CardHead,
  CardIcon,
  CardSection,
  CardSectionError,
  CardStats,
  CardStatus,
  type CardTone,
  RecordCard,
} from "../../record-card/record-card";
import {
  assignableRows,
  type LoadState,
  loadStateOf,
  type PlanStanding,
  planMoveIds,
  planStanding,
  planTally,
} from "./dispatch-plan-model";
import { PlanGap, PlanPair } from "./dispatch-plan-parts";
import { DispatchPlanSkeleton } from "./dispatch-plan-skeleton";

const STANDING_TONE: Record<PlanStanding, CardTone> = {
  plan: "progress",
  proposed: "warn",
  partial: "progress",
  assigned: "success",
  empty: "danger",
};

function standingWords(standing: PlanStanding, t: TranslateFn): string {
  switch (standing) {
    case "plan":
      return t("Planned, nothing assigned");
    case "proposed":
      return t("Waiting for a decision");
    case "partial":
      return t("Partly assigned");
    case "assigned":
      return t("Assigned");
    case "empty":
      return t("Nothing can be covered");
  }
}

function barHint(
  plan: DispatchPlanArtifact,
  assignable: number,
  mayAssign: boolean,
  t: TranslateFn,
): string {
  if (plan.lines.length === 0) {
    return t("No driver can take these loads as things stand; offer them to a carrier.");
  }
  if (!mayAssign) {
    return t("Assigning drivers takes the move assign permission.");
  }
  if (assignable === 0) {
    return t("Every planned load is covered or closed.");
  }

  return t("Assigning tells each driver about their load.");
}

/**
 * A dispatch plan: each load beside the driver the optimizer put on it, what that costs and how
 * sure it is, the loads nobody can take and why, read live for which loads are covered now, and
 * the assignments made from it in one step.
 */
export function DeskDispatchPlanBody({ artifact }: { artifact: AssistantArtifact }) {
  const plan = useMemo(() => dispatchPlanFrom(artifact), [artifact]);
  const moveIds = useMemo(() => planMoveIds(plan), [plan]);
  const movesQuery = useQuery({
    ...dispatchConsoleQueries.moves(moveIds),
    enabled: moveIds.length > 0,
  });

  if (moveIds.length > 0 && movesQuery.isPending) {
    return <DispatchPlanSkeleton />;
  }

  return (
    <DispatchPlanCard
      title={artifact.title}
      plan={plan}
      moves={movesQuery.data ?? []}
      unread={movesQuery.isError}
    />
  );
}

function DispatchPlanCard({
  title,
  plan,
  moves,
  unread,
}: {
  title: string;
  plan: DispatchPlanArtifact;
  moves: readonly DispatchBoardMove[];
  unread: boolean;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed: mayAssign } = usePermission(Resource.ShipmentMove, Operation.Assign);
  const sending = useRef(false);

  const byMove = new Map(moves.map((move) => [move.moveId, move]));
  const states = new Map<string, LoadState>(
    plan.lines.map((line) => [line.moveId, loadStateOf(line, byMove.get(line.moveId))]),
  );
  const assignable = assignableRows(plan, states);
  const standing = planStanding(plan, states);
  const tally = planTally(plan);

  const assign = useMutation({
    mutationFn: () =>
      assignDispatchMovesGraphQL(
        assignable.map((row) => ({
          moveId: row.moveId,
          primaryWorkerId: row.workerId,
          tractorId: row.tractorId,
          trailerId: row.trailerId,
        })),
      ),
    onSettled: () => {
      sending.current = false;
      void queryClient.invalidateQueries({ queryKey: [DISPATCH_BOARD_KEY] });
    },
  });
  const assignAll = async () => {
    if (sending.current) {
      return;
    }
    sending.current = true;
    await assign.mutateAsync().catch(() => undefined);
  };

  const refused = (assign.data?.results ?? []).filter((result) => !result.success);
  const proOf = (moveId: string) =>
    plan.lines.find((line) => line.moveId === moveId)?.proNumber ?? t("a load");

  return (
    <RecordCard className="dk-dpl">
      <CardHead
        kicker={[
          t("Dispatch plan"),
          t("{0, plural, one {# load} other {# loads}}", tally.loads),
          plan.planningMode,
        ].filter((piece) => piece !== "")}
        title={title}
        subtitle={t("Planned together, so no driver is on two loads")}
        figure={plan.cost === "" ? undefined : plan.cost}
        meta={<CardStatus tone={STANDING_TONE[standing]}>{standingWords(standing, t)}</CardStatus>}
      />
      <CardStats
        className="dk-dpl-tally"
        stats={[
          {
            key: "covered",
            label: t("Covered"),
            value: t("{0} of {1}", tally.covered, tally.loads),
          },
          {
            key: "gaps",
            label: t("No driver"),
            value: tally.gaps,
            state: tally.gaps > 0 ? "bad" : "ok",
          },
          {
            key: "empty",
            label: t("Empty miles"),
            value: tally.emptyMiles === null ? t("Unknown") : tally.emptyMiles.toLocaleString(),
          },
          {
            key: "check",
            label: t("To check"),
            value: tally.toCheck,
            state: tally.toCheck > 0 ? "warn" : "ok",
          },
        ]}
      />
      <CardSection title={t("Who takes each load")}>
        {unread && (
          <CardSectionError>
            {t("Which loads are covered now could not be read; this is the plan as it was made.")}
          </CardSectionError>
        )}
        <ol className="dk-dpl-ledger">
          {plan.lines.map((line) => (
            <PlanPair
              key={line.moveId}
              line={line}
              state={states.get(line.moveId) ?? "unread"}
              move={byMove.get(line.moveId)}
            />
          ))}
          {plan.gaps.map((gap) => (
            <PlanGap key={gap.moveId} gap={gap} move={byMove.get(gap.moveId)} />
          ))}
        </ol>
        {plan.notPlanned > 0 && (
          <p className="dk-dpl-left">
            {t(
              "{0, plural, one {# load you named was not planned: it is covered, closed or picks up later.} other {# loads you named were not planned: they are covered, closed or pick up later.}}",
              plan.notPlanned,
            )}
          </p>
        )}
        {assign.isError && (
          <CardSectionError>{t("The assignments could not be made. Try again.")}</CardSectionError>
        )}
        {refused.length > 0 && (
          <CardSectionError>
            {t(
              "{0, plural, one {# load was not assigned:} other {# loads were not assigned:}}",
              refused.length,
            )}{" "}
            {refused
              .map((result) =>
                t("{0}: {1}", proOf(result.moveId), withoutRecordIds(result.error ?? t("refused"))),
              )
              .join("; ")}
          </CardSectionError>
        )}
      </CardSection>
      <CardBar>
        <CardBarHint>{barHint(plan, assignable.length, mayAssign, t)}</CardBarHint>
        <Link className="dk-ax-btn" to="/dispatch/console">
          <CardIcon name="ext" size={13} />
          {t("Open dispatch board")}
        </Link>
        {mayAssign && plan.lines.length > 0 && (
          <CardActionButton
            blocker={assignable.length === 0 ? t("Every planned load is covered or closed.") : null}
            busy={assign.isPending}
            isLoading={assign.isPending}
            onClick={assignAll}
          >
            {assignable.length === 0
              ? t("Assign loads")
              : t("{0, plural, one {Assign # load} other {Assign # loads}}", assignable.length)}
          </CardActionButton>
        )}
      </CardBar>
    </RecordCard>
  );
}
