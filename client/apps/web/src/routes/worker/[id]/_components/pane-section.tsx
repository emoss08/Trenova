import { FormGroup } from "@trenova/shared/components/ui/form";
import { useId, type ReactNode } from "react";

/** A titled part of a read pane's body: a small quiet heading over what it holds. */
export function PaneSection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section aria-label={title} className="flex flex-col gap-2">
      <h4 className="text-xs font-medium text-foreground-subtle">{title}</h4>
      {children}
    </section>
  );
}

/**
 * A titled group of a form pane's fields, two to a row, with a line under the title when the
 * group needs saying what it is for.
 */
export function PaneFormSection({
  title,
  description,
  children,
}: {
  title: string;
  description?: string;
  children: ReactNode;
}) {
  const descriptionId = useId();
  return (
    <section
      aria-label={title}
      aria-describedby={description ? descriptionId : undefined}
      className="flex flex-col gap-3"
    >
      <div className="flex flex-col gap-0.5">
        <h4 className="text-sm font-semibold">{title}</h4>
        {description ? (
          <p id={descriptionId} className="text-xs text-muted-foreground">
            {description}
          </p>
        ) : null}
      </div>
      <FormGroup cols={2}>{children}</FormGroup>
    </section>
  );
}
