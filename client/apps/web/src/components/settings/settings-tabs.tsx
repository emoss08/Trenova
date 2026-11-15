import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Tabs, TabsList, TabsPanel, TabsTab } from "@trenova/shared/components/ui/tabs";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useLayoutEffect, useRef, type ReactNode } from "react";
import { useFormState } from "react-hook-form";
import {
  settingsColumnClass,
  settingsScrollBodyClass,
  settingsScrollClass,
  settingsTabRowClass,
  settingsTabsClass,
} from "./settings-layout";
import type { SettingsTabSpec } from "./settings-tab";

export type SettingsTab<TField extends string = string> = SettingsTabSpec<TField> & {
  content: ReactNode;
};

/**
 * A settings form split into tabs, one job to a tab, inside one form with one save.
 *
 * Every tab stays mounted, so each field is registered whichever tab is open: a server
 * error lands on its field, and a tab that holds a problem is marked. The page opens that
 * tab when a save is refused (`tabHoldingError`). A tab's content comes in from the side
 * the person moved towards. A page with one tab draws no tab row.
 *
 * The tab row holds its place at the top and only the open tab's content scrolls, so the
 * tabs are where they were however far down a person has read. It needs a page that gives
 * it a height: `PageLayout fill`, with the form between them passing it on
 * (`settingsFillClass`). A tab opens at its top, whichever way it was opened.
 */
export function SettingsTabs<TField extends string>({
  tabs,
  value,
  onValueChange,
}: {
  tabs: readonly SettingsTab<TField>[];
  value: string;
  onValueChange: (value: string) => void;
}) {
  const t = useT();
  const { errors } = useFormState();
  const viewport = useRef<HTMLDivElement>(null);
  const bodyClass = cn(settingsColumnClass, settingsScrollBodyClass);

  // A tab is opened by its tab, by a refused save and by a link inside another tab; each
  // starts it at its top rather than as far down as the last tab was read.
  useLayoutEffect(() => {
    viewport.current?.scrollTo({ top: 0 });
  }, [value]);

  if (tabs.length === 1) {
    return (
      <ScrollArea className={settingsScrollClass}>
        <div className={bodyClass}>{tabs[0].content}</div>
      </ScrollArea>
    );
  }

  return (
    <Tabs
      value={value}
      onValueChange={(next) => onValueChange(String(next))}
      className={settingsTabsClass}
    >
      <div className={cn(settingsColumnClass, settingsTabRowClass)}>
        <TabsList variant="underline">
          {tabs.map((tab) => (
            <TabsTab key={tab.value} value={tab.value}>
              {tab.label}
              {tab.fields.some((field) => field in errors) ? (
                <>
                  <span aria-hidden className="size-1.5 animate-rise rounded-full bg-danger" />
                  <span className="sr-only">{t("Has fields to fix")}</span>
                </>
              ) : null}
            </TabsTab>
          ))}
        </TabsList>
      </div>
      <ScrollArea viewportRef={viewport} className={settingsScrollClass}>
        <div className={bodyClass}>
          {tabs.map((tab) => (
            <TabsPanel
              key={tab.value}
              value={tab.value}
              keepMounted
              // The tab library keeps a tab that was just left on screen until its animations
              // end, and the arrival below would replay on it: the old tab's settings showed
              // under the new one's for the length of the slide, and several stacked up when
              // tabs were switched quickly. A tab that is left goes at once; only the one
              // arriving moves.
              className="animate-slide-in data-ending-style:hidden data-[activation-direction=left]:animate-slide-in-back data-[activation-direction=none]:animate-none"
            >
              {tab.content}
            </TabsPanel>
          ))}
        </div>
      </ScrollArea>
    </Tabs>
  );
}
