import { createFileRoute } from "@tanstack/react-router";
import { pageLoader } from "@/lib/route-prefetch";
import { SettingsPage, prefetch } from "@/routes/settings/page";

// Settings is a room: the module rail stays, and its sections take the panel's
// place and its fold.
export const Route = createFileRoute("/_app/settings")({
  staticData: { frame: "room" },
  component: SettingsPage,
  loader: pageLoader(prefetch),
});
