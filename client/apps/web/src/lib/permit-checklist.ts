import type { Permit, PermitRequirement } from "@trenova/shared/types/permit";

/** Where one state's permit stands on the shipment's checklist. */
export type PermitChecklistPhase = "todo" | "filed" | "waived";

/**
 * One line of the checklist: a state's requirement with the permit that satisfies it, or a
 * permit on file that no current requirement asks for.
 */
export type PermitChecklistEntry = {
  key: string;
  phase: PermitChecklistPhase;
  requirement: PermitRequirement | null;
  permit: Permit | null;
};

export type PermitChecklist = {
  entries: PermitChecklistEntry[];
  /** Requirements still waiting for a permit. */
  open: number;
  /** Requirements a permit satisfies or someone waived. */
  resolved: number;
};

function phaseOf(requirement: PermitRequirement): PermitChecklistPhase {
  if (requirement.status === "Open") return "todo";
  if (requirement.status === "Waived") return "waived";
  return "filed";
}

const PHASE_ORDER: Record<PermitChecklistPhase, number> = { todo: 0, filed: 1, waived: 2 };

/**
 * The shipment's permits as one checklist, a line per state: what is still to file first (in
 * route order), then what is filed or waived, then any permit on file that no requirement asks
 * for. A requirement superseded by a newer derivation is no longer asked for, so it is left out.
 * A satisfied requirement shows the permit that satisfied it, or, for one saved before that link
 * was kept, the permit filed for the same state.
 */
export function permitChecklist(
  requirements: readonly PermitRequirement[],
  permits: readonly Permit[],
): PermitChecklist {
  const used = new Set<string>();
  const current = requirements.filter((requirement) => requirement.status !== "Superseded");

  const fromRequirements = current.map((requirement): PermitChecklistEntry => {
    const phase = phaseOf(requirement);
    const permit =
      phase === "filed"
        ? (permits.find((entry) => entry.id === requirement.satisfiedByPermitId) ??
          permits.find((entry) => entry.stateId === requirement.stateId && !used.has(entry.id)) ??
          null)
        : null;
    if (permit) used.add(permit.id);
    return {
      key: requirement.id ?? `${requirement.stateId}-${requirement.routeSequence}`,
      phase,
      requirement,
      permit,
    };
  });

  fromRequirements.sort(
    (a, b) =>
      PHASE_ORDER[a.phase] - PHASE_ORDER[b.phase] ||
      (a.requirement?.routeSequence ?? 0) - (b.requirement?.routeSequence ?? 0),
  );

  const unclaimed = permits
    .filter((entry) => !used.has(entry.id))
    .map((permit): PermitChecklistEntry => ({
      key: permit.id,
      phase: "filed",
      requirement: null,
      permit,
    }));

  const open = current.filter((requirement) => requirement.status === "Open").length;
  return {
    entries: [...fromRequirements, ...unclaimed],
    open,
    resolved: current.length - open,
  };
}
