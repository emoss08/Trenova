import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { RefreshCw02Icon } from "@trenova/shared/components/icons";
import { cn } from "@trenova/shared/lib/utils";
import { queries } from "@/lib/queries";
import { useIsFetching, useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import { SHIPMENT_LIST_KEY } from "../../shipment-queries";

/**
 * Reloads everything the board shows, and spins while any of it is loading.
 * It watches the fetch count itself: the count moves on every list, summary,
 * briefing and capacity request, and watched from the page it redrew the
 * whole board each time.
 */
export function BoardRefreshButton() {
  const t = useT();
  const queryClient = useQueryClient();
  const refreshing =
    useIsFetching({ queryKey: [SHIPMENT_LIST_KEY] }) +
      useIsFetching({ queryKey: queries.shipmentBoard._def }) >
    0;

  const handleRefresh = useCallback(() => {
    void Promise.all([
      queryClient.invalidateQueries({ queryKey: [SHIPMENT_LIST_KEY] }),
      queryClient.invalidateQueries({ queryKey: queries.shipmentBoard._def }),
      queryClient.invalidateQueries({ queryKey: ["shipment-events"] }),
    ]);
  }, [queryClient]);

  return (
    <Button
      type="button"
      variant="ghost"
      size="icon-sm"
      aria-label={t("Refresh")}
      onClick={handleRefresh}
    >
      <RefreshCw02Icon className={cn("size-3.5", refreshing && "animate-spin")} />
    </Button>
  );
}
