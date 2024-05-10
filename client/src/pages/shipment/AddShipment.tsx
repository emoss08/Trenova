import { Skeleton } from "@/components/ui/skeleton";
import { Suspense, lazy } from "react";

const AddShipment = lazy(
  () => import("@/components/shipment-management/add-shipment"),
);

export default function AddShipmentPage() {
  return (
    <Suspense fallback={<Skeleton className="size-full" />}>
      <AddShipment />
    </Suspense>
  );
}
