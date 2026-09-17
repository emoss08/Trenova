import { useT } from "@trenova/shared/i18n/use-t";
import { usePermission } from "@/hooks/use-permission";
import { useAssistantStore } from "@/stores/assistant-store";
import { Button } from "@trenova/shared/components/ui/button";
import { Kbd, KbdGroup } from "@trenova/shared/components/ui/kbd";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { SparklesIcon } from "lucide-react";

/**
 * The header's way into the assistant. It toggles the floating panel rather
 * than leaving the page, because the question is usually about the page.
 */
export function AssistantTrigger({
  className,
  tooltipSide = "bottom",
}: {
  className?: string;
  tooltipSide?: "right" | "bottom";
}) {
  const t = useT();
  const { allowed } = usePermission(Resource.Assistant, Operation.Read);
  const open = useAssistantStore((state) => state.open);
  const toggleWidget = useAssistantStore((state) => state.toggleWidget);

  if (!allowed) {
    return null;
  }

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label={t("Assistant")}
            aria-pressed={open}
            onClick={toggleWidget}
            className={cn(
              "text-muted-foreground hover:text-foreground",
              open && "bg-muted text-foreground",
              className,
            )}
          />
        }
      >
        <SparklesIcon className="size-4" strokeWidth={1.75} />
      </TooltipTrigger>
      <TooltipContent side={tooltipSide} sideOffset={10} className="flex items-center gap-2">
        {t("Assistant")}
        <KbdGroup>
          <Kbd>⌘</Kbd>
          <Kbd>J</Kbd>
        </KbdGroup>
      </TooltipContent>
    </Tooltip>
  );
}
