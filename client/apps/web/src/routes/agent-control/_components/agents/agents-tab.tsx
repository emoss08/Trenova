import { useT } from "@trenova/shared/i18n/use-t";
import { AgentList } from "./agent-list";

export default function AgentsTab() {
  const t = useT();

  return (
    <section className="flex flex-col gap-3">
      <div>
        <h2 className="text-base font-semibold">{t("Agents")}</h2>
        <p className="text-muted-foreground max-w-prose text-sm">
          {t(
            "Each agent carries its own instructions, the tools it may call and how much it may do on its own. Start from a template or write one from scratch.",
          )}
        </p>
      </div>
      <AgentList />
    </section>
  );
}
