import { useT } from "@trenova/shared/i18n/use-t";
import { PageHeader } from "@/components/page-header";
import { AssistantWorkspace } from "./_components/assistant-workspace";

export function AssistantPage() {
  const t = useT();

  return (
    <div className="flex h-[calc(100vh-var(--header-height,3.5rem))] flex-col">
      <PageHeader
        title={t("Assistant")}
        description={t(
          "Ask about shipments, dispatch, billing, and how to get things done in Trenova.",
        )}
      />
      <AssistantWorkspace />
    </div>
  );
}
