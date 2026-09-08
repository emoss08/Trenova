import { CustomizeSidebarDialog } from "@/components/navigation/customize-sidebar-dialog";
import { AttentionCountBadge } from "@/components/navigation/sidebar-chrome";
import {
  moduleDisplayLabel,
  type ModuleAttention,
  type SidebarDomain,
  type SidebarModuleView,
} from "@/components/navigation/sidebar-model";
import { ModuleTile } from "@/components/navigation/workspace-primitives";
import type { ModuleId } from "@/config/navigation.types";
import { useSidebarNavigation } from "@/hooks/use-sidebar-navigation";
import { SIDEBAR_SECTION_KEYS } from "@/lib/graphql/sidebar-preferences";
import { queries } from "@/lib/queries";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { formatShortcut } from "@trenova/shared/lib/shortcuts";
import { cn } from "@trenova/shared/lib/utils";
import { useRecentPages } from "@/stores/recent-pages-store";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useQuery } from "@tanstack/react-query";
import { ClockIcon, HomeIcon, SettingsIcon, SlidersHorizontalIcon, StarIcon } from "lucide-react";
import { useState, type ReactElement } from "react";
import { Link, useLocation } from "react-router";

const PINNED_LIMIT = 5;
const RECENT_LIMIT = 5;

function ModuleItem({
  view,
  current,
  attention,
  onNavigate,
}: {
  view: SidebarModuleView;
  current: boolean;
  attention: ModuleAttention | undefined;
  onNavigate: () => void;
}) {
  return (
    <Link
      to={view.landingPath}
      onClick={onNavigate}
      aria-current={current ? "page" : undefined}
      className="hover:bg-muted focus-visible:bg-muted flex items-start gap-2 rounded-md px-1.5 py-1.5 transition-colors outline-none"
    >
      <ModuleTile icon={view.module.icon} active={current} size="sm" />
      <span className="flex min-w-0 flex-col">
        <span className="flex items-center gap-1.5 text-sm leading-tight font-semibold">
          <span className="truncate">{moduleDisplayLabel(view.module)}</span>
          <AttentionCountBadge attention={attention} className="h-3.5 min-w-3.5 text-3xs" />
        </span>
        {view.module.description && (
          <span className="text-muted-foreground mt-0.5 line-clamp-2 text-2xs leading-snug">
            {view.module.description}
          </span>
        )}
      </span>
    </Link>
  );
}

function DomainColumn({
  domain,
  views,
  attention,
  activeModuleId,
  onNavigate,
}: {
  domain: SidebarDomain;
  views: ReadonlyMap<ModuleId, SidebarModuleView>;
  attention: ReadonlyMap<ModuleId, ModuleAttention>;
  activeModuleId: ModuleId | null;
  onNavigate: () => void;
}) {
  return (
    <div className="border-border flex min-w-0 flex-col gap-0.5 px-2 pt-3 pb-2.5 first:pl-3 not-first:border-l">
      <span className="text-muted-foreground px-1.5 pb-1.5 text-2xs font-semibold tracking-wide select-none">
        {domain.label}
      </span>
      {domain.modules.map((module) => {
        const view = views.get(module.id);
        if (!view) return null;
        return (
          <ModuleItem
            key={module.id}
            view={view}
            current={module.id === activeModuleId}
            attention={attention.get(module.id)}
            onNavigate={onNavigate}
          />
        );
      })}
    </div>
  );
}

function SideRow({
  to,
  icon,
  current = false,
  onNavigate,
  children,
}: {
  to: string;
  icon: ReactElement;
  current?: boolean;
  onNavigate: () => void;
  children: string;
}) {
  return (
    <Link
      to={to}
      onClick={onNavigate}
      aria-current={current ? "page" : undefined}
      className={cn(
        "hover:bg-muted focus-visible:bg-muted flex h-6 items-center gap-2 rounded-md px-1.5 text-sm transition-colors outline-none",
        current && "bg-nav-active text-nav-active-foreground font-semibold",
      )}
    >
      {icon}
      <span className="truncate">{children}</span>
    </Link>
  );
}

function SideHeading({ children }: { children: string }) {
  return (
    <span className="text-muted-foreground px-1.5 pb-1 text-2xs font-semibold tracking-wide select-none">
      {children}
    </span>
  );
}

function SideEmpty({ children }: { children: string }) {
  return <p className="text-muted-foreground/80 px-1.5 py-1 text-2xs leading-snug">{children}</p>;
}

