import { createFileRoute } from "@tanstack/react-router";
import { AuthPage } from "@/routes/auth/page";

export const Route = createFileRoute("/_guest/login_/$orgSlug")({
  component: AuthPage,
});
