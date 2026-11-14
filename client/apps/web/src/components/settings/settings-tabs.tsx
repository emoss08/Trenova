import { Tabs, TabsList, TabsPanel, TabsTab } from "@trenova/shared/components/ui/tabs";
import { useT } from "@trenova/shared/i18n/use-t";
import type { ReactNode } from "react";
import { useFormState } from "react-hook-form";
import { settingsTabRowClass, settingsTabsClass } from "./settings-layout";
import type { SettingsTabSpec } from "./settings-tab";

export type SettingsTab<TField extends string = string> = SettingsTabSpec<TField> & {
  content: ReactNode;
};

/**
 * A settings form split into tabs, one job to a tab, inside one form with one save.
 *
 * Every tab stays mounted, so each field is registered whichever tab is open: a server
 * error lands on its field, and a tab that holds a problem is marked. The page opens that
 * tab when a save is refused (`tabHoldingError`). A page with one tab draws no tab row.
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

  if (tabs.length === 1) {
    return <>{tabs[0].content}</>;
  }

  return (
    <Tabs
      value={value}
      onValueChange={(next) => onValueChange(String(next))}
      className={settingsTabsClass}
    >
      <div className={settingsTabRowClass}>
        <TabsList variant="underline">
          {tabs.map((tab) => (
            <TabsTab key={tab.value} value={tab.value}>
              {tab.label}
              {tab.fields.some((field) => field in errors) ? (
                <>
                  <span aria-hidden className="size-1.5 rounded-full bg-danger" />
                  <span className="sr-only">{t("Has fields to fix")}</span>
                </>
              ) : null}
            </TabsTab>
          ))}
        </TabsList>
      </div>
      {tabs.map((tab) => (
        <TabsPanel key={tab.value} value={tab.value} keepMounted>
          {tab.content}
        </TabsPanel>
      ))}
    </Tabs>
  );
}
