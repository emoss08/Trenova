/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { billingClientSearchParams } from "@/hooks/use-billing-client-state";
import { queries } from "@/lib/queries";
import { cn } from "@/lib/utils";
import { AnalyticsPage } from "@/types/analytics";
import { faChevronRight } from "@fortawesome/pro-solid-svg-icons";
import { useSuspenseQuery } from "@tanstack/react-query";
import { useQueryStates } from "nuqs";
import { TransferDialog } from "./transfer-to-billing/transfer-dialog";

export default function BillingQueueAnalytics() {
  const { data: analytics } = useSuspenseQuery({
    ...queries.analytics.getAnalytics(AnalyticsPage.BillingClient),
  });

  return (
    <dl className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
      <ShipmentReadyToBillCard
        count={analytics?.shipmentReadyBillCard?.count}
      />
    </dl>
  );
}

function ShipmentReadyToBillCard({ count }: { count: number }) {
  const [searchParams, setSearchParams] = useQueryStates(
    billingClientSearchParams,
  );

  return (
    <>
      <div
        className={cn(
          "p-4 rounded-lg bg-gradient-to-r from-5% via-foreground via-50% to-foreground to-90%",
          count > 0 ? "from-blue-500" : "from-green-500",
        )}
      >
        <div className="p-0">
          <div className="flex flex-col pb-2">
            <dt className="font-medium text-xl text-background">
              Shipments Awaiting Transfer
            </dt>
            <dd className="text-2xs text-background/80">
              Count of shipments that are ready to be transferred to billing
            </dd>
          </div>
          <div className="flex items-center justify-between">
            <dd className="text-2xl font-bold text-background">{count}</dd>
            {count > 0 && (
              <Button
                className="[&_svg]:size-2"
                size="sm"
                onClick={() => setSearchParams({ transferModalOpen: true })}
              >
                Transfer to billing
                <Icon icon={faChevronRight} className="size-3" />
              </Button>
            )}
          </div>
        </div>
      </div>
      <TransferDialog
        open={searchParams.transferModalOpen}
        onOpenChange={() =>
          setSearchParams({
            transferModalOpen: !searchParams.transferModalOpen,
          })
        }
      />
    </>
  );
}
