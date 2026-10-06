import { queries } from "@/lib/queries";
import type { ShipmentBoardCapabilities } from "@/lib/graphql/shipment-board";
import { useSuspenseQuery } from "@tanstack/react-query";
import { createContext, use, type ReactNode } from "react";

export type BoardCapabilities = ShipmentBoardCapabilities & {
  runsAssets: boolean;
  runsBrokerage: boolean;
};

const CapabilitiesContext = createContext<BoardCapabilities | null>(null);

export function toBoardCapabilities(raw: ShipmentBoardCapabilities): BoardCapabilities {
  return {
    ...raw,
    runsAssets: raw.operationType === "asset" || raw.operationType === "both",
    runsBrokerage: raw.operationType === "brokerage" || raw.operationType === "both",
  };
}

/**
 * What this workspace can do: an AI provider, its own drivers, carriers, an
 * ELD, a map. Every conditional on the board reads it here, so turning a
 * capability on or off is one server fact rather than a branch per component.
 */
export function ShipmentCapabilitiesProvider({ children }: { children: ReactNode }) {
  const { data } = useSuspenseQuery({ ...queries.shipmentBoard.capabilities(), staleTime: 60_000 });
  return <BoardCapabilitiesProvider value={data}>{children}</BoardCapabilitiesProvider>;
}

/** Supplies capabilities already in hand, such as a test's or a story's. */
export function BoardCapabilitiesProvider({
  value,
  children,
}: {
  value: ShipmentBoardCapabilities;
  children: ReactNode;
}) {
  return <CapabilitiesContext value={toBoardCapabilities(value)}>{children}</CapabilitiesContext>;
}

export function useShipmentCapabilities(): BoardCapabilities {
  const value = use(CapabilitiesContext);
  if (!value) {
    throw new Error("useShipmentCapabilities must be used inside ShipmentCapabilitiesProvider");
  }
  return value;
}
