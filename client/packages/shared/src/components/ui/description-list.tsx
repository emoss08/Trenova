import { cn } from "@trenova/shared/lib/utils";
import { createContext, useContext } from "react";
import type React from "react";

type DescriptionListLayout = "stacked" | "inline" | "split";

const DescriptionListContext = createContext<DescriptionListLayout>("stacked");

const LIST_LAYOUT: Record<DescriptionListLayout, string> = {
  stacked: "grid gap-x-6 gap-y-3",
  inline: "grid grid-cols-[minmax(7rem,auto)_minmax(0,1fr)] gap-x-4 gap-y-1.5",
  split: "divide-border-subtle flex flex-col divide-y",
};

const STACKED_COLUMNS = {
  1: "grid-cols-1",
  2: "grid-cols-2",
  3: "grid-cols-2 md:grid-cols-3",
  4: "grid-cols-2 md:grid-cols-4",
} as const;

type DescriptionListProps = Omit<React.ComponentProps<"dl">, "children"> & {
  layout?: DescriptionListLayout;
  columns?: keyof typeof STACKED_COLUMNS;
  children: React.ReactNode;
};

function DescriptionList({
  layout = "stacked",
  columns = 2,
  className,
  children,
  ...props
}: DescriptionListProps) {
  return (
    <DescriptionListContext.Provider value={layout}>
      <dl
        data-slot="description-list"
        data-layout={layout}
        className={cn(
          "min-w-0 text-sm",
          LIST_LAYOUT[layout],
          layout === "stacked" && STACKED_COLUMNS[columns],
          className,
        )}
        {...props}
      >
        {children}
      </dl>
    </DescriptionListContext.Provider>
  );
}

type DescriptionItemProps = {
  label: React.ReactNode;
  children: React.ReactNode;
  numeric?: boolean;
  span?: 1 | 2 | "full";
  className?: string;
  valueClassName?: string;
};

const TERM_CLASS = "text-foreground-subtle text-xs font-medium";
const VALUE_CLASS = "text-foreground min-w-0 text-sm";

function DescriptionItem({
  label,
  children,
  numeric = false,
  span = 1,
  className,
  valueClassName,
}: DescriptionItemProps) {
  const layout = useContext(DescriptionListContext);
  const value = cn(VALUE_CLASS, numeric && "tabular-nums", valueClassName);

  if (layout === "inline") {
    return (
      <>
        <dt className={cn(TERM_CLASS, "py-px", className)}>{label}</dt>
        <dd className={cn(value, "break-words")}>{children}</dd>
      </>
    );
  }

  if (layout === "split") {
    return (
      <div className={cn("flex items-baseline justify-between gap-4 py-1.5", className)}>
        <dt className={cn(TERM_CLASS, "shrink-0")}>{label}</dt>
        <dd className={cn(value, "truncate text-right")}>{children}</dd>
      </div>
    );
  }

  return (
    <div
      className={cn(
        "flex min-w-0 flex-col gap-0.5",
        span === 2 && "col-span-2",
        span === "full" && "col-span-full",
        className,
      )}
    >
      <dt className={TERM_CLASS}>{label}</dt>
      <dd className={cn(value, "break-words")}>{children}</dd>
    </div>
  );
}

const DESCRIPTION_EMPTY = "—";

function DescriptionEmpty() {
  return <span className="text-foreground-subtle">{DESCRIPTION_EMPTY}</span>;
}

export { DescriptionEmpty, DescriptionItem, DescriptionList };
