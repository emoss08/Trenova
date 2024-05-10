import AdminLayout from "@/components/admin-page/layout";
import { lazy } from "react";

const InvoiceControl = lazy(() => import("@/components/invoice-control"));

export default function InvoiceControlPage() {
  return (
    <AdminLayout>
      <InvoiceControl />
    </AdminLayout>
  );
}
