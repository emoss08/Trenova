import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { lazy, Suspense } from "react";
import { DESK_RAIL_WIDTH, DESK_WORKSPACE_WIDTH } from "./_components/desk-dimensions";

const DeskLoading = lazy(() =>
  import("@/components/assistant/voice/desk-loading").then((module) => ({
    default: module.DeskLoading,
  })),
);

/**
 * The Desk while it is first opened: the room's shape, with the desk visitor
 * in the conversation column, in place of the app-wide loading card.
 *
 * It is the Desk shell's hydrate fallback, so it shows while the session is
 * checked and the Desk's code arrives, and the shell paints the same three
 * columns when it takes over — the rail, the conversation, the workspace —
 * so nothing jumps when it does. The drawing is loaded on demand so its
 * keyframes stay out of the bundle every other page pays for; until it
 * arrives the column is simply empty, as the shell would be.
 */
export function DeskLoadingScreen() {
  const t = useT();

  return (
    <main
      data-slot="desk-loading-screen"
      className="bg-desk-canvas text-foreground flex h-dvh w-full overflow-hidden"
    >
      <aside
        aria-hidden
        className="bg-desk-rail border-desk-hairline hidden h-full shrink-0 flex-col border-r lg:flex"
        style={{ width: DESK_RAIL_WIDTH }}
      >
        <div className="flex h-12 items-center gap-2 px-3">
          <Skeleton className="size-6" />
          <Skeleton className="h-3.5 w-12" />
        </div>
        <div className="flex flex-col gap-2 px-3 pt-1 pb-3">
          <Skeleton className="h-9" />
          <Skeleton className="h-8" />
        </div>
        <div className="flex flex-col gap-1.5 px-3">
          <Skeleton className="h-6 w-28" />
          <Skeleton className="h-6 w-32" />
        </div>
        <div className="mt-6 flex flex-col gap-1.5 px-3">
          <Skeleton className="mb-1 h-3 w-12" />
          <Skeleton className="h-10" />
          <Skeleton className="h-10" />
          <Skeleton className="h-10" />
        </div>
        <div className="border-desk-hairline mt-auto flex items-center gap-2 border-t px-3 py-2.5">
          <Skeleton className="size-7" />
          <Skeleton className="h-3.5 w-28" />
        </div>
      </aside>

      <div className="flex min-w-0 flex-1 flex-col">
        <div
          aria-hidden
          className="border-desk-hairline flex h-12 shrink-0 items-center gap-2 border-b px-3"
        >
          <Skeleton className="size-7" />
          <Skeleton className="h-3.5 w-40" />
        </div>

        <div className="flex min-h-0 flex-1">
          <div className="bg-desk-column flex min-h-0 min-w-0 flex-1 items-center justify-center">
            <Suspense fallback={null}>
              <DeskLoading label={t("Opening the desk")} />
            </Suspense>
          </div>
          <div
            aria-hidden
            className="border-desk-hairline hidden shrink-0 flex-col gap-3 border-l p-5 lg:flex"
            style={{ width: DESK_WORKSPACE_WIDTH }}
          >
            <Skeleton className="h-9 w-2/3" />
            <Skeleton className="rounded-surface h-40" />
            <Skeleton className="h-4 w-1/2" />
          </div>
        </div>
      </div>
    </main>
  );
}
