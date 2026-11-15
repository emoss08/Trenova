import { createFileRoute } from "@tanstack/react-router";
import { DeskHomePage } from "@/routes/desk/home-page";

export const Route = createFileRoute("/_desk/desk/")({
  component: DeskHomePage,
});
