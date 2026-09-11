import { useT } from "@trenova/shared/i18n/use-t";
import { BetaTag } from "@/components/beta-tag";
import { preloadCommandPalette } from "@/components/command-palette/command-palette-mount";
import type {
  ModuleAttention,
  SidebarModuleView,
  SidebarPageItem,
} from "@/components/navigation/sidebar-model";
import {
  WorkspaceGroupLabel,
  WorkspaceNavRow,
  WorkspaceRowLabel,
} from "@/components/navigation/workspace-primitives";
import type { AttentionTone } from "@/config/attention-rows";
import { Button } from "@trenova/shared/components/ui/button";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { formatShortcut } from "@trenova/shared/lib/shortcuts";
import { cn } from "@trenova/shared/lib/utils";
import { useCommandPaletteStore } from "@/stores/command-palette-store";
import { SearchIcon } from "lucide-react";

const ATTENTION_PILL_CLASSES: Record<AttentionTone, string> = {
  default: "bg-info text-white",
  warning: "bg-warning text-warning-foreground",
  destructive: "bg-destructive text-white",
};

const EMPTY_ATTENTION: ReadonlyMap<string, ModuleAttention> = new Map();

export function formatAttentionCount(count: number): string {
  return count > 99 ? "99+" : String(count);
}

/**
 * How much work waits behind a link, as a pill on the module's row or icon.
 */
export function AttentionCountBadge({
  attention,
  className,
}: {
  attention: ModuleAttention | undefined;
  className?: string;
}) {
  if (!attention || attention.count <= 0) {
    return null;
  }

  return (
    <span
      aria-label={`${attention.count} items need attention`}
      className={cn(
        "inline-flex h-4 min-w-4 items-center justify-center rounded-full px-1 text-2xs leading-none font-semibold tabular-nums",
        ATTENTION_PILL_CLASSES[attention.tone],
        className,
      )}
    >
      {formatAttentionCount(attention.count)}
    </span>
  );
}

export function SearchTrigger({
  compact = false,
  className,
  tooltipSide = "right",
}: {
  compact?: boolean;
  className?: string;
  tooltipSide?: "right" | "bottom";
}) {
  const t = useT();

  const setOpen = useCommandPaletteStore((state) => state.setOpen);
  const preload = () => void preloadCommandPalette();
  const shortcut = formatShortcut("K");

  if (compact) {
    return (
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              aria-label={t("Search")}
              onClick={() => setOpen(true)}
              onPointerEnter={preload}
              onFocus={preload}
              className={cn("text-muted-foreground hover:text-foreground", className)}
            />
          }
        >
          <SearchIcon className="size-4" strokeWidth={1.75} />
        </TooltipTrigger>
        <TooltipContent side={tooltipSide} sideOffset={10}>
          <span className="flex items-center gap-2">
            {t("Search")}
            <Kbd>{shortcut}</Kbd>
          </span>
        </TooltipContent>
      </Tooltip>
    );
  }

  return (
    <button
      type="button"
      onClick={() => setOpen(true)}
      onPointerEnter={preload}
      onFocus={preload}
      className={cn(
        "border-border bg-background text-muted-foreground hover:border-ring/40 hover:text-foreground flex h-7 w-full min-w-0 items-center gap-2 rounded-md border px-2 text-xs transition-colors",
        className,
      )}
    >
      <SearchIcon className="size-3.5 shrink-0" strokeWidth={1.75} />
      <span className="flex-1 truncate text-left">{t("Search or jump to…")}</span>
      <Kbd>{shortcut}</Kbd>
    </button>
  );
}

function PageLink({
  item,
  active,
  sub,
  attention,
  onNavigate,
}: {
  item: SidebarPageItem;
  active: boolean;
  sub: boolean;
  attention: ModuleAttention | undefined;
  onNavigate?: () => void;
}) {
  return (
    <WorkspaceNavRow
      to={item.path}
      active={active}
      disabled={item.disabled}
      sub={sub}
      onClick={onNavigate}
    >
      <WorkspaceRowLabel>{item.label}</WorkspaceRowLabel>
      <AttentionCountBadge attention={attention} />
      {item.includeBetaTag && <BetaTag className="ml-auto" />}
    </WorkspaceNavRow>
  );
}

/**
 * A module's pages the way the sidebar lists them: working pages first,
 * grouped when the module groups them, the configuration catalogue last
 * under its own label. A page the person watches carries its count.
 */
export function ModulePageList({
  view,
  activePath,
  attentionByPath = EMPTY_ATTENTION,
  onNavigate,
  className,
}: {
  view: SidebarModuleView;
  activePath: string | null;
  attentionByPath?: ReadonlyMap<string, ModuleAttention>;
  onNavigate?: () => void;
  className?: string;
}) {
  const t = useT();

  const hasPages = view.sections.some((section) => section.items.length > 0);
  const hasConfiguration = view.configuration.length > 0;

  if (!hasPages && !hasConfiguration) {
    return <p className="text-muted-foreground px-2.5 py-4 text-xs">{t("This area has no pages yet.")}</p>;
  }

  return (
    <div className={cn("flex flex-col gap-0.5", className)}>
      {view.sections.map((section, index) => (
        <div key={section.id} className="flex flex-col gap-0.5">
          {section.label !== null && <WorkspaceGroupLabel>{section.label}</WorkspaceGroupLabel>}
          {section.label === null && index > 0 && <div className="h-1.5" aria-hidden />}
          {section.items.map((item) => (
            <PageLink
              key={item.id}
              item={item}
              active={item.path === activePath}
              sub={section.label !== null}
              attention={attentionByPath.get(item.path)}
              onNavigate={onNavigate}
            />
          ))}
        </div>
      ))}
      {hasConfiguration && (
        <div className={cn("flex flex-col gap-0.5", hasPages && "border-border mt-2.5 border-t")}>
          <WorkspaceGroupLabel>{t("Configuration")}</WorkspaceGroupLabel>
          {view.configuration.map((item) => (
            <PageLink
              key={item.id}
              item={item}
              active={item.path === activePath}
              sub
              attention={attentionByPath.get(item.path)}
              onNavigate={onNavigate}
            />
          ))}
        </div>
      )}
    </div>
  );
}
