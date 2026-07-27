import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DetentionIntelligence } from "./_components/detention-intelligence";

export function DetentionIntelligencePage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Detention Intelligence",
        description:
          "Which facilities cost the most, which customers are unprofitable once driver pay is netted off, and where discretionary revenue is going",
      }}
    >
      <DetentionIntelligence />
    </PageLayout>
  );
}
