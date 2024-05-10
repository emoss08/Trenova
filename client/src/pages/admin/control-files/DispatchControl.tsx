import AdminLayout from "@/components/admin-page/layout";
import { lazy } from "react";

const DispatchControl = lazy(() => import("@/components/dispatch-control"));

export default function DispatchControlPage() {
  return (
    <AdminLayout>
      <DispatchControl />
    </AdminLayout>
  );
}
