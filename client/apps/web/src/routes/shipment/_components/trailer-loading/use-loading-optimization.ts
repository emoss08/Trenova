import { apiService } from "@/services/api";
import type {
  LoadingOptimizationRequest,
  LoadingOptimizationResult,
  StopInput,
} from "@/types/loading-optimization";
import type { Shipment } from "@/types/shipment";
import { useMutation } from "@tanstack/react-query";
import { useFormContext, useWatch } from "react-hook-form";

function extractDeliveryStops(moves: Shipment["moves"]): StopInput[] {
  if (!moves?.length) return [];

  const stops: StopInput[] = [];

  for (const move of moves) {
    if (!move?.stops) continue;

    for (const stop of move.stops) {
      if (!stop) continue;
      if (stop.type !== "Delivery" && stop.type !== "SplitDelivery") continue;

      stops.push({
        sequence: typeof move.sequence === "number" ? move.sequence : stops.length,
        locationName: stop.location?.name ?? "",
        locationCity: stop.location?.city ?? "",
      });
    }
  }

  return stops;
}

export interface RevenueContext {
  totalChargeAmount: number;
  distance: number;
  revenuePerFoot: number;
  revenuePerMile: number;
  emptySpaceFeet: number;
}

function computeRevenue(
  totalChargeAmount: number,
  distance: number,
  totalLinearFeet: number,
  trailerLengthFeet: number,
): RevenueContext | null {
  if (totalChargeAmount <= 0) return null;

  const emptySpaceFeet = Math.max(trailerLengthFeet - totalLinearFeet, 0);

  return {
    totalChargeAmount,
    distance,
    revenuePerFoot: totalLinearFeet > 0 ? totalChargeAmount / totalLinearFeet : 0,
    revenuePerMile: distance > 0 ? totalChargeAmount / distance : 0,
    emptySpaceFeet,
  };
}

export function useLoadingOptimization() {
  const { control } = useFormContext<Shipment>();
  const commodities = useWatch({ control, name: "commodities" }) ?? [];
  const moves = useWatch({ control, name: "moves" }) ?? [];
  const totalChargeAmount = useWatch({ control, name: "totalChargeAmount" }) ?? 0;

  const equipmentTypeId = moves[0]?.assignment?.trailer?.equipmentTypeId ?? undefined;
  const distance = moves[0]?.distance ?? 0;

  const mutation = useMutation<LoadingOptimizationResult, Error, undefined>({
    mutationFn: () => {
      const req: LoadingOptimizationRequest = {
        commodities: commodities
          .filter((c) => !!c?.commodityId)
          .map((c) => ({
            commodityId: c.commodityId,
            pieces: typeof c.pieces === "number" ? c.pieces : 1,
            weight: typeof c.weight === "number" ? c.weight : 0,
          })),
        equipmentTypeId,
        stops: extractDeliveryStops(moves),
      };
      return apiService.shipmentService.calculateLoadingOptimization(req);
    },
  });

  const chargeNum = typeof totalChargeAmount === "number" ? totalChargeAmount : Number(totalChargeAmount) || 0;
  const distNum = typeof distance === "number" ? distance : Number(distance) || 0;

  const revenue = mutation.data
    ? computeRevenue(chargeNum, distNum, mutation.data.totalLinearFeet, mutation.data.trailerLengthFeet)
    : null;

  return {
    data: mutation.data,
    revenue,
    calculate: () => mutation.mutate(undefined),
    isPending: mutation.isPending,
    hasCommodities: commodities.filter((c) => !!c?.commodityId).length > 0,
  };
}
