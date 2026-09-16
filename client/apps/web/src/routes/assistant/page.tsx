import { useT } from "@trenova/shared/i18n/use-t";
import { AssistantWorkspace } from "./_components/assistant-workspace";

export function AssistantPage() {
  const t = useT();

  return (
    <div className="flex h-[calc(100vh-var(--header-height,3.5rem))] flex-col">
      <div className="border-border border-b px-4 py-3">
        <h1 className="text-lg font-semibold">{t("Assistant")}</h1>
        <p className="text-muted-foreground text-sm">
          {t("Ask about shipments, dispatch, billing, and how to get things done in Trenova.")}
        </p>
      </div>
      <AssistantWorkspace />
    </div>
  );
}
