import { ShipmentStatusBadge } from "@/components/status-badge";
import { Separator } from "@/components/ui/separator";
import { AdvancedMarker } from "@vis.gl/react-google-maps";
import { Activity, Building2, Clock, MapPin, Package, Truck, User, X } from "lucide-react";

import { ELD_STATUS_LABELS, type MockTractor } from "./mock-data";

function InfoRow({
  icon: Icon,
  label,
  value,
}: {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  value: string;
}) {
  return (
    <div className="flex items-center justify-between gap-3">
      <div className="flex items-center gap-1.5 text-muted-foreground">
        <Icon className="size-3 shrink-0" />
        <span>{label}</span>
      </div>
      <span className="truncate font-medium text-foreground">{value}</span>
    </div>
  );
}

export function TractorInfoWindow({
  tractor,
  onClose,
}: {
  tractor: MockTractor;
  onClose: () => void;
}) {
  return (
    <AdvancedMarker
      position={{ lat: tractor.lat, lng: tractor.lng }}
      zIndex={200}
      onClick={(e) => e.stop()}
    >
      <div className="relative mb-6">
        <div className="relative z-10 flex w-60 flex-col gap-2 rounded-lg border bg-popover p-3 text-xs text-popover-foreground shadow-md ring-1 ring-foreground/10">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-1.5">
              <Truck className="size-3.5 text-foreground" />
              <span className="text-sm font-semibold text-foreground">{tractor.unitNumber}</span>
            </div>
            <div className="flex items-center gap-1.5">
              <ShipmentStatusBadge status={tractor.shipmentStatus} />
              <button
                type="button"
                onClick={onClose}
                className="rounded-sm p-0.5 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
              >
                <X className="size-3.5" />
              </button>
            </div>
          </div>

          <Separator />

          <div className="flex flex-col gap-1.5">
            <InfoRow icon={User} label="Driver" value={tractor.driverName} />
            <InfoRow icon={Package} label="Load" value={tractor.currentLoad} />
            <InfoRow icon={Building2} label="Customer" value={tractor.customerName} />
            <InfoRow icon={MapPin} label="Next" value={tractor.nextStop} />
          </div>

          <Separator />

          <div className="flex flex-col gap-1.5">
            <InfoRow icon={Activity} label="Status" value={ELD_STATUS_LABELS[tractor.eldStatus]} />
            <InfoRow icon={Clock} label="ETA" value={tractor.eta} />
            <InfoRow icon={Clock} label="Drive Rem." value={tractor.hosRemaining} />
          </div>
        </div>

        <div className="absolute -bottom-1 left-1/2 size-2.5 -translate-x-1/2 rotate-45 border-r border-b border-foreground/10 bg-popover" />
      </div>
    </AdvancedMarker>
  );
}
