import { createFileRoute } from "@tanstack/react-router";
import { pageLoader } from "@/lib/route-prefetch";
import { CaptureDevicesPage, prefetch } from "@/routes/capture/devices/page";

export const Route = createFileRoute("/_app/capture/devices")({
  component: CaptureDevicesPage,
  loader: pageLoader(prefetch),
});
