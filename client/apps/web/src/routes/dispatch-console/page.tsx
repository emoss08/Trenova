import { PageLayout } from "@/components/navigation/sidebar-layout";
import { dispatchConsoleQueries } from "@/lib/queries/dispatch-console";
import type { RoutePrefetch } from "@/lib/route-prefetch";
import { DispatchConsoleContent } from "./_components/page-content";
import { dispatchBoardInputFromRequest } from "./_components/url-state";

export const prefetch: RoutePrefetch = ({ request }) => [
  dispatchConsoleQueries.board(dispatchBoardInputFromRequest(request)),
];

export function DispatchConsolePage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Console",
        description: "Cover open moves against available capacity without opening a shipment",
      }}
    >
      <DispatchConsoleContent />
    </PageLayout>
  );
}
