import { useT } from "@trenova/shared/i18n/use-t";
import { lazy, Suspense } from "react";

const DeskLoading = lazy(() =>
  import("@/components/assistant/voice/desk-loading").then((module) => ({
    default: module.DeskLoading,
  })),
);

/**
 * The Desk while it is first opened: the desk visitor on the Desk's own
 * canvas, in place of the app-wide loading card.
 *
 * It is the Desk shell's hydrate fallback, so it shows while the session is
 * checked and the Desk's code arrives, and the shell paints the same canvas
 * when it takes over. The drawing is loaded on demand so its keyframes stay
 * out of the bundle every other page pays for; until it arrives the canvas is
 * simply empty, as the shell would be.
 */
export function DeskLoadingScreen() {
  const t = useT();

  return (
    <main
      data-slot="desk-loading-screen"
      className="bg-desk-canvas text-foreground flex h-dvh w-full items-center justify-center"
    >
      <Suspense fallback={null}>
        <DeskLoading label={t("Opening the desk")} />
      </Suspense>
    </main>
  );
}
