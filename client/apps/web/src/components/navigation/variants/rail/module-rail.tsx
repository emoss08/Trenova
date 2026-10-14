import { useT } from "@trenova/shared/i18n/use-t";
import { OrgSwitcher } from "@/components/navigation/org-switcher";
import {
  AttentionCountBadge,
  ModulePageList,
} from "@/components/navigation/sidebar-chrome";
import {
  moduleDisplayLabel,
  type ModuleAttention,
  type SidebarModuleView,
} from "@/components/navigation/sidebar-model";
import { UserMenu } from "@/components/navigation/user-menu";
import { NotificationSheet } from "@/components/notification-center/notification-sheet";
import { preloadCommandPalette } from "@/components/command-palette/command-palette-mount";
import type { ModuleId, NavModule } from "@/config/navigation.types";
import { useSidebarNavigation } from "@/hooks/use-sidebar-navigation";
import { useCommandPaletteStore } from "@/stores/command-palette-store";
import { SearchLgIcon } from "@trenova/shared/components/icons";
import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@trenova/shared/components/ui/hover-card";
import { Kbd, KbdGroup } from "@trenova/shared/components/ui/kbd";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { formatShortcut } from "@trenova/shared/lib/shortcuts";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
import { Link } from "react-router";
import { GO_CHORD_KEY, MODULE_GO_KEYS } from "./rail-shortcuts";

export const RAIL_WIDTH_CLASS = "w-12";

const TILE_CLASS =
  "ui-focus-ring relative flex size-8 shrink-0 items-center justify-center rounded-md transition-colors outline-none";
const TILE_REST_CLASS = "text-foreground-muted hover:bg-surface-hover hover:text-foreground";
const TILE_ACTIVE_CLASS = "bg-nav-active text-nav-active-foreground";

function ChordHint({ letter }: { letter: string }) {
  return (
    <KbdGroup className="ml-2">
      <Kbd>{GO_CHORD_KEY}</Kbd>
      <Kbd>{letter}</Kbd>
    </KbdGroup>
  );
}

function TileLabel({ label, letter }: { label: string; letter?: string }) {
  return (
    <span className="flex items-center">
      {label}
      {letter && <ChordHint letter={letter} />}
    </span>
  );
}

/**
 * The mark that says which module you are in: an ink bar on the rail's
 * edge, the way an active line is marked in a gutter.
 */
function ActiveMark() {
  return (
    <span
      aria-hidden
      className="bg-ink absolute top-1.5 bottom-1.5 -left-2 w-0.5 rounded-full ease-settle animate-in fade-in-0 zoom-in-50"
    />
  );
}

function RailTile({
  to,
  label,
  icon: Icon,
  active,
  attention,
  letter,
  flyout,
}: {
  to: string;
  label: string;
  icon: NavModule["icon"];
  active: boolean;
  attention?: ModuleAttention;
  letter?: string;
  /** The module's pages, shown beside the rail on hover while the panel is folded. */
  flyout?: ReactNode;
}) {
  const link = (
    <Link
      to={to}
      aria-label={label}
      aria-current={active ? "page" : undefined}
      className={cn(TILE_CLASS, active ? TILE_ACTIVE_CLASS : TILE_REST_CLASS)}
    >
      <Icon className="size-4" strokeWidth={active ? 2 : 1.75} />
      <AttentionCountBadge
        attention={attention}
        className="ring-sidebar absolute -top-1 -right-1 h-3.5 min-w-3.5 px-0.5 text-3xs ring-2"
      />
      {active && <ActiveMark />}
    </Link>
  );

  if (flyout) {
    return (
      <HoverCard>
        <HoverCardTrigger render={link} />
        <HoverCardContent side="right" align="start" sideOffset={8} className="w-60 p-1.5">
          <div className="flex items-center justify-between px-2 pt-1 pb-1.5">
            <span className="text-sm font-semibold">{label}</span>
            {letter && <ChordHint letter={letter} />}
          </div>
          {flyout}
        </HoverCardContent>
      </HoverCard>
    );
  }

  return (
    <Tooltip>
      <TooltipTrigger render={link} />
      <TooltipContent side="right" sideOffset={10}>
        <TileLabel label={label} letter={letter} />
      </TooltipContent>
    </Tooltip>
  );
}

