import { getNameInitials } from "@trenova/shared/lib/utils";
import { isActiveCarrierAssignment, type Shipment } from "@trenova/shared/types/shipment";

function initialsOf(name: string): string {
  return getNameInitials(name.replace(/[-.&+]/g, " "));
}

export type Coverage =
  | { kind: "driver"; id: string; name: string; initials: string; detail: string | null }
  | { kind: "carrier"; id: string; name: string; initials: string; detail: string | null }
  | { kind: "tendered" }
  | { kind: "uncovered" };

/**
 * Who is hauling a shipment, read from its first move: a driver, a carrier,
 * a tender waiting on a carrier's answer, or nobody. Every surface that shows
 * coverage reads it here, so a brokered load never asks for a driver.
 */
export function resolveCoverage(shipment: Shipment): Coverage {
  const move = shipment.moves?.[0];
  const carrierAssignment = move?.carrierAssignment;
  if (move?.coverageType === "carrier" && isActiveCarrierAssignment(carrierAssignment)) {
    const carrier = carrierAssignment?.carrier;
    const name = carrier?.name || "External carrier";
    return {
      kind: "carrier",
      id: carrier?.id ?? name,
      name,
      initials: initialsOf(name),
      detail: carrierAssignment?.externalDriverName || carrier?.scac || null,
    };
  }

  const driver = move?.assignment?.primaryWorker;
  if (driver) {
    const name = [driver.firstName, driver.lastName].filter(Boolean).join(" ").trim() || "—";
    return {
      kind: "driver",
      id: driver.id ?? name,
      name,
      initials: initialsOf(name),
      detail: move?.assignment?.tractor?.code ?? null,
    };
  }

  if (shipment.tenderStatus === "Tendered") {
    return { kind: "tendered" };
  }
  return { kind: "uncovered" };
}
