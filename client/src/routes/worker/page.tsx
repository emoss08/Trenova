import { PageLayout } from "@/components/navigation/sidebar-layout";
import WorkersContent from "./_components/page-content";
import { PTOContent } from "./_components/pto-content";

export function WorkersPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Workers",
        description: "Manage and track workers along with their compliance and paid time off",
      }}
    >
      <PTOContent />
      <WorkersContent />
    </PageLayout>
  );
}
