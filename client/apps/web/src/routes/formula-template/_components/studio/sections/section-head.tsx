import type { IconComponent } from "@trenova/shared/components/icons";
import type { ReactNode } from "react";
import { sectionDomId, type StudioSection } from "../studio-context";

/** A section's title row: icon and title over one line of what it is for, actions on the right. */
export function SectionHead({
  section,
  icon: Icon,
  title,
  description,
  children,
}: {
  section: StudioSection;
  icon: IconComponent;
  title: string;
  description?: string;
  children?: ReactNode;
}) {
  return (
    <div id={sectionDomId(section)} className="flex scroll-mt-6 items-end justify-between gap-3">
      <div className="flex min-w-0 flex-col gap-0.5">
        <h2 className="flex items-center gap-1.5 text-lg font-semibold">
          <Icon className="text-foreground-subtle size-4" />
          {title}
        </h2>
        {description && <p className="text-foreground-muted text-sm">{description}</p>}
      </div>
      {children && <div className="flex shrink-0 items-center gap-1.5">{children}</div>}
    </div>
  );
}
