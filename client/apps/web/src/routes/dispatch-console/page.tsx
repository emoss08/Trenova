import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { dispatchConsoleQueries } from "@/lib/queries/dispatch-console";
import type { RoutePrefetch } from "@/lib/route-prefetch";
import { ConsolePageActions } from "./_components/console-page-actions";
import { DispatchConsoleContent } from "./_components/page-content";
import { useDispatchActions } from "./_components/use-dispatch-actions";
import { dispatchBoardInputFromRequest, useDispatchWindow } from "./_components/url-state";

export const prefetch: RoutePrefetch = ({ request }) => [
  dispatchConsoleQueries.board(dispatchBoardInputFromRequest(request)),
];

export function DispatchConsolePage() {
  const t = useT();
  // The page bar's Undo and "Propose coverage" act on the same writes as the queue, so the
  // actions live here and are handed to both.
  const actions = useDispatchActions();
  const { range } = useDispatchWindow();

  return (
    <PageLayout
      fill
      bleed
      pageHeaderProps={{
        title: t("Console"),
        description: t("Cover open moves against available capacity without opening a shipment"),
        actions: (
          <ConsolePageActions
            actions={actions}
            onPropose={() =>
              actions.planAutoAssign({ windowStart: range.start, windowEnd: range.end })
            }
          />
        ),
      }}
    >
      <DispatchConsoleContent actions={actions} />
    </PageLayout>
  );
}
