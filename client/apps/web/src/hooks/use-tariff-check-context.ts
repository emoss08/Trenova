import { formulaTemplate } from "@/lib/queries/formula-template";
import { useQuery } from "@tanstack/react-query";
import type { TariffContext } from "@trenova/shared/types/tariff";

const CONTEXT_STALE_MS = 5 * 60_000;

export type TariffCheckContextState = {
  /**
   * The organisation's settings, rate table shapes, accessorial codes and importable library
   * sources; null until they load, which checks with no tables and leaves codes unchecked.
   */
  context: TariffContext | null;
  isError: boolean;
  refetch: () => void;
};

/**
 * What every Tariff file in the browser is checked against (`GET /formula-templates/context`),
 * read once and shared by every editor on the screen.
 */
export function useTariffCheckContext(): TariffCheckContextState {
  const query = useQuery({ ...formulaTemplate.context, staleTime: CONTEXT_STALE_MS });
  return {
    context: query.data ?? null,
    isError: query.isError,
    refetch: () => void query.refetch(),
  };
}
