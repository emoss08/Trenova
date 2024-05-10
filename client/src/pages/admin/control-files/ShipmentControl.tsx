import AdminLayout from "@/components/admin-page/layout";
import { lazy } from "react";

const ShipmentControl = lazy(() => import("@/components/shipment-control"));

export default function ShipmentControlPage() {
  return (
    <AdminLayout>
      <ShipmentControl />
    </AdminLayout>
  );
}
