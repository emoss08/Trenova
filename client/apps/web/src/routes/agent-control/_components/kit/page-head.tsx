import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useLayoutEffect, useRef, useState } from "react";
import type { AIControlTab } from "../../ai-control-tabs";
import type { TabCount } from "../rail-items";
import { Ic } from "./ic";

const LABELS: Record<AIControlTab, { label: string }> = {
  overview: { label: "Overview" },
  agents: { label: "Agents" },
  providers: { label: "Providers" },
  extensions: { label: "Extensions" },
  memory: { label: "Memory" },
  retrieval: { label: "Retrieval" },
  safety: { label: "Safety" },
  quality: { label: "Quality" },
  activity: { label: "Activity" },
  audit: { label: "Audit trail" },
};

type PageHeadProps = {
  tabs: AIControlTab[];
  active: AIControlTab;
  counts: Partial<Record<AIControlTab, TabCount | null>>;
  onSelect: (tab: AIControlTab) => void;
};

function typing(target: EventTarget | null): boolean {
  return (
    target instanceof HTMLElement &&
    (target.isContentEditable || ["INPUT", "TEXTAREA", "SELECT"].includes(target.tagName))
  );
}

/** The page's title and its tabs, with what each tab holds and a line under the open one. */
export function PageHead({ tabs, active, counts, onSelect }: PageHeadProps) {
  const t = useT();
  const bar = useRef<HTMLDivElement>(null);
  const [indicator, setIndicator] = useState<{ left: number; width: number } | null>(null);

  useLayoutEffect(() => {
    const element = bar.current?.querySelector<HTMLElement>(".tab.on");
    if (element) {
      setIndicator({ left: element.offsetLeft, width: element.offsetWidth });
    }
  }, [active, counts, tabs]);

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.metaKey || event.ctrlKey || event.altKey || typing(event.target)) {
        return;
      }
      if (document.querySelector("[role=dialog]")) {
        return;
      }
      const index = Number(event.key) - 1;
      if (Number.isInteger(index) && index >= 0 && index < Math.min(tabs.length, 9)) {
        event.preventDefault();
        onSelect(tabs[index]);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onSelect, tabs]);

  return (
    <div className="ph">
      <div className="ph-r">
        <div className="ph-t">
          <span className="crumb">
            {t("Settings")}
            <Ic n="chevR" s={11} />
            {t("AI & Automation")}
          </span>
          <h1>{t("AI control")}</h1>
        </div>
      </div>
      <div className="tabs" ref={bar} role="tablist">
        {tabs.map((tab, index) => {
          const count = counts[tab];
          return (
            <button
              key={tab}
              type="button"
              role="tab"
              aria-selected={tab === active}
              className={cn("tab", tab === active && "on")}
              onClick={() => onSelect(tab)}
            >
              {t(LABELS[tab].label)}
              {count && <em className={cn("mono", count.tone)}>{count.text}</em>}
              {index < 9 && <span className="kbd">{index + 1}</span>}
            </button>
          );
        })}
        {indicator && (
          <span className="tab-ind" style={{ left: indicator.left, width: indicator.width }} />
        )}
      </div>
    </div>
  );
}
