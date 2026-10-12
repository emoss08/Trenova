import { useT } from "@trenova/shared/i18n/use-t";
import { translate } from "@trenova/shared/i18n/runtime";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { fuelCardProviderChoices } from "@/lib/choices";
import { FUEL_FEED_RUN_LIST_KEY } from "@/lib/graphql/fuel-purchase-import";
import {
  syncFuelCardFeed,
  UNASSIGNED_FUEL_CARD_LIST_KEY,
  type FuelCardSyncResult,
} from "@/lib/graphql/fuel-card";
import { useQueryClient } from "@tanstack/react-query";
import type { FuelCardProvider } from "@trenova/graphql/generated/graphql";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { RefreshCw02Icon } from "@trenova/shared/components/icons";
import { toast } from "sonner";

/**
 * Reads a connected feed now instead of waiting for the hourly schedule. Useful
 * right after connecting one, and after fixing whatever a run was held up on.
 *
 * Only the fleet networks are offered: a provider with no connection simply
 * returns "not connected", which is a clearer answer than hiding the option and
 * leaving somebody wondering where it went.
 */
export function SyncFeedButton() {
  const t = useT();

  const queryClient = useQueryClient();

  const { mutateAsync, isPending } = useApiMutation<FuelCardSyncResult, FuelCardProvider>({
    resourceName: "Fuel card feed",
    mutationFn: (provider) => syncFuelCardFeed(provider),
    onSuccess: async (result) => {
      toast.success(describeSync(result), {
        description:
          result.queued > 0
            ? t(
                "{0, plural, one {# row is} other {# rows are}} waiting on a card assignment or a missing unit.",
                result.queued,
              )
            : undefined,
      });
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: [FUEL_FEED_RUN_LIST_KEY],
          refetchType: "all",
        }),
        queryClient.invalidateQueries({
          queryKey: [UNASSIGNED_FUEL_CARD_LIST_KEY],
          refetchType: "all",
        }),
      ]);
    },
  });

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button variant="outline" size="sm" isLoading={isPending} loadingText={t("Reading...")}>
            <RefreshCw02Icon className="size-4" />
            {t("Sync now")}
          </Button>
        }
      />
      <DropdownMenuContent align="end">
        {fuelCardProviderChoices
          .filter((choice) => choice.value !== "Other")
          .map((choice) => (
            <DropdownMenuItem
              key={choice.value}
              title={t(choice.label)}
              onClick={() => mutateAsync(choice.value as FuelCardProvider)}
            />
          ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function describeSync(result: FuelCardSyncResult): string {
  if (result.fetched === 0) {
    return translate("Nothing new from {0}", result.provider);
  }

  return translate(
    "{0}: {1} posted of {2} read",
    result.provider,
    result.committed,
    result.fetched,
  );
}
