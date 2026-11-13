import { Kept } from "@/components/kept";
import { usePermission } from "@/hooks/use-permission";
import { lazyComponent } from "@/lib/lazy-component";
import { deskAttention } from "@/lib/queries/desk-attention";
import { whenIdle } from "@/lib/when-idle";
import { useDeskSetting } from "@/stores/desk-settings-store";
import { useQueryClient } from "@tanstack/react-query";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { getStartOfDay, resolveUserTimezone } from "@trenova/shared/lib/date";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useEffect, useState } from "react";
import { useLocation, useNavigate, useSearchParams } from "react-router";
import { pendingFilter } from "../decisions/decision-filters";
import { useDecisionsHead } from "../decisions/use-decisions-head";
import { pendingDecisionsOptions, startOfToday } from "../decisions/use-pending-decisions";
import { DESK_VIEWS_HEAD_CLASS, DeskViews } from "../list/desk-views";
import { NEEDS_LENSES, lensFromParam, lensParam, type NeedsLens, type RecordLens } from "./lens";
import { NeedsAttention } from "./needs-attention";
import { attentionFilter } from "./needs-filters";
import { NeedsHead } from "./needs-head";
import { NeedsLensSkeleton } from "./needs-lens-skeleton";
import { useAttentionHead } from "./use-attention-head";

/**
 * The Decisions lens is read when it is first opened, or a moment before:
 * when the pointer reaches its tab, and once the page has nothing else to do.
 */
const decisionsLens = lazyComponent(
  () => import("../decisions/decision-flow"),
  (module) => module.DecisionFlow,
);

/** How long a lens read ahead on the way to its tab is taken as it is. */
const AHEAD_FRESH_MS = 15_000;

/**
 * Needs you: everything that waits on people, in one place. Watchtower and
 * Decisions are lenses on it, not separate pages, and the page is one frame
 * for all of them: the head is drawn once and stays, and a lens brings only
 * its list and what the chosen row reads beside it. Picking a lens moves the
 * marker under the tabs and changes those two in place; nothing else on the
 * page is taken down or drawn again, and a lens left keeps where the person
 * was in it.
 *
 * Everything and Watchtower list by record, soonest deadline first.
 * Decisions lists what agents ask to change by the kind of change, so a run
 * of the same change is read on one sheet and decided one at a time or
 * together. Every lens is searched by the same words, which stay as the
 * person goes from one to the next, and narrowed by what it holds.
 *
 * A lens is read when it is first opened, and the one the pointer is heading
 * for is read ahead, so it is there when it is picked.
 */
export function DeskNeeds() {
  const { allowed: canWatch } = usePermission(Resource.Watchtower, Operation.Read);
  const { allowed: canDecide } = usePermission(Resource.AgentProposal, Operation.Read);
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const location = useLocation();
  const [searchParams] = useSearchParams();

  const lenses = NEEDS_LENSES.filter((value) => (value === "Decisions" ? canDecide : canWatch));
  const asked = lensFromParam(searchParams.get("lens"));
  const lens = lenses.includes(asked) ? asked : (lenses[0] ?? asked);
  const onDecisions = lens === "Decisions";

  // The lens that lists by record stays the one last opened while Decisions
  // is in front, so coming back to it finds it as it was left.
  const [recordLens, setRecordLens] = useState<RecordLens>(onDecisions ? "Everything" : lens);
  if (!onDecisions && lens !== recordLens) {
    setRecordLens(lens);
  }

  const timezone = resolveUserTimezone(useAuthStore((state) => state.user)?.timezone);
  const [clearedSince] = useState(() => getStartOfDay(new Date(), timezone));
  const [decidedSince] = useState(startOfToday);
  const mine = useDeskSetting("cases") === "mine";

  // One filter button for the page; the lens in front says what it opens.
  const [filterOpen, setFilterOpen] = useState(false);
  const attention = useAttentionHead({
    lens: recordLens,
    enabled: canWatch,
    since: clearedSince,
    filterOpen,
    onFilterOpen: setFilterOpen,
  });
  // What Decisions says in the head is read while it is in front. Left, the
  // lens keeps what it read and stops asking; the by-record list is read
  // throughout, since the tabs count what waits in it under every lens.
  const decisions = useDecisionsHead({
    enabled: canDecide && onDecisions,
    since: decidedSince,
    filterOpen,
    onFilterOpen: setFilterOpen,
  });

  const setLens = (next: NeedsLens) => {
    if (next === lens) {
      return;
    }
    // The address as it stands, not as the router last heard it: the search
    // and the filters are written to it without a navigation.
    const params = new URLSearchParams(window.location.search);
    const param = lensParam(next);
    if (param === "") {
      params.delete("lens");
    } else {
      params.set("lens", param);
    }
    void navigate(
      { search: params.toString() },
      { replace: true, viewTransition: true, state: location.state },
    );
  };

  const readAhead = (next: NeedsLens) => {
    if (next === lens) {
      return;
    }
    if (next === "Decisions") {
      void decisionsLens.preload();
      void queryClient.prefetchInfiniteQuery({
        ...pendingDecisionsOptions(pendingFilter(decisions.filters)),
        staleTime: AHEAD_FRESH_MS,
      });
      return;
    }
    void queryClient.prefetchQuery({
      ...deskAttention.list(attentionFilter(next, mine, attention.filters)),
      staleTime: AHEAD_FRESH_MS,
    });
  };

  // The keys pick a lens with no pointer on the way to it, so the lens's
  // code is read once the page is idle.
  useEffect(() => {
    if (!canDecide) {
      return;
    }
    return whenIdle(() => void decisionsLens.preload());
  }, [canDecide]);

  const head = onDecisions ? decisions.head : attention.head;

  return (
    <DeskViews>
      <NeedsHead
        className={DESK_VIEWS_HEAD_CLASS}
        lens={lens}
        lenses={lenses}
        onLens={setLens}
        onLensIntent={readAhead}
        counts={{
          Everything: attention.counts?.everything,
          Watchtower: attention.counts?.watchtower,
          // Someone who decides but does not watch has the queue's own count.
          Decisions: attention.counts?.decisions ?? decisions.total ?? undefined,
        }}
        keysEnabled={!filterOpen}
        onOpenFilter={() => setFilterOpen(true)}
        {...head}
      />
      {canWatch && (
        <Kept shown={!onDecisions}>
          <NeedsAttention
            lens={recordLens}
            shown={!onDecisions}
            filters={attention.filters}
            onFilters={attention.setFilters}
            since={clearedSince}
            keysPaused={filterOpen}
          />
        </Kept>
      )}
      {canDecide && (
        <Kept shown={onDecisions}>
          <LazyComponent fallback={<NeedsLensSkeleton lens="Decisions" />}>
            <decisionsLens.Component
              shown={onDecisions}
              filter={decisions.filters}
              onFilter={decisions.setFilters}
              since={decidedSince}
              keysPaused={filterOpen}
            />
          </LazyComponent>
        </Kept>
      )}
    </DeskViews>
  );
}
