import { createFileRoute } from "@tanstack/react-router";
import { DeskViewPage } from "@/routes/desk/view-page";

// What an agent laid out, on the canvas, with the conversation
// that made it in the margin.
export const Route = createFileRoute("/_desk/desk/v/$viewThreadId/$slug")({
  component: DeskViewPage,
});
