import { createFileRoute } from "@tanstack/react-router";
import { pageLoader } from "@/lib/route-prefetch";
import { DeskScheduledRoutePage, prefetch } from "@/routes/desk/scheduled-page";

export const Route = createFileRoute("/_desk/desk/scheduled")({
  component: DeskScheduledRoutePage,
  loader: pageLoader(prefetch),
});
