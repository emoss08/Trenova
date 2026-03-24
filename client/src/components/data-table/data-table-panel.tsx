"use no memo";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";
import { Dialog } from "@base-ui/react/dialog";
import { XIcon } from "lucide-react";

const PANEL_SIZES = {
  sm: 400,
  md: 500,
  lg: 650,
  xl: 800,
} as const;

export type PanelSize = keyof typeof PANEL_SIZES;

type DataTablePanelContainerProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description?: string;
  titleComponent?: React.ReactNode;
  children: React.ReactNode;
  footer?: React.ReactNode;
  headerActions?: React.ReactNode;
  size?: PanelSize;
};

export function DataTablePanelContainer({
  open,
  onOpenChange,
  title,
  description,
  titleComponent,
  children,
  footer,
  headerActions,
  size = "md",
}: DataTablePanelContainerProps) {
  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Popup
          className={cn(
            "fixed top-4 right-4 bottom-4 z-50 flex flex-col rounded-lg border border-border bg-background shadow-lg outline-none",
            "data-[open]:animate-in data-[open]:slide-in-from-right",
            "data-[closed]:animate-out data-[closed]:slide-out-to-right",
            "duration-200",
          )}
          style={{ width: PANEL_SIZES[size] }}
        >
          <div className="flex items-center justify-between border-b border-border px-4 py-3">
            {titleComponent ?? (
              <div className="flex flex-col gap-0.5">
                <Dialog.Title className="text-sm leading-none font-medium">{title}</Dialog.Title>
                {description && (
                  <Dialog.Description className="text-xs text-muted-foreground">
                    {description}
                  </Dialog.Description>
                )}
              </div>
            )}
            <div className="flex items-center gap-1">
              {headerActions}
              <Dialog.Close
                render={
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    className="text-muted-foreground hover:text-foreground"
                  />
                }
              >
                <XIcon className="size-4" />
                <span className="sr-only">Close panel</span>
              </Dialog.Close>
            </div>
          </div>
          <ScrollArea className="flex-1">
            <div className="p-4">{children}</div>
          </ScrollArea>
          {footer && (
            <div className="flex items-center justify-between gap-2 border-t border-border bg-muted/30 px-4 py-3">
              {footer}
            </div>
          )}
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

type DataTablePanelWrapperProps = {
  children: React.ReactNode;
  className?: string;
};

export function DataTablePanelWrapper({ children, className }: DataTablePanelWrapperProps) {
  return <div className={cn("flex h-full", className)}>{children}</div>;
}

type DataTablePanelContentProps = {
  className?: string;
  children: React.ReactNode;
};

export function DataTablePanelContent({ className, children }: DataTablePanelContentProps) {
  return (
    <div
      className={cn("flex min-w-0 flex-1 flex-col gap-2 transition-all duration-200", className)}
    >
      {children}
    </div>
  );
}
