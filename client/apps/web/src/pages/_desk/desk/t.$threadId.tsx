import { createFileRoute } from "@tanstack/react-router";
import { DeskConversationPage } from "@/routes/desk/conversation-page";

export const Route = createFileRoute("/_desk/desk/t/$threadId")({
  component: DeskConversationPage,
});
