import AdminLayout from "@/components/admin-page/layout";
import { lazy } from "react";

const EmailControl = lazy(() => import("@/components/email-control"));

export default function EmailControlPage() {
  return (
    <AdminLayout>
      <EmailControl />
    </AdminLayout>
  );
}