function ShortcutsColumn({
  showPinned,
  onNavigate,
}: {
  showPinned: boolean;
  onNavigate: () => void;
}) {
  const { pathname } = useLocation();
  const { data: favorites } = useQuery({ ...queries.pageFavorite.all(), enabled: showPinned });
  const organizationId = useAuthStore((state) => state.user?.currentOrganizationId);
  const recent = useRecentPages(organizationId);
  const pinned = (favorites ?? []).slice(0, PINNED_LIMIT);
  const recentPages = recent.slice(0, RECENT_LIMIT);

  return (
    <div className="bg-sidebar border-border flex flex-col gap-3 border-l px-2.5 pt-3 pb-2.5">
      <SideRow
        to="/"
        current={pathname === "/"}
        onNavigate={onNavigate}
        icon={<HomeIcon className="size-3.5 shrink-0" strokeWidth={1.75} />}
      >
        Home
      </SideRow>
      {showPinned && (
        <div className="flex flex-col gap-0.5">
          <SideHeading>Pinned</SideHeading>
          {pinned.length === 0 ? (
            <SideEmpty>Star a page from its header to keep it here.</SideEmpty>
          ) : (
            pinned.map((favorite) => (
              <SideRow
                key={favorite.id}
                to={favorite.pageUrl}
                onNavigate={onNavigate}
                icon={<StarIcon className="size-3 shrink-0 fill-amber-400 text-amber-400" />}
              >
                {favorite.pageTitle}
              </SideRow>
            ))
          )}
        </div>
      )}
      <div className="flex flex-col gap-0.5">
        <SideHeading>Recent</SideHeading>
        {recentPages.length === 0 ? (
          <SideEmpty>Pages you open show up here.</SideEmpty>
        ) : (
          recentPages.map((page) => (
            <SideRow
              key={page.path}
              to={page.path}
              onNavigate={onNavigate}
              icon={
                <ClockIcon className="text-muted-foreground size-3.5 shrink-0" strokeWidth={1.75} />
              }
            >
              {page.title}
            </SideRow>
          ))
        )}
      </div>
    </div>
  );
}

/**
 * The map of the product: every module the person may open, grouped by
 * domain, one line each, plus the pages they pinned or just left. It is the
 * only place the whole product is listed, so it stays calm: no page lists,
 * no nesting, one click to any module.
 */
export function ModulesMenu({
  trigger,
  side = "bottom",
  align = "start",
  sideOffset = 6,
}: {
  trigger: ReactElement;
  side?: "bottom" | "right";
  align?: "start" | "end";
  sideOffset?: number;
}) {
  const [open, setOpen] = useState(false);
  const close = () => setOpen(false);
  const { settings, domains, views, activeModule, attention, hiddenSections } =
    useSidebarNavigation();
  const activeModuleId = activeModule?.id ?? null;
  const columnCount = Math.max(domains.length, 1);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger render={trigger} />
      <PopoverContent
        side={side}
        align={align}
        sideOffset={sideOffset}
        className="w-[min(52rem,calc(100vw-1.5rem))] gap-0 overflow-hidden rounded-lg p-0"
      >
        <nav
          aria-label="Modules"
          className="grid"
          style={{ gridTemplateColumns: `repeat(${columnCount}, minmax(0, 1fr)) 13rem` }}
        >
          {domains.map((domain) => (
            <DomainColumn
              key={domain.id}
              domain={domain}
              views={views}
              attention={attention}
              activeModuleId={activeModuleId}
              onNavigate={close}
            />
          ))}
          <ShortcutsColumn
            showPinned={!hiddenSections.has(SIDEBAR_SECTION_KEYS.favorites)}
            onNavigate={close}
          />
        </nav>
        <div className="border-border text-muted-foreground flex items-center gap-4 border-t px-3 py-2 text-xs">
          {settings && (
            <Link
              to={settings.basePath}
              onClick={close}
              className={cn(
                "text-foreground flex items-center gap-1.5 rounded-sm transition-colors outline-none",
                "hover:text-nav-active-foreground focus-visible:ring-ring/50 focus-visible:ring-2",
              )}
            >
              <SettingsIcon className="size-3" strokeWidth={1.75} />
              Organization settings
            </Link>
          )}
          <CustomizeSidebarDialog
            trigger={
              <button
                type="button"
                className="text-foreground hover:text-nav-active-foreground focus-visible:ring-ring/50 flex items-center gap-1.5 rounded-sm transition-colors outline-none focus-visible:ring-2"
              >
                <SlidersHorizontalIcon className="size-3" strokeWidth={1.75} />
                Customize navigation
              </button>
            }
          />
          <span className="ml-auto hidden items-center gap-1.5 sm:flex">
            Type a page name in <Kbd>{formatShortcut("K")}</Kbd> to jump anywhere
          </span>
        </div>
      </PopoverContent>
    </Popover>
  );
}
