import { createFileRoute } from "@tanstack/react-router";
import { AdminLayout } from "@/routes/admin-layout";

export const Route = createFileRoute("/_app/admin")({
  component: AdminLayout,
});
