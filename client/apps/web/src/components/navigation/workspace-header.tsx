import {
  FavoriteToggle,
  HeaderBreadcrumbs,
  HistoryNavigation,
  SidebarToggle,
} from "@/components/header";
import { ModulesMenu } from "@/components/navigation/modules-menu";
import { OrgSwitcher } from "@/components/navigation/org-switcher";
import { SearchTrigger } from "@/components/navigation/sidebar-chrome";
import { moduleDisplayLabel } from "@/components/navigation/sidebar-model";
import { UserMenu } from "@/components/navigation/user-menu";
import { ModuleTile } from "@/components/navigation/workspace-primitives";
import { NotificationSheet } from "@/components/notification-center/notification-sheet";
import { useSidebarNavigation } from "@/hooks/use-sidebar-navigation";
import { cn } from "@trenova/shared/lib/utils";
import { ChevronDownIcon, GripIcon } from "lucide-react";

export const WORKSPACE_HEADER_HEIGHT_CLASS = "h-10";

/**
 * Reads as where you are and opens the map of where you can go, the way a
 * scope switcher does. On a page no module owns it falls back to a plain
 * "Modules" label so the control never disappears.
 */
function ModulesTrigger() {
  const { activeModule } = useSidebarNavigation();
  const label = activeModule ? moduleDisplayLabel(activeModule) : "Modules";

  return (
    <ModulesMenu
      trigger={
        <button
          type="button"
          aria-label={activeModule ? `Switch module (current: ${label})` : "Open modules"}
          className={cn(
            "flex h-7 max-w-56 items-center gap-1.5 rounded-md pr-1.5 pl-1 text-sm font-medium transition-colors outline-none",
            "text-foreground hover:bg-muted data-popup-open:bg-muted focus-visible:ring-ring/50 focus-visible:ring-2",
          )}
        >
          {activeModule ? (
            <ModuleTile icon={activeModule.icon} size="sm" className="size-5 rounded-[5px]" />
          ) : (
            <GripIcon className="text-muted-foreground ml-0.5 size-3.5" strokeWidth={1.75} />
          )}
          <span className="truncate">{label}</span>
          <ChevronDownIcon className="text-muted-foreground size-3 shrink-0" strokeWidth={1.75} />
        </button>
      }
    />
  );
}

/**
 * The global strip of the workspace layout, in three zones: who you work as
 * and the map of the product on the left, search in the middle, and the
 * actions that apply everywhere on the right. Where you are is not its job;
 * the context bar above the page carries that.
 */
export function WorkspaceHeader() {
  return (
    <header
      className={cn(
        "bg-sidebar border-border relative z-50 grid shrink-0 grid-cols-[1fr_auto_1fr] items-center gap-3 border-b pr-2.5 pl-2",
        WORKSPACE_HEADER_HEIGHT_CLASS,
      )}
    >
      <div className="flex min-w-0 items-center gap-1.5">
        <div className="flex min-w-0 max-w-64 shrink items-center">
          <OrgSwitcher />
        </div>
        <div aria-hidden className="bg-border mx-0.5 h-4 w-px shrink-0" />
        <ModulesTrigger />
      </div>

      <div className="flex justify-center">
        <div className="hidden w-[min(26rem,40vw)] md:block">
          <SearchTrigger />
        </div>
        <div className="md:hidden">
          <SearchTrigger compact tooltipSide="bottom" />
        </div>
      </div>

      <div className="flex min-w-0 items-center justify-end gap-1.5">
        <NotificationSheet />
        <UserMenu compact />
      </div>
    </header>
  );
}

/**
 * Where you are and the few things you do with a page as a whole: show or
 * hide the sidebar, step back through history, pin it. It sits over the
 * page, not in the header, because it changes with every route while the
 * header never does.
 */
export function WorkspaceContextBar() {
  return (
    <div className="border-border bg-background flex h-8 shrink-0 items-center gap-2 border-b px-2">
      <SidebarToggle />
      <HistoryNavigation />
      <div className="min-w-0 flex-1">
        <HeaderBreadcrumbs />
      </div>
      <FavoriteToggle />
    </div>
  );
}
