import {
  Popover,
  PopoverContent,
  PopoverHeader,
  PopoverTitle,
  PopoverTrigger,
} from "@trenova/shared/components/ui/popover";
import { cn } from "@trenova/shared/lib/utils";
import { InfoIcon } from "lucide-react";
import type { ReactNode } from "react";

type InfoPopoverProps = {
  /** Names what the note explains; also the accessible name of the button. */
  title: string;
  /** Short prose. A string becomes one paragraph; pass elements for more. */
  children: ReactNode;
  className?: string;
};

/**
 * A small info icon that opens a note in plain words: how a figure is made,
 * what a section leaves out, what a button will do. Prose only; for a list
 * of labelled figures use KpiInfoPopover.
 */
export function InfoPopover({ title, children, className }: InfoPopoverProps) {
  return (
    <Popover>
      <PopoverTrigger
        render={
          <button
            type="button"
            aria-label={`About ${title}`}
            className={cn(
              "inline-flex size-4 shrink-0 items-center justify-center rounded-sm",
              "text-muted-foreground/70 hover:bg-muted hover:text-foreground transition-colors",
              "focus-visible:ring-ring focus-visible:ring-2 focus-visible:ring-offset-1 focus-visible:outline-hidden",
              className,
            )}
          >
            <InfoIcon className="size-3" />
          </button>
        }
      />
      <PopoverContent align="start" sideOffset={8} className="w-80 gap-2 p-3">
        <PopoverHeader>
          <PopoverTitle className="text-xs">{title}</PopoverTitle>
        </PopoverHeader>
        {typeof children === "string" ? (
          <p className="text-muted-foreground text-xs leading-snug">{children}</p>
        ) : (
          <div className="text-muted-foreground flex flex-col gap-1.5 text-xs leading-snug">
            {children}
          </div>
        )}
      </PopoverContent>
    </Popover>
  );
}
