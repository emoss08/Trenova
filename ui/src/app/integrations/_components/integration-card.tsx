import { IntegrationCategoryBadge } from "@/components/status-badge";
import { Button } from "@/components/ui/button";
import { LazyImage } from "@/components/ui/image";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import type { Integration } from "@/types/integration";
import { integrationImages } from "../_utils/integration";

export default function IntegrationCard({
  integration,
  handleConfigureClick,
}: {
  integration: Integration;
  handleConfigureClick: (integration: Integration) => void;
}) {
  const getButtonText = (integration: Integration) => {
    return integration.enabled ? "Manage" : "Enable";
  };

  return (
    <div
      key={integration.id}
      className="overflow-hidden border border-input rounded-md transition-all p-4"
    >
      <div className="flex flex-row items-center justify-between gap-4">
        <div className="flex flex-col">
          <div className="text-xl font-semibold">{integration.name}</div>
          <div className="text-xs text-muted-foreground">
            By {integration.builtBy}
          </div>
        </div>
        <div className="flex-shrink-0 rounded-full flex items-center justify-center p-2 border border-input">
          <LazyImage
            src={integrationImages[integration.type]}
            layout="fixed"
            width={10}
            height={10}
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
        <Tooltip delayDuration={300}>
          <TooltipTrigger asChild>
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
          </TooltipTrigger>
          <TooltipContent>
            {integration.enabled
              ? `Manage ${integration.name} configuration`
              : `Enable ${integration.name} integration`}
          </TooltipContent>
        </Tooltip>

        <IntegrationCategoryBadge category={integration.category} />
      </div>
    </div>
  );
}
