import type { AssistantDock } from "@/lib/assistant-dock";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";

export function dockLabel(t: TranslateFn, dock: AssistantDock): string {
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
