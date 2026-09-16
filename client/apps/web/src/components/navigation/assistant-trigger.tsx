import { useT } from "@trenova/shared/i18n/use-t";
import { usePermission } from "@/hooks/use-permission";
import { Button } from "@trenova/shared/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { SparklesIcon } from "lucide-react";
import { Link, useLocation } from "react-router";

/**
 * The one global way into the assistant.
 *
 * It sits with the other actions that apply everywhere because a question
 * about a shipment comes up on every page, not on one. A reader who may not
 * use the assistant gets no button rather than a button that refuses.
 */
export function AssistantTrigger({
  className,
  tooltipSide = "bottom",
}: {
  className?: string;
  tooltipSide?: "right" | "bottom";
}) {
  const t = useT();
  const { pathname } = useLocation();
  const { allowed } = usePermission(Resource.Assistant, Operation.Read);

  if (!allowed) {
    return null;
  }

  const active = pathname.startsWith("/assistant");

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label={t("Assistant")}
            aria-current={active ? "page" : undefined}
            nativeButton={false}
            render={<Link to="/assistant" />}
            className={cn(
              "text-muted-foreground hover:text-foreground",
              active && "bg-muted text-foreground",
              className,
            )}
          />
        }
      >
        <SparklesIcon className="size-4" strokeWidth={1.75} />
      </TooltipTrigger>
      <TooltipContent side={tooltipSide} sideOffset={10}>
        {t("Assistant")}
      </TooltipContent>
    </Tooltip>
  );
}
