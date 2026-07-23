import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { TabsList, TabsTab } from "@trenova/shared/components/ui/tabs";
import { cn } from "@trenova/shared/lib/utils";
import { CheckIcon, ChevronDownIcon } from "lucide-react";
import { useCallback, useLayoutEffect, useRef, useState } from "react";

export type OverflowTab = {
  value: string;
  label: string;
  icon?: React.ComponentType<{ className?: string }>;
  className?: string;
};

const TAB_GAP = 2;

const measureTabClassName =
  "flex h-9 shrink-0 items-center justify-center gap-1.5 border border-transparent px-[calc(--spacing(2.5)-1px)] text-base font-medium whitespace-nowrap sm:h-8 sm:text-sm [&_svg]:-mx-0.5 [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4.5 sm:[&_svg:not([class*='size-'])]:size-4";

type OverflowTabsListProps = {
  items: OverflowTab[];
  activeValue: string;
  onSelect: (value: string) => void;
  className?: string;
  moreLabel?: string;
};

export function OverflowTabsList({
  items,
  activeValue,
  onSelect,
  className,
  moreLabel = "More",
}: OverflowTabsListProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const measureRef = useRef<HTMLDivElement>(null);
  const [visibleCount, setVisibleCount] = useState(items.length);

  const recompute = useCallback(() => {
    const container = containerRef.current;
    const measure = measureRef.current;
    if (!container || !measure) {
      return;
    }

    const tabNodes = Array.from(measure.querySelectorAll<HTMLElement>("[data-measure-tab]"));
    const moreNode = measure.querySelector<HTMLElement>("[data-measure-more]");
    if (tabNodes.length === 0) {
      return;
    }

    const available = container.clientWidth;
    const widths = tabNodes.map((node) => node.offsetWidth);
    const totalAll = widths.reduce((sum, width) => sum + width, 0) + TAB_GAP * (widths.length - 1);

    if (totalAll <= available) {
      setVisibleCount(widths.length);
      return;
    }

    const moreWidth = (moreNode?.offsetWidth ?? 80) + TAB_GAP;
    let used = 0;
    let count = 0;
    for (const width of widths) {
      const next = used + (count > 0 ? TAB_GAP : 0) + width;
      if (next + moreWidth > available) {
        break;
      }
      used = next;
      count += 1;
    }
    setVisibleCount(count);
  }, []);

  useLayoutEffect(() => {
    recompute();
    const container = containerRef.current;
    if (!container) {
      return;
    }
    const observer = new ResizeObserver(() => recompute());
    observer.observe(container);
    return () => observer.disconnect();
  }, [recompute, items]);

  const visibleItems = items.slice(0, visibleCount);
  const overflowItems = items.slice(visibleCount);
  const hiddenActive = overflowItems.find((item) => item.value === activeValue);

  return (
    <div ref={containerRef} className={cn("relative min-w-0", className)}>
      <TabsList variant="underline" className="max-w-full">
        {visibleItems.map((tab) => (
          <TabsTab
            key={tab.value}
            value={tab.value}
            className={cn("hover:text-foreground", tab.className)}
          >
            {tab.icon ? <tab.icon className="mr-1 size-4" /> : null}
            {tab.label}
          </TabsTab>
        ))}
        {overflowItems.length > 0 ? (
          <DropdownMenu>
            <DropdownMenuTrigger
              className={cn(
                measureTabClassName,
                "cursor-pointer rounded-md text-muted-foreground transition-colors outline-none hover:bg-accent hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring",
                hiddenActive && "text-foreground",
              )}
              aria-label={hiddenActive ? `${hiddenActive.label} (more tabs)` : `${moreLabel} tabs`}
            >
              {hiddenActive?.icon ? <hiddenActive.icon className="mr-1 size-4" /> : null}
              {hiddenActive ? hiddenActive.label : moreLabel}
              <ChevronDownIcon className="size-3.5" />
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-auto min-w-44">
              {overflowItems.map((tab) => (
                <DropdownMenuItem
                  key={tab.value}
                  title={tab.label}
                  onClick={() => onSelect(tab.value)}
                  className={tab.className}
                  startContent={tab.icon ? <tab.icon className="size-4" /> : undefined}
                  endContent={
                    tab.value === activeValue ? <CheckIcon className="size-4" /> : undefined
                  }
                />
              ))}
            </DropdownMenuContent>
          </DropdownMenu>
        ) : null}
      </TabsList>

      <div
        ref={measureRef}
        aria-hidden
        className="pointer-events-none invisible absolute inset-x-0 top-0 -z-10 overflow-hidden"
      >
        <div className="flex w-max items-center" style={{ columnGap: TAB_GAP }}>
          {items.map((tab) => (
            <div key={tab.value} data-measure-tab className={measureTabClassName}>
              {tab.icon ? <tab.icon className="mr-1 size-4" /> : null}
              {tab.label}
            </div>
          ))}
          <div data-measure-more className={measureTabClassName}>
            {moreLabel}
            <ChevronDownIcon className="size-3.5" />
          </div>
        </div>
      </div>
    </div>
  );
}
