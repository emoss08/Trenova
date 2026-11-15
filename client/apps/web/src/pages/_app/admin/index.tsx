import { createFileRoute } from "@tanstack/react-router";
import { SectionIndexPage } from "@/routes/section-index/page";

// Settings at /admin draws only its frame; the section index fills it.
export const Route = createFileRoute("/_app/admin/")({
  component: SectionIndexPage,
});
