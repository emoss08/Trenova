import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DeskHeaderActions, DeskLivePulse } from "./_components/desk-header";
import { DetentionDesk } from "./_components/detention-desk";
import { useDetentionDesk } from "./_components/use-detention-desk";

export function DetentionDeskPage() {
  // One query and one clock for the whole screen, so the header, the exposure
  // strip, and the board can never disagree about what is on the floor.
  const desk = useDetentionDesk();

  return (
    <PageLayout
      className="p-0 gap-y-0"
      pageHeaderProps={{
        title: "Detention Desk",
        description:
          "Live free-time clocks, notice deadlines, and accruing detention across every driver currently on a dock",
        context: <DeskLivePulse desk={desk} />,
        actions: <DeskHeaderActions desk={desk} />,
      }}
    >
      <DetentionDesk desk={desk} />
    </PageLayout>
  );
}
