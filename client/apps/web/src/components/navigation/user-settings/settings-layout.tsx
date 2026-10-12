import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";

export function settingsSectionId(section: string): string {
  return `settings-section-${section}`;
}

export function settingsRowId(row: string): string {
  return `settings-row-${row}`;
}

export function SettingsSection({
  id,
  title,
  description,
  badge,
  children,
}: {
  id: string;
  title: string;
  description: string;
  badge?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section
      id={settingsSectionId(id)}
      data-settings-section={id}
      aria-labelledby={`${settingsSectionId(id)}-title`}
      className="mb-14 scroll-mt-6 pt-3"
    >
      <div className="mb-4 flex items-end gap-4">
        <div className="min-w-0 flex-1">
          <h2
            id={`${settingsSectionId(id)}-title`}
            className="text-xl leading-tight font-medium tracking-tight"
          >
            {title}
          </h2>
          <p className="text-foreground-subtle mt-1 max-w-[56ch] text-sm text-pretty">
            {description}
          </p>
        </div>
        {badge}
      </div>
      <div className="flex flex-col gap-3.5">{children}</div>
    </section>
  );
}

export function SettingsCard({
  id,
  title,
  description,
  actions,
  children,
  className,
}: {
  id?: string;
  title?: string;
  description?: ReactNode;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
}) {
  return (
    <div
      id={id ? settingsRowId(id) : undefined}
      data-settings-row={id}
      className={cn(
        "bg-card overflow-hidden rounded-lg border [&_[data-slot=label]]:font-sans [&_[data-slot=label]]:text-base",
        className,
      )}
    >
      {title ? (
        <div className="border-border-subtle flex items-start gap-3 border-b px-4.5 pt-3.5 pb-3">
          <div className="flex min-w-0 flex-1 flex-col gap-0.5">
            <h3 className="text-sm font-medium">{title}</h3>
            {description ? (
              <p className="text-foreground-subtle text-xs text-pretty">{description}</p>
            ) : null}
          </div>
          {actions}
        </div>
      ) : null}
      <div className="divide-border-subtle flex flex-col divide-y">{children}</div>
    </div>
  );
}

/**
 * One setting in a card. A sub-row is revealed by the setting above it and sits on the
 * sunken fill so it reads as belonging to it.
 */
export function SettingsRow({
  id,
  sub,
  children,
  className,
}: {
  id?: string;
  sub?: boolean;
  children: ReactNode;
  className?: string;
}) {
  return (
    <div
      id={id ? settingsRowId(id) : undefined}
      data-settings-row={id}
      className={cn(
        "px-4.5 py-3.5",
        sub && "bg-sunken animate-in fade-in-0 slide-in-from-top-1 duration-300",
        className,
      )}
    >
      {children}
    </div>
  );
}
