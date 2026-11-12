import { usePermission } from "@/hooks/use-permission";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { parseAsString, useQueryState } from "nuqs";
import { DecisionFlow } from "../decisions/decision-flow";
import { NEEDS_LENSES, lensFromParam, lensParam, type NeedsLens } from "./lens";
import { NeedsAttention } from "./needs-attention";
import { NeedsHead } from "./needs-head";
import { useLensCounts } from "./use-lens-counts";

/**
 * Needs you: everything that waits on people, in one place. Watchtower and
 * Decisions are lenses on it, not separate pages, and every lens keeps the
 * same head, the same list on the left and the same keys. Everything and
 * Watchtower list by record, soonest deadline first. Decisions lists what
 * agents ask to change by the kind of change, so a run of the same change is
 * read on one sheet and decided one at a time or together.
 */
export function DeskNeeds() {
  const { allowed: canWatch } = usePermission(Resource.Watchtower, Operation.Read);
  const { allowed: canDecide } = usePermission(Resource.AgentProposal, Operation.Read);
  const [param, setParam] = useQueryState("lens", parseAsString.withOptions({ history: "replace" }));
  const counts = useLensCounts(canWatch);

  const lenses = NEEDS_LENSES.filter((value) => (value === "Decisions" ? canDecide : canWatch));
  const asked = lensFromParam(param);
  const lens = lenses.includes(asked) ? asked : (lenses[0] ?? asked);
  const setLens = (next: NeedsLens) => void setParam(lensParam(next) || null);

  if (lens !== "Decisions") {
    return <NeedsAttention lens={lens} lenses={lenses} onLens={setLens} />;
  }

  return (
    <DecisionFlow
      head={({ progressLabel, done, total, waiting, keysEnabled }) => (
        <NeedsHead
          lens={lens}
          lenses={lenses}
          onLens={setLens}
          counts={{
            Everything: counts.everything,
            Watchtower: counts.watchtower,
            // Someone who decides but does not watch has no counts to read; the
            // queue's own count stands in.
            Decisions: canWatch ? counts.decisions : waiting,
          }}
          progressLabel={progressLabel}
          done={done}
          total={total}
          keysEnabled={keysEnabled}
        />
      )}
    />
  );
}
