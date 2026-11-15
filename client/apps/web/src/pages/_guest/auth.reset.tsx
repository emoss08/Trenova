import { createFileRoute } from "@tanstack/react-router";
import { ResetPasswordPage } from "@/routes/auth/reset-page";

// Opened from the emailed reset link. Guest-only, like the sign-in page:
// somebody already signed in has no use for it.
export const Route = createFileRoute("/_guest/auth/reset")({
  component: ResetPasswordPage,
});
