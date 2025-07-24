/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { queries } from "@/lib/queries";
import type { IntegrationSchema } from "@/lib/schemas/integration-schema";
import { IntegrationCategory } from "@/types/integration";
import { useQuery } from "@tanstack/react-query";
import { lazy, Suspense, useMemo, useState } from "react";
import { getCategoryDisplayName } from "../_utils/integration";
import { IntegrationConfigDialog } from "./integration-config-dialog";
import { IntegrationSkeleton } from "./integration-skeleton";

const IntegrationCard = lazy(() => import("./integration-card"));

export function IntegrationGrid() {
  const [selectedIntegration, setSelectedIntegration] =
    useState<IntegrationSchema | null>(null);

  // Fetch integrations
  const { data } = useQuery({
    ...queries.integration.getIntegrations(),
  });

  // Group integrations by category
  const groupedIntegrations = useMemo(() => {
    if (!data?.results) return {};

    return data.results.reduce<Record<string, IntegrationSchema[]>>(
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

  const handleConfigureClick = (integration: IntegrationSchema) => {
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
          onOpenChange={() => setSelectedIntegration(null)}
        />
      )}
    </>
  );
}
