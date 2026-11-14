import { FieldLayoutProvider } from "@/components/fields/field-components";
import { cn } from "@trenova/shared/lib/utils";
import { useId, type ReactNode } from "react";
import {
  settingsColumnClass,
  settingsListClass,
  settingsNestedClass,
  settingsReadoutClass,
  settingsSectionClass,
  settingsSectionHeadClass,
  settingsSectionsClass,
} from "./settings-layout";
import { SettingsReveal } from "./settings-reveal";

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

/** The centred column of a settings page that has no tabs; `SettingsTabs` draws its own. */
export function SettingsColumn({ children }: { children: ReactNode }) {
  return <div className={settingsColumnClass}>{children}</div>;
}

/**
 * One group of settings: what it is, then a list with one setting to a row. Every shared
 * field placed in it draws as a row (its name and meaning on the left, its control at the
 * end) without being told to, so the list reads down one column of values.
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
    <section
      data-slot="settings-section"
      aria-labelledby={titleId}
      className={settingsSectionClass}
    >
      <div className={settingsSectionHeadClass}>
        <h2 id={titleId} className="text-base leading-snug font-semibold">
          {title}
        </h2>
        {description ? (
          <p className="max-w-160 text-sm text-pretty text-muted-foreground">{description}</p>
        ) : null}
      </div>
      <FieldLayoutProvider layout="row">
        <div data-slot="settings-list" className={settingsListClass}>
          {children}
        </div>
      </FieldLayoutProvider>
    </section>
  );
}

/**
 * Rows that only apply while the row above them is on (a switch, a choice other than
 * "never"): set in under it on a ground of their own, opening and closing with it.
 */
export function SettingsNested({ show, children }: { show: boolean; children: ReactNode }) {
  return (
    <SettingsReveal show={show} className={settingsNestedClass}>
      {children}
    </SettingsReveal>
  );
}

/** What a section reports rather than asks, such as what the planner has learned. */
export function SettingsReadout({ children }: { children: ReactNode }) {
  return <div className={settingsReadoutClass}>{children}</div>;
}
