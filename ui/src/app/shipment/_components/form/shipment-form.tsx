/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Badge } from "@/components/ui/badge";
import {
  ScrollArea,
  ScrollAreaShadow,
  ScrollBar,
} from "@/components/ui/scroll-area";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useUrlFragment } from "@/hooks/use-url-fragment";
import { queries } from "@/lib/queries";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { useQuery } from "@tanstack/react-query";
import { HouseIcon, MessageCircleIcon, PanelsTopLeftIcon } from "lucide-react";
import { lazy, Suspense, useCallback, useEffect, useState } from "react";
import { ShipmentNotFoundOverlay } from "../sidebar/shipment-not-found-overlay";
import { ShipmentDetailsSkeleton } from "./shipment-details-skeleton";
import { ShipmentFormContent } from "./shipment-form-body";
import { ShipmentFormHeader } from "./shipment-form-header";

// Lazy loaded components
const ShipmentBillingDetails = lazy(
  () => import("./billing-details/shipment-billing-details"),
);
const ShipmentGeneralInformation = lazy(
  () => import("./shipment-general-information"),
);
const ShipmentCommodityDetails = lazy(
  () => import("./commodity/commodity-details"),
);
const ShipmentMovesDetails = lazy(() => import("./move/move-details"));
const ShipmentServiceDetails = lazy(
  () => import("./service-details/shipment-service-details"),
);
const ShipmentCommentDetails = lazy(() => import("./comment/comment-details"));

type ShipmentDetailsProps = {
  selectedShipment?: ShipmentSchema | null;
  isLoading?: boolean;
  isError?: boolean;
};

export function ShipmentForm({
  selectedShipment,
  isLoading,
  isError,
}: ShipmentDetailsProps) {
  if (isLoading) {
    return <ShipmentDetailsSkeleton />;
  }

  return (
    <Suspense fallback={<ShipmentDetailsSkeleton />}>
      <ShipmentFormBody selectedShipment={selectedShipment} isError={isError}>
        <ShipmentSectionTabs shipmentId={selectedShipment?.id} />
      </ShipmentFormBody>
    </Suspense>
  );
}

function ShipmentSectionTabs({
  shipmentId,
}: {
  shipmentId: ShipmentSchema["id"];
}) {
  const { fragment, setFragment } = useUrlFragment();

  const { data: commentCount } = useQuery({
    ...queries.shipment.getCommentCount(shipmentId),
  });

  const [activeTab, setActiveTab] = useState(() => {
    return fragment === "comments" ? "comments" : "general-information";
  });

  useEffect(() => {
    const validTab =
      fragment === "comments" ? "comments" : "general-information";
    setActiveTab(validTab);
  }, [fragment]);

  const handleTabChange = useCallback(
    (value: string) => {
      setActiveTab(value);
      setFragment(value);
    },
    [setFragment],
  );

  return (
    <Tabs value={activeTab} onValueChange={handleTabChange}>
      <ScrollArea className="w-full">
        <TabsList className="text-foreground mb-3 h-auto gap-2 px-2 rounded-none border-b bg-transparent py-1 w-full justify-start overflow-x-auto">
          <TabsTrigger
            value="general-information"
            className="hover:bg-accent shrink-0 hover:text-foreground text-xs data-[state=active]:after:bg-primary data-[state=active]:hover:bg-accent relative after:absolute after:inset-x-0 after:bottom-0 after:-mb-1 after:h-0.5 data-[state=active]:bg-transparent data-[state=active]:shadow-none"
          >
            <HouseIcon
              className="-ms-0.5 me-1.5 opacity-60"
              size={16}
              aria-hidden="true"
            />
            General Information
          </TabsTrigger>
          <TabsTrigger
            value="comments"
            className="hover:bg-accent hover:text-foreground text-xs data-[state=active]:after:bg-primary data-[state=active]:hover:bg-accent relative after:absolute after:inset-x-0 after:bottom-0 after:-mb-1 after:h-0.5 data-[state=active]:bg-transparent data-[state=active]:shadow-none"
          >
            <MessageCircleIcon
              className="-ms-0.5 me-1.5 opacity-60"
              size={16}
              aria-hidden="true"
            />
            Comments
            <Badge
              withDot={false}
              className="bg-primary/15 ms-1.5 min-w-5"
              variant="secondary"
            >
              {commentCount?.count ?? 0}
            </Badge>
          </TabsTrigger>
          <TabsTrigger
            value="documents"
            className="hover:bg-accent hover:text-foreground text-xs data-[state=active]:after:bg-primary data-[state=active]:hover:bg-accent relative after:absolute after:inset-x-0 after:bottom-0 after:-mb-1 after:h-0.5 data-[state=active]:bg-transparent data-[state=active]:shadow-none"
          >
            <PanelsTopLeftIcon
              className="-ms-0.5 me-1.5 opacity-60"
              size={16}
              aria-hidden="true"
            />
            Documents
            <Badge
              withDot={false}
              className="bg-primary/15 ms-1.5 min-w-5"
              variant="secondary"
            >
              3
            </Badge>
          </TabsTrigger>
        </TabsList>
        <ScrollBar orientation="horizontal" />
      </ScrollArea>
      <ScrollArea className="flex flex-col overflow-y-auto px-4 max-h-[calc(100vh-12rem)]">
        <TabsContent className="pb-16" value="general-information">
          <ShipmentServiceDetails />
          <ShipmentBillingDetails />
          <ShipmentGeneralInformation />
          <ShipmentCommodityDetails />
          <ShipmentMovesDetails />
        </TabsContent>
        <TabsContent value="comments">
          <ShipmentCommentDetails />
        </TabsContent>
        <TabsContent value="documents">
          <ShipmentCommentDetails />
        </TabsContent>
        <ScrollAreaShadow />
      </ScrollArea>
    </Tabs>
  );
}

export function ShipmentFormBody({
  selectedShipment,
  isError,
  children,
}: Omit<ShipmentDetailsProps, "isLoading"> & { children: React.ReactNode }) {
  // Handle error state
  if (isError) {
    return (
      <div className="flex size-full items-center justify-center">
        <ShipmentNotFoundOverlay />
      </div>
    );
  }

  return (
    <ShipmentFormBodyOuter>
      <ShipmentFormHeader selectedShipment={selectedShipment} />
      <ShipmentFormContent selectedShipment={selectedShipment}>
        {children}
      </ShipmentFormContent>
    </ShipmentFormBodyOuter>
  );
}

function ShipmentFormBodyOuter({ children }: { children: React.ReactNode }) {
  return <div className="size-full">{children}</div>;
}
