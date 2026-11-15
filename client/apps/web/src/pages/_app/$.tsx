import { createFileRoute } from "@tanstack/react-router";
import { SectionIndexPage } from "@/routes/section-index/page";

export const Route = createFileRoute("/_app/$")({
  component: SectionIndexPage,
});
