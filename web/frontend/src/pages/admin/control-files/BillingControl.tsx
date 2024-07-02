import AdminLayout from "@/components/admin-page/layout";
import { lazy } from "react";

const BillingControl = lazy(() => import("@/components/billing-control"));

export default function BillingControlPage() {
  return (
    <AdminLayout>
      <BillingControl />
    </AdminLayout>
  );
}
