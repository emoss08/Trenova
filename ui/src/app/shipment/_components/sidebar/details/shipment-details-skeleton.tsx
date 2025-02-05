import { ComponentLoader } from "@/components/ui/component-loader";

export function ShipmentDetailsSkeleton() {
  return (
    <div className="flex items-center justify-center h-full">
      <ComponentLoader message="Loading shipment details..." />
    </div>
  );
}
