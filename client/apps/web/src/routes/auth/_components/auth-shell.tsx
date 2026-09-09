import type { ReactNode } from "react";
import { AuthPanel, type CredentialReceipt } from "./auth-panel";

// The step the flow is on. It drives the crumb counter, the aura position on the
// panel and which receipt rows have been filled in.
// "forgot" sits off the main path and gets no aura rule of its own, so the panel stays
// where the login step left it.
export type AuthStep = "login" | "forgot" | "org" | "role" | "done";

/**
 * Two-pane sign-in frame: an ambient panel on the left that shows the session
 * credential assembling itself, and a single morphing card on the right.
 *
 * The `minmax(0,1fr)` track, the `min(400px,100%)` column and `scrollbar-gutter:stable`
 * are load-bearing together — an auto-sized track grows a horizontal scrollbar the
 * moment the tallest step makes the pane scroll vertically.
 */
export function AuthShell({
  step,
  receipt,
  children,
}: {
  step: AuthStep;
  receipt: CredentialReceipt;
  children: ReactNode;
}) {
  return (
    <div
      data-auth-step={step}
      className="bg-background font-geist text-foreground fixed inset-0 grid h-svh w-full grid-cols-1 overflow-hidden text-[13.5px] tracking-[-0.006em] antialiased min-[900px]:grid-cols-[1fr_clamp(440px,42%,560px)]"
    >
      <AuthPanel receipt={receipt} />
      <main className="relative grid min-w-0 grid-cols-[minmax(0,1fr)] items-center justify-items-center overflow-auto px-6 py-10 [scrollbar-gutter:stable]">
        <div className="flex w-[min(400px,100%)] max-w-full flex-col gap-4">{children}</div>
      </main>
    </div>
  );
}
