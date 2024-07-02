import AdminLayout from "@/components/admin-page/layout";
import { lazy } from "react";

const AccountingControl = lazy(() => import("@/components/accounting-control"));

export default function AccountingControlPage() {
  return (
    <AdminLayout>
      <AccountingControl />
    </AdminLayout>
  );
}
