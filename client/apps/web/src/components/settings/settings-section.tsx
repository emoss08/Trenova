import { Card, CardContent } from "@trenova/shared/components/ui/card";
import { cn } from "@trenova/shared/lib/utils";
import { useId, type ReactNode } from "react";
import {
  settingsNestedClass,
  settingsPanelClass,
  settingsSectionClass,
  settingsSectionHeadClass,
  settingsSectionsClass,
} from "./settings-layout";

/** The sections of one settings page or tab, stacked. */
export function SettingsSections({
  className,
  children,
}: {
  className?: string;
  children: ReactNode;
}) {
  return <div className={cn(settingsSectionsClass, className)}>{children}</div>;
}

/**
 * One group of settings: what it is and what it does on the left, its controls in a panel
 * on the right. The title column is what a person scans to find a setting, so it stays
 * free of controls; on a narrow screen it sits above the panel.
 */
export function SettingsSection({
  title,
  description,
  children,
}: {
  title: string;
  description?: ReactNode;
  children: ReactNode;
}) {
  const titleId = useId();

  return (
    <section aria-labelledby={titleId} className={settingsSectionClass}>
      <div className={settingsSectionHeadClass}>
        <h2 id={titleId} className="text-base leading-snug font-semibold">
          {title}
        </h2>
        {description ? (
          <p className="text-sm text-pretty text-muted-foreground">{description}</p>
        ) : null}
      </div>
      <Card>
        <CardContent className={settingsPanelClass}>{children}</CardContent>
      </Card>
    </section>
  );
}

/** Fields that only apply while the switch above them is on, set in under its label. */
export function SettingsNested({ children }: { children: ReactNode }) {
  return <div className={settingsNestedClass}>{children}</div>;
}