function RailButton({
  label,
  hint,
  onClick,
  onPointerEnter,
  children,
}: {
  label: string;
  hint?: ReactNode;
  onClick: () => void;
  onPointerEnter?: () => void;
  children: ReactNode;
}) {
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <button
            type="button"
            aria-label={label}
            onClick={onClick}
            onPointerEnter={onPointerEnter}
            onFocus={onPointerEnter}
            className={cn(TILE_CLASS, TILE_REST_CLASS)}
          />
        }
      >
        {children}
      </TooltipTrigger>
      <TooltipContent side="right" sideOffset={10}>
        <span className="flex items-center">
          {label}
          {hint}
        </span>
      </TooltipContent>
    </Tooltip>
  );
}

function DomainRule() {
  return <div aria-hidden className="bg-border my-1.5 h-px w-5 shrink-0" />;
}

function moduleTile({
  module,
  view,
  active,
  attention,
  panelFolded,
  activePath,
  pageAttention,
}: {
  module: NavModule;
  view: SidebarModuleView | undefined;
  active: boolean;
  attention: ModuleAttention | undefined;
  panelFolded: boolean;
  activePath: string | null;
  pageAttention: ReadonlyMap<string, ModuleAttention>;
}) {
  const hasPages =
    view !== undefined &&
    (view.sections.some((section) => section.items.length > 0) || view.configuration.length > 0);
  return (
    <RailTile
      key={module.id}
      to={view?.landingPath ?? module.basePath}
      label={moduleDisplayLabel(module)}
      icon={module.icon}
      active={active}
      attention={attention}
      letter={MODULE_GO_KEYS[module.id]}
      flyout={
        panelFolded && hasPages && view ? (
          <ModulePageList view={view} activePath={activePath} attentionByPath={pageAttention} />
        ) : undefined
      }
    />
  );
}

/**
 * The map of the product, always on screen: one tile per module in its
 * domain, the one you are in marked on the rail's edge, how much work waits
 * behind each as a count on its corner. Who you work as sits at the top; what
 * applies everywhere (search, notifications, settings, you) sits at the
 * bottom. It never scrolls the page and never folds; the panel beside it
 * does.
 */
export function ModuleRail({ panelFolded, className }: { panelFolded: boolean; className?: string }) {
  const t = useT();
  const { home, settings, domains, views, activeModule, activePath, attention, pageAttention } =
    useSidebarNavigation();
  const activeModuleId: ModuleId | null = activeModule?.id ?? null;
  const openPalette = useCommandPaletteStore((state) => state.setOpen);

  const tileFor = (module: NavModule) =>
    moduleTile({
      module,
      view: views.get(module.id),
      active: module.id === activeModuleId,
      attention: attention.get(module.id),
      panelFolded,
      activePath,
      pageAttention,
    });

  return (
    <div
      className={cn(
        "bg-sidebar border-border flex h-full shrink-0 flex-col items-center border-r",
        RAIL_WIDTH_CLASS,
        className,
      )}
    >
      <div className="flex h-11 shrink-0 items-center justify-center">
        <OrgSwitcher compact />
      </div>

      <nav
        aria-label={t("Modules")}
        className="flex min-h-0 w-full flex-1 flex-col items-center gap-0.5 overflow-y-auto px-2 py-1 [scrollbar-width:none]"
      >
        {home && tileFor(home)}
        {domains.map((domain) => (
          <div key={domain.id} className="flex flex-col items-center gap-0.5">
            <DomainRule />
            {domain.modules.map(tileFor)}
          </div>
        ))}
      </nav>

      <div className="flex shrink-0 flex-col items-center gap-0.5 pt-1 pb-2">
        <DomainRule />
        {panelFolded && (
          <RailButton
            label={t("Search")}
            hint={<Kbd className="ml-2">{formatShortcut("K")}</Kbd>}
            onClick={() => openPalette(true)}
            onPointerEnter={preloadCommandPalette}
          >
            <SearchLgIcon className="size-4" strokeWidth={1.75} />
          </RailButton>
        )}
        <NotificationSheet
          className={cn(TILE_CLASS, TILE_REST_CLASS, "size-8 w-8")}
          iconClassName="size-4"
          tooltipSide="right"
        />
        {settings && tileFor(settings)}
        <div className="flex h-8 items-center justify-center">
          <UserMenu compact />
        </div>
      </div>
    </div>
  );
}
