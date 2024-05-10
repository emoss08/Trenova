import AdminLayout from "@/components/admin-page/layout";
import { lazy } from "react";

const GoogleAPI = lazy(() => import("@/components/google-api"));

export default function GoogleAPIPage() {
  return (
    <AdminLayout>
      <GoogleAPI />
    </AdminLayout>
  );
}
