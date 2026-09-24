import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  ActivityIcon,
  BotIcon,
  BrainIcon,
  LayoutDashboardIcon,
  PlugZapIcon,
  PuzzleIcon,
  ShieldCheckIcon,
  TargetIcon,
} from "lucide-react";
import { AnimatePresence, m, useReducedMotion } from "motion/react";
import type { AIControlTab } from "../ai-control-tabs";
import type { ActivityView, RailItem } from "./rail-items";

const ICONS: Record<AIControlTab, typeof BotIcon> = {
  overview: LayoutDashboardIcon,
  agents: BotIcon,
  providers: PlugZapIcon,
  extensions: PuzzleIcon,
  memory: BrainIcon,
  safety: ShieldCheckIcon,
  quality: TargetIcon,
  activity: ActivityIcon,
};

type ControlRailProps = {
  items: RailItem[];
  active: AIControlTab;
  activeView: ActivityView;
  onSelect: (tab: AIControlTab, view?: ActivityView) => void;
};

/**
 * The page's sections down its side, each saying what it holds before it
 * is opened: how many agents are on, whether a provider is connected, and
 * what is waiting on a person. Tabs across the top said only their names,
 * and every section then repeated its own name and count in a heading.
 *
 * The selection slides between rows rather than cutting; a row that needs
 * attention carries a warning dot. On a narrow screen the rail lies flat
 * above the content and scrolls sideways.
 */
export function ControlRail({ items, active, activeView, onSelect }: ControlRailProps) {
  const t = useT();
  const reduceMotion = useReducedMotion();

  return (
    <nav
      aria-label={t("AI Control sections")}
      className="scrollbar-overlay -mx-4 flex gap-1 overflow-x-auto px-4 md:mx-0 md:flex-col md:overflow-visible md:px-0"
    >
      {items.map((item) => {
        const Icon = ICONS[item.tab];
        const isActive = item.tab === active;
        return (
          <div key={item.tab} className="flex shrink-0 flex-col md:shrink">
            <button
              type="button"
              aria-current={isActive ? "page" : undefined}
              onClick={() => onSelect(item.tab)}
              className={cn(
                "ui-focus-ring relative flex items-center gap-2.5 rounded-md px-2.5 py-2 text-left transition-colors",
                isActive ? "text-foreground" : "text-muted-foreground hover:text-foreground",
              )}
            >
              {isActive && (
                <m.span
                  layoutId="ai-control-rail-active"
                  aria-hidden
                  className="bg-surface-selected absolute inset-0 rounded-md"
                  transition={
                    reduceMotion ? { duration: 0 } : { type: "spring", stiffness: 500, damping: 40 }
                  }
                />
              )}
              <Icon className="relative size-4 shrink-0" aria-hidden />
              <span className="relative flex min-w-0 flex-col">
                <span className="text-sm font-medium">{t(LABELS[item.tab].label)}</span>
                {item.status !== "" && (
                  <span
                    className={cn(
                      "flex items-center gap-1 text-xs",
                      item.attention ? "text-warning-foreground" : "text-muted-foreground",
                    )}
                  >
                    {item.attention && (
                      <span aria-hidden className="bg-warning size-1.5 shrink-0 rounded-full" />
                    )}
                    <span className="truncate">{item.status}</span>
                  </span>
                )}
              </span>
            </button>

            <AnimatePresence initial={false}>
              {isActive && item.children.length > 1 && (
                <m.div
                  key="children"
                  initial={reduceMotion ? false : { opacity: 0, height: 0 }}
                  animate={{ opacity: 1, height: "auto" }}
                  exit={{ opacity: 0, height: 0 }}
                  transition={{ duration: 0.18, ease: [0.16, 1, 0.3, 1] }}
                  className="flex gap-0.5 overflow-hidden md:flex-col md:pl-7"
                >
                  {item.children.map((child) => {
                    const selected = child.view === activeView;
                    return (
                      <button
                        key={child.view}
                        type="button"
                        aria-current={selected ? "true" : undefined}
                        onClick={() => onSelect(item.tab, child.view)}
                        className={cn(
                          "ui-focus-ring rounded-md px-2.5 py-1 text-left text-xs transition-colors",
                          selected
                            ? "text-foreground font-medium"
                            : "text-muted-foreground hover:text-foreground",
                        )}
                      >
                        {child.label}
                      </button>
                    );
                  })}
                </m.div>
              )}
            </AnimatePresence>
          </div>
        );
      })}
    </nav>
  );
}

const LABELS: Record<AIControlTab, { label: string }> = {
  overview: { label: "Overview" },
  agents: { label: "Agents" },
  providers: { label: "Providers" },
  extensions: { label: "Extensions" },
  memory: { label: "Memory" },
  safety: { label: "Safety" },
  quality: { label: "Quality" },
  activity: { label: "Activity" },
};
