import { useShipments } from "@/app/shipment/queries/shipment";
import { HoverCardTimestamp } from "@/components/data-table/_components/data-table-components";
import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { HeaderBackButton } from "@/components/ui/sheet-header-components";
import { VisuallyHidden } from "@/components/ui/visually-hidden";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import type { TableSheetProps } from "@/types/data-table";
import { ShipmentStatus } from "@/types/shipment";
import {
  faBox,
  faChevronDown,
  faChevronUp,
} from "@fortawesome/pro-regular-svg-icons";
import { useState } from "react";

export function TransferDialog({ ...props }: TableSheetProps) {
  const { open, onOpenChange } = props;

  return (
    <Sheet {...props}>
      <SheetContent withClose={false} className="w-[600px] sm:max-w-[640px]">
        <VisuallyHidden>
          <SheetHeader>
            <SheetTitle>Transfer Shipments</SheetTitle>
            <SheetDescription>
              Transfer the selected shipments to the billing team
            </SheetDescription>
          </SheetHeader>
        </VisuallyHidden>
        <SheetBody>
          <div className="flex items-center p-4 justify-between">
            <HeaderBackButton onBack={() => onOpenChange?.(false)} />
          </div>
          <TransferDialogContent open={open} />
        </SheetBody>
      </SheetContent>
    </Sheet>
  );
}

function TransferDialogContent({ open }: { open: boolean }) {
  const { data, isLoading, isError } = useShipments({
    status: ShipmentStatus.ReadyToBill,
    pageIndex: 0,
    pageSize: 10,
    expandShipmentDetails: true,
    enabled: open,
  });

  console.info("data", data);

  return (
    <div className="flex flex-col gap-2 border-t border-border pt-4">
      <div className="flex items-center justify-between text-xl font-medium">
        <p className="text-muted-foreground">Total Shipments:</p>
        <span className="text-xl font-semibold">{data?.count ?? 0}</span>
      </div>
      {data?.results && (
        <div className="flex flex-col gap-2">
          {data?.results.map((shipment) => (
            <ShipmentCard key={shipment.id} shipment={shipment} />
          ))}
        </div>
      )}
    </div>
  );
}

function ShipmentCard({ shipment }: { shipment: ShipmentSchema }) {
  const [detailsOpen, setDetailsOpen] = useState<boolean>(false);

  return (
    <div className="bg-card border border-border rounded-lg p-4">
      <div className="flex items-center gap-2 pb-4">
        <div className="bg-muted rounded-lg p-4 flex items-center justify-center size-12">
          <Icon icon={faBox} className="size-6" />
        </div>
        <div className="flex items-center justify-between w-full">
          <div className="flex flex-col">
            <p className="text-sm font-medium text-muted-foreground">
              Shipment ID
            </p>
            <p className="text-xl font-semibold">
              {shipment?.proNumber || "-"}
            </p>
          </div>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setDetailsOpen(!detailsOpen)}
          >
            <Icon icon={detailsOpen ? faChevronUp : faChevronDown} />
          </Button>
        </div>
      </div>
      {detailsOpen && (
        <div className="flex flex-col gap-2 border-t border-border pt-4">
          <ShipmentDetailSectionItem
            label="Customer"
            value={shipment?.customer?.name ?? "-"}
          />
          <ShipmentDetailSectionItem
            label="Pro #"
            value={shipment?.proNumber ?? "-"}
          />
          <ShipmentDetailSectionItem label="BOL" value={shipment?.bol ?? "-"} />
          <ShipmentDetailSectionItem
            label="Actual Ship Date"
            value={shipment?.actualShipDate ?? "-"}
          />
          <ShipmentDetailSectionItem
            label="Actual Delivery Date"
            value={shipment?.actualDeliveryDate ?? "-"}
          />
        </div>
      )}
    </div>
  );
}

function ShipmentDetailSectionItem({
  label,
  value,
}: {
  label: string;
  value: string | number;
}) {
  const valueType = typeof value;

  const valueComponent =
    valueType === "string" ? (
      <p className="text-sm font-medium">{value}</p>
    ) : (
      <HoverCardTimestamp
        align="start"
        side="left"
        className="text-sm"
        timestamp={value as number}
      />
    );

  return (
    <div className="flex items-center justify-between">
      <p className="text-sm text-muted-foreground">{label}</p>
      {valueComponent}
    </div>
  );
}
