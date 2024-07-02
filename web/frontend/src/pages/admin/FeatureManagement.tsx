import AdminLayout from "@/components/admin-page/layout";
import { lazy } from "react";

const FeatureList = lazy(() => import("@/components/feature-list"));

export default function FeatureManagementPage() {
  return (
    <AdminLayout>
      <FeatureList />
    </AdminLayout>
  );
}
