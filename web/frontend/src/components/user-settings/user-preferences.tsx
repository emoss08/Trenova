import { ScrollArea } from "../ui/scroll-area";
import { ColorBlindSwitcher } from "./appearance/color-mode-switcher";
import { ThemeSwitcher } from "./appearance/theme-switcher";

export default function UserPreferences() {
  return (
    <>
      <div className="space-y-3">
        <div className="sticky top-0 z-20 mb-6 flex items-center gap-x-2">
          <h2 className="shrink-0 text-sm" id="personal-information">
            Preferences
          </h2>
          <p className="text-xs text-muted-foreground">
            Adjust your interface settings to suit your individual needs.
          </p>
        </div>
      </div>
      <ScrollArea className="-mr-4 h-[550px]">
        <ThemeSwitcher />
        <ColorBlindSwitcher />
      </ScrollArea>
    </>
  );
}
