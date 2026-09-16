import { useT } from "@trenova/shared/i18n/use-t";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { AgentList } from "./_components/agent-list";

export function AgentsPage() {
  const t = useT();

  return (
    <AdminPageLayout>
      <PageHeader
        title={t("Agents")}
        description={t(
          "Build assistants from Trenova's templates. You choose which tools each one may use and how much it may do on its own; the template decides what it is allowed to reach at all.",
        )}
      />
      <div className="p-4">
        <AgentList />
      </div>
    </AdminPageLayout>
  );
}
