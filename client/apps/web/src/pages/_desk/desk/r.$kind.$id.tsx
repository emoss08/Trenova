import { createFileRoute } from "@tanstack/react-router";
import { DeskCasePage } from "@/routes/desk/case-page";

export const Route = createFileRoute("/_desk/desk/r/$kind/$id")({
  component: DeskCasePage,
});
