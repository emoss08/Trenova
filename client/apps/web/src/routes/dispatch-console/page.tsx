import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { dispatchConsoleQueries } from "@/lib/queries/dispatch-console";
import type { RoutePrefetch } from "@/lib/route-prefetch";
import { DispatchConsoleContent } from "./_components/page-content";
import { dispatchBoardInputFromRequest } from "./_components/url-state";

export const prefetch: RoutePrefetch = ({ request }) => [
  dispatchConsoleQueries.board(dispatchBoardInputFromRequest(request)),
];

export function DispatchConsolePage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Console"),
        description: t("Cover open moves against available capacity without opening a shipment"),
      }}
    >
      <DispatchConsoleContent />
    </PageLayout>
  );
}
