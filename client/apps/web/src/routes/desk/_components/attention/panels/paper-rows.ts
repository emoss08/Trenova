import type { ShipmentBillingRequirement } from "@trenova/shared/types/shipment";

export type PaperRow = {
  key: string;
  name: string;
  /** The kind of document a file is filed as; null when the checklist does not name one. */
  documentTypeId: string | null;
  /** On file and accepted; filed but not yet accepted; or not there at all. */
  state: "onFile" | "waiting" | "missing";
};

/**
 * The papers a load is waiting on, as rows. The shipment's own checklist is
 * the truth when it has been read: every document billing asks for, and where
 * each stands. Until then, or for a shipment billing asks nothing of, the
 * rows are the papers the Watchtower named when it raised the item.
 */
export function paperRows(
  named: readonly { name: string; onFile: boolean }[],
  requirements: readonly ShipmentBillingRequirement[] | undefined,
): PaperRow[] {
  if (requirements !== undefined && requirements.length > 0) {
    return requirements.map((requirement) => ({
      key: requirement.documentTypeId,
      name: requirement.documentTypeName,
      documentTypeId: requirement.documentTypeId,
      state: requirement.satisfied
        ? "onFile"
        : requirement.documentCount > 0
          ? "waiting"
          : "missing",
    }));
  }

  return named.map((paper) => ({
    key: paper.name,
    name: paper.name,
    documentTypeId: null,
    state: paper.onFile ? "onFile" : "missing",
  }));
}
