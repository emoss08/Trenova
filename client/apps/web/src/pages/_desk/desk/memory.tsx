import { createFileRoute } from "@tanstack/react-router";
import { pageLoader } from "@/lib/route-prefetch";
import { DeskMemoryRoutePage, prefetch } from "@/routes/desk/memory-page";

export const Route = createFileRoute("/_desk/desk/memory")({
  component: DeskMemoryRoutePage,
  loader: pageLoader(prefetch),
});
