import { createFileRoute } from "@tanstack/react-router";
import { CapturePairPage } from "@/routes/capture/pair/page";

export const Route = createFileRoute("/_app/capture/pair")({
  component: CapturePairPage,
});
