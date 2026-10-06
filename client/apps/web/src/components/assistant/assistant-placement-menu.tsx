import { DeskIcon } from "@/components/desk-chat/desk-icons";
import {
  isAssistantDock,
  isAssistantLayout,
  type AssistantDock,
  type AssistantLayout,
} from "@/lib/assistant-dock";
import { useAssistantStore } from "@/stores/assistant-store";
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
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { dockLabel } from "./dock-label";

const DOCK_ORDER: readonly AssistantDock[] = [
  "bottom-right",
  "bottom-left",
  "top-right",
  "top-left",
];

const LAYOUT_ORDER: readonly AssistantLayout[] = ["compact", "side", "full"];

function layoutLabel(t: TranslateFn, layout: AssistantLayout): string {
  switch (layout) {
    case "side":
      return t("Docked to the side");
    case "full":
      return t("Full screen");
    default:
      return t("Floating in the corner");
  }
}

/**
 * Where the assistant sits and how much of the screen it takes. The same
 * choices the header's buttons, dragging the beacon and its hide button
 * offer, reachable from the keyboard.
 */
export function AssistantPlacementMenu() {
  const t = useT();
  const layout = useAssistantStore((state) => state.layout);
  const dock = useAssistantStore((state) => state.dock);
  const launcherHidden = useAssistantStore((state) => state.launcherHidden);
  const panelSize = useAssistantStore((state) => state.panelSize);
  const setLayout = useAssistantStore((state) => state.setLayout);
  const setDock = useAssistantStore((state) => state.setDock);
  const setLauncherHidden = useAssistantStore((state) => state.setLauncherHidden);
  const setPanelSize = useAssistantStore((state) => state.setPanelSize);

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <button
            type="button"
            className="dk-ib"
            title={t("Position and size")}
            aria-label={t("Position and size")}
          />
        }
      >
        <DeskIcon name="layout" size={15} />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-60">
        <DropdownMenuGroup>
          <DropdownMenuLabel>{t("Layout")}</DropdownMenuLabel>
          <DropdownMenuRadioGroup
            value={layout}
            onValueChange={(value) => {
              if (isAssistantLayout(value)) {
                setLayout(value);
              }
            }}
          >
            {LAYOUT_ORDER.map((option) => (
              <DropdownMenuRadioItem key={option} value={option}>
                {layoutLabel(t, option)}
              </DropdownMenuRadioItem>
            ))}
          </DropdownMenuRadioGroup>
        </DropdownMenuGroup>
        <DropdownMenuSeparator />
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
        <DropdownMenuItem
          title={t("Reset size")}
          disabled={panelSize === null}
          onClick={() => setPanelSize(null)}
        />
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
