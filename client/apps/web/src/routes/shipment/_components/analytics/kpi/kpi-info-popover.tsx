import {
  Popover,
  PopoverContent,
  PopoverDescription,
  PopoverHeader,
  PopoverTitle,
  PopoverTrigger,
} from "@trenova/shared/components/ui/popover";
import { cn } from "@trenova/shared/lib/utils";
import { InfoIcon } from "lucide-react";

export type KpiInfoRow = {
  label: string;
  value: string;
};

type KpiInfoPopoverProps = {
  title: string;
  description?: string;
  rows: KpiInfoRow[];
};

export function KpiInfoPopover({ title, description, rows }: KpiInfoPopoverProps) {
  return (
    <Popover>
      <PopoverTrigger
        render={
          <button
            type="button"
            aria-label={`${title} calculation details`}
            className={cn(
              "inline-flex size-4 shrink-0 items-center justify-center rounded-sm",
              "text-muted-foreground/70 transition-colors hover:bg-muted hover:text-foreground",
              "focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 focus-visible:outline-hidden",
            )}
          >
            <InfoIcon className="size-3" />
          </button>
        }
      />
      <PopoverContent align="start" sideOffset={8} className="w-80 gap-3 p-3">
        <PopoverHeader>
          <PopoverTitle className="text-xs">{title}</PopoverTitle>
          {description && (
            <PopoverDescription className="text-xs leading-snug">{description}</PopoverDescription>
          )}
        </PopoverHeader>
        <dl className="grid gap-2">
          {rows.map((row) => (
            <div key={row.label} className="grid gap-0.5">
              <dt className="font-mono text-[10px] tracking-wide text-muted-foreground uppercase">
                {row.label}
              </dt>
              <dd className="text-xs leading-snug text-foreground/90">{row.value}</dd>
            </div>
          ))}
        </dl>
      </PopoverContent>
    </Popover>
  );
}
