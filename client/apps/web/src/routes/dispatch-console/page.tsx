import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DispatchConsoleContent } from "./_components/page-content";

export function DispatchConsolePage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Dispatcher Console",
        description:
          "Cover open moves against available capacity without opening a shipment",
      }}
    >
      <DispatchConsoleContent />
    </PageLayout>
  );
}
