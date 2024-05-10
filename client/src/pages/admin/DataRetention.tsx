import AdminLayout from "@/components/admin-page/layout";
import { lazy } from "react";

const DataRetention = lazy(() => import("@/components/data-retention"));

export default function DataRetentionPage() {
  return (
    <AdminLayout>
      <DataRetention />
    </AdminLayout>
  );
}
