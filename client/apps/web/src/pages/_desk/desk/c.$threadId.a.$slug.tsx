import { createFileRoute } from "@tanstack/react-router";
import { DeskArtifactLinkPage } from "@/routes/desk/artifact-link-page";

// A link to one artifact, named by its slug: the conversation
// opens with it beside it.
export const Route = createFileRoute("/_desk/desk/c/$threadId/a/$slug")({
  component: DeskArtifactLinkPage,
});
