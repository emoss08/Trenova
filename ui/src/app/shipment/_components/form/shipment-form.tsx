/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */
import { ScrollArea, ScrollBar } from "@/components/ui/scroll-area";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { queries } from "@/lib/queries";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { useQuery } from "@tanstack/react-query";
import {
  HouseIcon,
  LockIcon,
  MessageCircleIcon,
  PanelsTopLeftIcon,
} from "lucide-react";
import { Suspense, useCallback, useState } from "react";
import { ShipmentNotFoundOverlay } from "../sidebar/shipment-not-found-overlay";
import { ShipmentCommentDetails } from "./comment/comment-details";
import { HoldList } from "./holds/hold-list";
import { ShipmentDetailsSkeleton } from "./shipment-details-skeleton";
import { ShipmentEditFormWrapper } from "./shipment-edit-form-wrapper";
import { ShipmentFormContent } from "./shipment-form-body";
import { ShipmentFormHeader } from "./shipment-form-header";
import { ShipmentGeneralInfoForm } from "./shipment-general-info-form";

export function ShipmentCreateForm() {
  return (
    <Suspense fallback={<ShipmentDetailsSkeleton />}>
      <ShipmentFormBody>
        <ShipmentGeneralInfoForm className="max-h-[calc(100vh-7rem)]" />
      </ShipmentFormBody>
    </Suspense>
  );
}

// For edit mode - manages its own form
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
        <ShipmentEditTabs
          shipmentId={selectedShipment?.id}
          selectedShipment={selectedShipment}
        />
      </ShipmentFormBody>
    </Suspense>
  );
}

// Tabs for edit mode - uses ShipmentEditFormWrapper
function ShipmentEditTabs({
  shipmentId,
  selectedShipment,
}: {
  shipmentId: ShipmentSchema["id"];
  selectedShipment?: ShipmentSchema | null;
}) {
  const { data: commentCount } = useQuery({
    ...queries.shipment.getCommentCount(shipmentId),
    enabled: !!shipmentId,
  });

  const { data: holds, isLoading: isHoldsLoading } = useQuery({
    ...queries.shipment.getHolds(shipmentId),
    enabled: !!shipmentId,
  });

  const hasHolds = holds && holds.count > 0;

  console.info(hasHolds);

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
          <TabsTrigger
            value="documents"
            className="h-7 shrink-0 hover:bg-accent hover:text-foreground text-xs data-[state=active]:after:bg-primary data-[state=active]:hover:bg-accent relative after:absolute after:inset-x-0 after:bottom-0 after:-mb-1 after:h-0.5 data-[state=active]:bg-transparent data-[state=active]:shadow-none"
          >
            <PanelsTopLeftIcon
              className="-ms-0.5 mb-0.5 opacity-60"
              size={16}
              aria-hidden="true"
            />
            Documents
            <span className="max-w-6 bg-primary/15 py-0.5 px-1.5 rounded-sm text-2xs">
              3
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
        <ShipmentEditFormWrapper currentRecord={selectedShipment} />
      </TabsContent>
      <TabsContent value="comments">
        <ShipmentCommentDetails shipmentId={shipmentId} />
      </TabsContent>
      <TabsContent value="documents">
        <ShipmentCommentDetails shipmentId={shipmentId} />
      </TabsContent>
      <TabsContent value="holds">
        <HoldList holds={holds?.results ?? []} />
      </TabsContent>
    </Tabs>
  );
}

export function ShipmentFormBody({
  selectedShipment,
  isError,
  children,
}: {
  selectedShipment?: ShipmentSchema | null;
  isError?: boolean;
  children: React.ReactNode;
}) {
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
