import { ScrollArea, ScrollBar } from "@/components/ui/scroll-area";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { queries } from "@/lib/queries";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { useQuery } from "@tanstack/react-query";
import { HouseIcon, LockIcon, MessageCircleIcon } from "lucide-react";
import { Suspense, useCallback, useState } from "react";
import { ShipmentCommentDetails } from "../comment/comment-details";
import { HoldList } from "../holds/hold-list";
import { ShipmentDetailsSkeleton } from "../shipment-details-skeleton";
import { ShipmentFormBody } from "../shipment-form";
import { ShipmentEditFormWrapper } from "./shipment-edit-form-wrapper";

export function ShipmentEditForm({
  selectedShipment,
  isLoading,
  isError,
}: {
  selectedShipment?: ShipmentSchema | null;
  isLoading?: boolean;
  isError?: boolean;
}) {
  if (isLoading) {
    return <ShipmentDetailsSkeleton />;
  }

  return (
    <Suspense fallback={<ShipmentDetailsSkeleton />}>
      <ShipmentFormBody selectedShipment={selectedShipment} isError={isError}>
        <ShipmentEditTabs selectedShipment={selectedShipment} />
      </ShipmentFormBody>
    </Suspense>
  );
}

function ShipmentEditTabs({
  selectedShipment,
}: {
  selectedShipment?: ShipmentSchema | null;
}) {
  const { data: commentCount } = useQuery({
    ...queries.shipment.getCommentCount(selectedShipment?.id),
    enabled: !!selectedShipment?.id,
  });

  const { data: holds } = useQuery({
    ...queries.shipment.getHolds(selectedShipment?.id),
    enabled: !!selectedShipment?.id,
  });

  const hasHolds = holds && holds.count > 0;

  const [activeTab, setActiveTab] = useState("general-information");

  const handleTabChange = useCallback((value: string) => {
    setActiveTab(value);
  }, []);

  return (
    <Tabs value={activeTab} onValueChange={handleTabChange}>
      <ScrollArea>
        <TabsList className="text-foreground h-auto bg-transparent gap-2 px-2 rounded-none border-b py-1 w-full justify-start overflow-x-auto">
          <TabsTrigger
            value="general-information"
            className="h-7 shrink-0 hover:bg-accent hover:text-foreground data-[state=active]:after:bg-primary data-[state=active]:hover:bg-accent relative after:absolute after:inset-x-0 after:bottom-0 after:-mb-1 after:h-0.5 data-[state=active]:bg-transparent data-[state=active]:shadow-none"
          >
            <HouseIcon
              className="-ms-0.5 mb-0.5 opacity-60"
              size={16}
              aria-hidden="true"
            />
            General Information
          </TabsTrigger>
          <TabsTrigger
            value="comments"
            className="h-7 shrink-0 hover:bg-accent hover:text-foreground text-xs data-[state=active]:after:bg-primary data-[state=active]:hover:bg-accent relative after:absolute after:inset-x-0 after:bottom-0 after:-mb-1 after:h-0.5 data-[state=active]:bg-transparent data-[state=active]:shadow-none"
          >
            <MessageCircleIcon
              className="-ms-0.5 mb-0.5 opacity-60"
              size={16}
              aria-hidden="true"
            />
            Comments
            <span className="max-w-6 bg-primary/15 py-0.5 px-1.5 rounded-sm text-2xs">
              {commentCount?.count && commentCount.count > 99
                ? "99+"
                : (commentCount?.count ?? 0)}
            </span>
          </TabsTrigger>
          {hasHolds && (
            <TabsTrigger
              value="holds"
              className="h-7 shrink-0 hover:bg-accent hover:text-foreground text-xs data-[state=active]:after:bg-primary data-[state=active]:hover:bg-accent relative after:absolute after:inset-x-0 after:bottom-0 after:-mb-1 after:h-0.5 data-[state=active]:bg-transparent data-[state=active]:shadow-none"
            >
              <LockIcon className="-ms-0.5 mb-0.5 opacity-60" size={16} />
              Holds
              <span className="max-w-6 bg-primary/15 py-0.5 px-1.5 rounded-sm text-2xs">
                {holds.count}
              </span>
            </TabsTrigger>
          )}
        </TabsList>
        <ScrollBar orientation="horizontal" />
      </ScrollArea>
      <TabsContent value="general-information">
        <ShipmentEditFormWrapper shipmentId={selectedShipment?.id} />
      </TabsContent>
      <TabsContent value="comments">
        <ShipmentCommentDetails shipmentId={selectedShipment?.id} />
      </TabsContent>
      <TabsContent value="documents">
        <ShipmentCommentDetails shipmentId={selectedShipment?.id} />
      </TabsContent>
      <TabsContent value="holds">
        <HoldList holds={holds?.results ?? []} />
      </TabsContent>
    </Tabs>
  );
}
