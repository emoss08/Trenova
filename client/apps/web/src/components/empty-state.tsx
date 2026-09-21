import { useT } from "@trenova/shared/i18n/use-t";
import * as React from "react";
import { cn } from "@trenova/shared/lib/utils";
import { Button } from "@trenova/shared/components/ui/button";
import type { LucideIcon } from "lucide-react";

interface EmptyStateProps {
  title: string;
  description: string;
  icons?: LucideIcon[];
  action?: {
    icon?: LucideIcon;
    label: string;
    onClick: () => void;
  };
  className?: string;
}

export function EmptyState({ title, description, icons = [], action, className }: EmptyStateProps) {
  const t = useT();

  return (
    <div
      className={cn(
        "border-border bg-card w-full max-w-155 rounded-lg border border-dashed px-10 py-12 text-center",
        className,
      )}
    >
      {icons.length > 0 && (
        <div className="flex justify-center">
          <div className="border-border bg-sunken divide-border text-foreground-subtle inline-flex divide-x overflow-hidden rounded-md border">
            {icons.map((icon, index) => (
              <span key={index} className="grid size-9 place-items-center">
                {React.createElement(icon, { className: "size-4", strokeWidth: 1.75 })}
              </span>
            ))}
          </div>
        </div>
      )}
      <h2 className="text-foreground mt-4 text-base font-semibold text-balance">{title}</h2>
      <p className="text-foreground-muted mx-auto mt-1 max-w-[52ch] text-sm text-pretty whitespace-pre-line">
        {description}
      </p>
      {action && (
        <Button onClick={action.onClick} variant="outline" size="sm" className="mt-4">
          {action.icon && React.createElement(action.icon, { className: "size-4" })}
          {t(action.label)}
        </Button>
      )}
    </div>
  );
}
