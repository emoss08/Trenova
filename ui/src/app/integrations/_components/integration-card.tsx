/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { IntegrationCategoryBadge } from "@/components/status-badge";
import { Button } from "@/components/ui/button";
import { LazyImage } from "@/components/ui/image";
import type { IntegrationSchema } from "@/lib/schemas/integration-schema";
import { integrationImages } from "../_utils/integration";

export default function IntegrationCard({
  integration,
  handleConfigureClick,
}: {
  integration: IntegrationSchema;
  handleConfigureClick: (integration: IntegrationSchema) => void;
}) {
  const getButtonText = (integration: IntegrationSchema) => {
    return integration.enabled ? "Manage" : "Enable";
  };

  return (
    <div
      key={integration.id}
      className="overflow-hidden border border-input rounded-md transition-all p-4 bg-card"
    >
      <div className="flex flex-row items-center justify-between gap-4">
        <div className="flex flex-col">
          <div className="text-xl font-semibold">{integration.name}</div>
          <div className="text-xs text-muted-foreground">
            By {integration.builtBy}
          </div>
        </div>
        <div className="flex-shrink-0 rounded-full flex items-center justify-center p-2 border border-input bg-background">
          <LazyImage
            src={integrationImages[integration.type]}
            className="size-6"
          />
        </div>
      </div>
      <div className="mt-2">
        <div className="h-[80px] text-wrap truncate text-sm text-muted-foreground">
          {integration.description}
        </div>
      </div>

      <div className="flex items-center justify-between">
        <Button
          onClick={(e) => {
            e.stopPropagation();
            handleConfigureClick(integration);
          }}
          type="button"
          variant={integration.enabled ? "outline" : "default"}
        >
          {getButtonText(integration)}
        </Button>

        <IntegrationCategoryBadge category={integration.category} />
      </div>
    </div>
  );
}
