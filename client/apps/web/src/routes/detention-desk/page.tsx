import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DetentionDesk } from "./_components/detention-desk";

export function DetentionDeskPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Detention Desk",
        description:
          "Live free-time clocks, notice deadlines, and accruing detention across every driver currently on a dock",
      }}
    >
      <DetentionDesk />
    </PageLayout>
  );
}
