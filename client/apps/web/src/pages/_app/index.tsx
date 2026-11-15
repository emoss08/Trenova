import { createFileRoute } from "@tanstack/react-router";
import { pageLoader } from "@/lib/route-prefetch";
import { Home, prefetch } from "@/routes/home/page";

export const Route = createFileRoute("/_app/")({
  component: Home,
  loader: pageLoader(prefetch),
});
