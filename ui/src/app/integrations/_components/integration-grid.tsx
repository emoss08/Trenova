/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { EmptyState } from "@/components/ui/empty-state";
import { queries } from "@/lib/queries";
import type { IntegrationSchema } from "@/lib/schemas/integration-schema";
import { IntegrationCategory } from "@/types/integration";
import { faPuzzlePiece } from "@fortawesome/pro-regular-svg-icons";
import { useQuery } from "@tanstack/react-query";
import { lazy, Suspense, useEffect, useMemo, useState } from "react";
import { getCategoryDisplayName } from "../_utils/integration";
import { IntegrationConfigDialog } from "./integration-config-dialog";
import { IntegrationSkeleton } from "./integration-skeleton";

const IntegrationCard = lazy(() => import("./integration-card"));

type Props = {
  search?: string;
  category?: "All" | IntegrationCategory;
  onCount?: (count: number) => void;
};

export function IntegrationGrid({
  search = "",
  category = "All",
  onCount,
}: Props) {
  const [selectedIntegration, setSelectedIntegration] =
    useState<IntegrationSchema | null>(null);

  // Fetch integrations
  const { data } = useQuery({
    ...queries.integration.getIntegrations(),
  });

  const normalizedSearch = search.trim().toLowerCase();

  const filtered = useMemo(() => {
    const items = data?.results ?? [];
    const byCategory =
      category === "All" ? items : items.filter((i) => i.category === category);
    if (!normalizedSearch) return byCategory;
    return byCategory.filter((i) => {
      const hay = `${i.name} ${i.builtBy} ${i.description}`.toLowerCase();
      return hay.includes(normalizedSearch);
    });
  }, [data?.results, category, normalizedSearch]);

  // Sort alphabetical for consistent browsing
  const sorted = useMemo(
    () => [...filtered].sort((a, b) => a.name.localeCompare(b.name)),
    [filtered],
  );

  // Expose result count to parent when it changes
  useEffect(() => {
    onCount?.(sorted.length);
  }, [onCount, sorted.length]);

  // Group integrations by category (for the All tab)
  const groupedIntegrations = useMemo(() => {
    if (category !== "All") return {} as Record<string, IntegrationSchema[]>;
    return sorted.reduce<Record<string, IntegrationSchema[]>>((acc, i) => {
      const cat = i.category;
      acc[cat] = acc[cat] || [];
      acc[cat].push(i);
      return acc;
    }, {});
  }, [sorted, category]);

  const handleConfigureClick = (integration: IntegrationSchema) => {
    setSelectedIntegration(integration);
  };

  return (
    <>
      <div className="mt-6 space-y-10">
        {category === "All" ? (
          Object.entries(groupedIntegrations).length ? (
            Object.entries(groupedIntegrations).map(([cat, integrations]) => (
              <section key={cat} className="space-y-4">
                <div className="flex items-end justify-between">
                  <h2 className="text-lg font-semibold text-foreground">
                    {getCategoryDisplayName(cat as IntegrationCategory)}
                  </h2>
                  <span className="text-xs text-muted-foreground">
                    {integrations.length} apps
                  </span>
                </div>
                <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
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
              </section>
            ))
          ) : (
            <div className="flex w-full items-center justify-center py-16">
              <EmptyState
                title="No integrations found"
                description="Try adjusting your search or filters to see more results."
                icons={[faPuzzlePiece]}
              />
            </div>
          )
        ) : sorted.length ? (
          <section className="space-y-4">
            <div className="flex items-end justify-between">
              <h2 className="text-lg font-semibold text-foreground">
                {getCategoryDisplayName(category as IntegrationCategory)}
              </h2>
              <span className="text-xs text-muted-foreground">
                {sorted.length} apps
              </span>
            </div>
            <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
              {sorted.map((integration) => (
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
          </section>
        ) : (
          <div className="flex w-full items-center justify-center py-16">
            <EmptyState
              title="No integrations in this category"
              description="No results match your current search. Try clearing the search or picking a different category."
              icons={[faPuzzlePiece]}
            />
          </div>
        )}
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
