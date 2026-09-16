import { useT } from "@trenova/shared/i18n/use-t";
import { SuspenseLoader } from "@trenova/shared/components/component-loader";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { usePermission } from "@/hooks/use-permission";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { lazy, type ReactNode } from "react";
import { AgentList } from "./_components/agents/agent-list";
import { AIReadinessBanner } from "./_components/ai-readiness-banner";

const AgentControlForm = lazy(() => import("./_components/agent-control-form"));

/**
 * Everything that decides what AI may do in this organization, on one page.
 *
 * Providers say where the work goes and live with the other integrations; this
 * page says what the work is allowed to be. The readiness banner joins the two
 * so an agent is never switched on with nowhere to run.
 */
export function AgentControlPage() {
  const t = useT();
  const { allowed: canReadAgents } = usePermission(Resource.AgentDefinition, Operation.Read);

  return (
    <AdminPageLayout>
      <PageHeader
        title={t("Agent Control")}
        description={t(
          "Which agents your team can talk to, what each may do, and how the billing exception agent runs.",
        )}
      />
      <div className="flex flex-col gap-6 p-4">
        <AIReadinessBanner />

        {canReadAgents && (
          <Section
            title={t("Assistant agents")}
            description={t(
              "Build assistants from Trenova's templates. You choose which tools each one may use and how much it may do on its own; the template decides what it is allowed to reach at all.",
            )}
          >
            <AgentList />
          </Section>
        )}

        <Section
          title={t("Billing exception agent")}
          description={t(
            "Runs against blocked billing queue items on its own schedule and proposes resolutions for a person to approve.",
          )}
        >
          <SuspenseLoader>
            <AgentControlForm />
          </SuspenseLoader>
        </Section>
      </div>
    </AdminPageLayout>
  );
}

function Section({
  title,
  description,
  children,
}: {
  title: string;
  description: string;
  children: ReactNode;
}) {
  return (
    <section className="flex flex-col gap-3">
      <div>
        <h2 className="text-base font-semibold">{title}</h2>
        <p className="text-muted-foreground max-w-prose text-sm">{description}</p>
      </div>
      {children}
    </section>
  );
}
