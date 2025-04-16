import { MetaTags } from "@/components/meta-tags";
import { IntegrationGrid } from "./_components/integration-grid";

export function IntegrationsPage() {
  return (
    <div className="p-4">
      <MetaTags title="Apps & Integrations" />
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold">Apps & Integrations</h1>
          <p className="text-sm text-muted-foreground">
            Connect and configure third-party services with your transportation
            management system
          </p>
        </div>
      </div>

      <IntegrationGrid />
    </div>
  );
}
