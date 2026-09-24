import { Button } from "@trenova/shared/components/ui/button";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { PaletteAction, PaletteIntent } from "../../palette-model";
import { ShortcutKeys } from "../../palette-row";
import { PaletteTile } from "../../palette-tile";
import type { PaletteIcon } from "../../palette-model";

const FOOTER_ACTIONS = 3;

/**
 * The frame every preview sits in: who the thing is at the top, what is
 * known about it in the middle, and the first few things to do with it at
 * the bottom. Buttons keep focus in the search box, so the keyboard never
 * loses its place by a click.
 */
export function PreviewFrame({
  icon,
  initials,
  tileClass,
  title,
  subtitle,
  badge,
  actions,
  onRun,
  children,
}: {
  icon?: PaletteIcon;
  initials?: string;
  tileClass: string;
  title: React.ReactNode;
  subtitle?: React.ReactNode;
  badge?: React.ReactNode;
  actions: readonly PaletteAction[];
  onRun: (intent: PaletteIntent) => void;
  children?: React.ReactNode;
}) {
  return (
    <div className="animate-rise flex h-full min-h-0 flex-col">
      <header className="border-border-subtle flex items-start gap-3 border-b px-4 py-3.5">
        <PaletteTile icon={icon} initials={initials} className={tileClass} size="lg" />
        <div className="flex min-w-0 flex-1 flex-col gap-0.5">
          <div className="flex min-w-0 items-start justify-between gap-2">
            <h3 className="text-foreground truncate text-base font-semibold">{title}</h3>
            {badge && <div className="shrink-0">{badge}</div>}
          </div>
          {subtitle && <p className="text-foreground-muted truncate text-xs">{subtitle}</p>}
        </div>
      </header>
      <ScrollArea className="min-h-0 flex-1">
        <div className="flex flex-col gap-4 px-4 py-3.5">{children}</div>
      </ScrollArea>
      {actions.length > 0 && (
        <footer className="border-border-subtle flex flex-wrap items-center gap-1.5 border-t px-3 py-2">
          {actions.slice(0, FOOTER_ACTIONS).map((action, index) => (
            <Button
              key={action.id}
              type="button"
              size="xs"
              variant={index === 0 ? "secondary" : "ghost"}
              tabIndex={-1}
              onMouseDown={(event) => event.preventDefault()}
              onClick={() => onRun(action.intent)}
              className={cn("gap-1.5", action.destructive && "text-danger")}
            >
              <action.icon className="size-3.5" />
              <span className="truncate">{action.label}</span>
              {action.shortcut && <ShortcutKeys keys={action.shortcut} className="ml-0.5" />}
            </Button>
          ))}
        </footer>
      )}
    </div>
  );
}

export function PreviewSection({
  title,
  children,
  className,
}: {
  title: string;
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <section className={cn("flex flex-col gap-2", className)}>
      <h4 className="text-foreground-subtle text-xs font-medium">{title}</h4>
      {children}
    </section>
  );
}

export function PreviewSkeleton() {
  return (
    <div className="flex h-full flex-col">
      <div className="border-border-subtle flex items-start gap-3 border-b px-4 py-3.5">
        <Skeleton className="rounded-control size-10 shrink-0" />
        <div className="flex flex-1 flex-col gap-1.5 pt-0.5">
          <Skeleton className="h-4 w-40" />
          <Skeleton className="h-3 w-28" />
        </div>
      </div>
      <div className="flex flex-col gap-4 px-4 py-3.5">
        {Array.from({ length: 3 }, (_, block) => (
          <div key={block} className="flex flex-col gap-2">
            <Skeleton className="h-3 w-16" />
            <div className="grid grid-cols-2 gap-x-6 gap-y-3">
              {Array.from({ length: 4 }, (_, cell) => (
                <div key={cell} className="flex flex-col gap-1">
                  <Skeleton className="h-2.5 w-12" />
                  <Skeleton className="h-3.5 w-20" />
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

export function PreviewMessage({ children }: { children: React.ReactNode }) {
  return (
    <div className="text-foreground-subtle flex h-full items-center justify-center px-6 text-center text-xs">
      {children}
    </div>
  );
}

export function PreviewError() {
  const t = useT();
  return (
    <PreviewMessage>
      {t("Couldn't load a preview. The record may have moved, or you may not have access to it.")}
    </PreviewMessage>
  );
}
