import { queries } from "@/lib/queries";
import type { Integration } from "@/types/integrations/integration";
import { IntegrationCategory } from "@/types/integrations/integration";
import { useQuery } from "@tanstack/react-query";
import { lazy, Suspense, useMemo, useState } from "react";
import { getCategoryDisplayName } from "../_utils/integration";
import { IntegrationConfigDialog } from "./integration-config-dialog";
import { IntegrationSkeleton } from "./integration-skeleton";

const IntegrationCard = lazy(() => import("./integration-card"));

export function IntegrationGrid() {
  const [selectedIntegration, setSelectedIntegration] =
    useState<Integration | null>(null);

  // Fetch integrations
  const { data } = useQuery({
    ...queries.integration.getIntegrations(),
  });

  // Group integrations by category
  const groupedIntegrations = useMemo(() => {
    if (!data?.results) return {};

    return data.results.reduce<Record<string, Integration[]>>(
      (acc, integration) => {
        const category = integration.category;
        if (!acc[category]) {
          acc[category] = [];
        }
        acc[category].push(integration);
        return acc;
      },
      {},
    );
  }, [data?.results]);

  const handleConfigureClick = (integration: Integration) => {
    setSelectedIntegration(integration);
  };

  return (
    <>
      <div className="mt-4 space-y-8">
        {Object.entries(groupedIntegrations).map(([category, integrations]) => (
          <div key={category} className="space-y-4">
            <h2 className="text-lg font-semibold">
              {getCategoryDisplayName(category as IntegrationCategory)}
            </h2>
            <div className="grid grid-cols-1 gap-6 xl:grid-cols-4">
              {integrations.map((integration) => (
                <Suspense
                  key={integration.id}
                  fallback={<IntegrationSkeleton />}
                >
                  <IntegrationCard
                    integration={integration}
                    handleConfigureClick={handleConfigureClick}
                  />
                </Suspense>
              ))}
            </div>
          </div>
        ))}
      </div>

      {selectedIntegration && (
        <IntegrationConfigDialog
          integration={selectedIntegration}
          open={!!selectedIntegration}
          onClose={() => setSelectedIntegration(null)}
        />
      )}
    </>
  );
}
