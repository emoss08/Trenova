import AdminLayout from "@/components/admin-page/layout";
import { lazy } from "react";

const FeasibilityControl = lazy(
  () => import("@/components/feasibility-control"),
);

export default function FeasibilityControlPage() {
  return (
    <AdminLayout>
      <FeasibilityControl />
    </AdminLayout>
  );
}
