import { isAssistantDock, type AssistantDock } from "@/lib/assistant-dock";
import { useAssistantStore } from "@/stores/assistant-store";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { PanelBottomDashedIcon } from "lucide-react";

function dockLabel(t: TranslateFn, dock: AssistantDock): string {
  switch (dock) {
    case "bottom-right":
      return t("Bottom right");
    case "bottom-left":
      return t("Bottom left");
    case "top-right":
      return t("Top right");
    case "top-left":
      return t("Top left");
  }
}

const DOCK_ORDER: readonly AssistantDock[] = [
  "bottom-right",
  "bottom-left",
  "top-right",
  "top-left",
];

/**
 * Where the assistant sits and how much of the screen it takes. The same
 * choices the launcher offers by dragging and its hide button, reachable
 * from the keyboard.
 */
export function AssistantPlacementMenu() {
  const t = useT();
  const dock = useAssistantStore((state) => state.dock);
  const launcherHidden = useAssistantStore((state) => state.launcherHidden);
  const panelSize = useAssistantStore((state) => state.panelSize);
  const setDock = useAssistantStore((state) => state.setDock);
  const setLauncherHidden = useAssistantStore((state) => state.setLauncherHidden);
  const setPanelSize = useAssistantStore((state) => state.setPanelSize);

  return (
    <DropdownMenu>
      <Tooltip>
        <TooltipTrigger
          render={
            <DropdownMenuTrigger
              render={
                <Button
                  variant="ghost"
                  size="icon-sm"
                  aria-label={t("Position and size")}
                  className="text-muted-foreground hover:text-foreground"
                />
              }
            />
          }
        >
          <PanelBottomDashedIcon className="size-4" />
        </TooltipTrigger>
        <TooltipContent>{t("Position and size")}</TooltipContent>
      </Tooltip>
      <DropdownMenuContent align="end" className="w-60">
        <DropdownMenuGroup>
          <DropdownMenuLabel>{t("Corner")}</DropdownMenuLabel>
          <DropdownMenuRadioGroup
            value={dock}
            onValueChange={(value) => {
              if (isAssistantDock(value)) {
                setDock(value);
              }
            }}
          >
            {DOCK_ORDER.map((option) => (
              <DropdownMenuRadioItem key={option} value={option}>
                {dockLabel(t, option)}
              </DropdownMenuRadioItem>
            ))}
          </DropdownMenuRadioGroup>
        </DropdownMenuGroup>
        <DropdownMenuSeparator />
        <DropdownMenuCheckboxItem
          checked={launcherHidden}
          onCheckedChange={(checked) => setLauncherHidden(checked)}
        >
          {t("Hide the button when closed")}
        </DropdownMenuCheckboxItem>
        <DropdownMenuItem disabled={panelSize === null} onClick={() => setPanelSize(null)}>
          {t("Reset size")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
