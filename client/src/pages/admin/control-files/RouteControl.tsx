import AdminLayout from "@/components/admin-page/layout";
import { lazy } from "react";

const RouteControl = lazy(() => import("@/components/route-control"));

export default function RouteControlPage() {
  return (
    <AdminLayout>
      <RouteControl />
    </AdminLayout>
  );
}
