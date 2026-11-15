import { PlanFeature } from "@/lib/plan-feature";
import { shipmentBoard } from "@/lib/queries/shipment-board";
import type { ShipmentBoardCapabilities } from "@/lib/graphql/shipment-board";
import { useSuspenseQuery } from "@tanstack/react-query";
import { usePlanFeature } from "@trenova/shared/hooks/use-plan-feature";
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
 * A capability that belongs to an add-on is on only while the plan includes it:
 * the briefing and insights are Trenova AI's, the ELD and the map Visibility's.
 */
export function ShipmentCapabilitiesProvider({ children }: { children: ReactNode }) {
  const { data } = useSuspenseQuery({ ...shipmentBoard.capabilities(), staleTime: 60_000 });
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
  const aiOnPlan = usePlanFeature(PlanFeature.AIDesk).available;
  const hosOnPlan = usePlanFeature(PlanFeature.Samsara).available;
  const mapsOnPlan = usePlanFeature(PlanFeature.GoogleMaps).available;
  const capabilities = toBoardCapabilities({
    ...value,
    ai: value.ai && aiOnPlan,
    hos: value.hos && hosOnPlan,
    maps: value.maps && mapsOnPlan,
  });
  return <CapabilitiesContext value={capabilities}>{children}</CapabilitiesContext>;
}

export function useShipmentCapabilities(): BoardCapabilities {
  const value = use(CapabilitiesContext);
  if (!value) {
    throw new Error("useShipmentCapabilities must be used inside ShipmentCapabilitiesProvider");
  }
  return value;
}
