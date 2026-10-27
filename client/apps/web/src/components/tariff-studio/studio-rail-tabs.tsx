import { XCloseIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useState } from "react";
import type { TariffTestsState } from "./use-tariff-tests";

/** True from the first time a flag is raised, so a piece opened once stays mounted. */
export function useOnceOpened(open: boolean): boolean {
  const [opened, setOpened] = useState(open);
  if (open && !opened) setOpened(true);
  return opened;
}

const tabClass = (active: boolean) =>
  cn(
    "ui-focus-ring relative flex h-10 shrink-0 items-center gap-1.5 rounded-sm px-2 text-sm whitespace-nowrap transition-colors",
    active
      ? "text-foreground after:bg-foreground font-medium after:absolute after:inset-x-2 after:-bottom-px after:h-0.5 after:rounded-full"
      : "text-foreground-muted hover:text-foreground",
  );

/**
 * The tab strip over a Studio's side panel: the Studio's own panes, the `tests` pane with the
 * last run's passed and total count, Nova at the end with a spinner while it answers, and the
 * button that closes the panel. The strip scrolls sideways when the panel is too narrow for it.
 */
export function StudioRailTabs<Tab extends string>({
  label,
  tabs,
  tab,
  onTab,
  tests,
  novaBusy,
  onClose,
}: {
  /** The tab list's accessible name. */
  label: string;
  /** The panes before Nova, in order. A pane valued `tests` carries the test count. */
  tabs: { value: Tab; label: string }[];
  tab: Tab | "nova";
  onTab: (tab: Tab | "nova") => void;
  tests: TariffTestsState;
  novaBusy: boolean;
  onClose: () => void;
}) {
  const t = useT();
  const total = tests.status === "ran" ? tests.passed + tests.failed : 0;
  const allPass = tests.status === "ran" && total > 0 && tests.failed === 0;

  return (
    <div className="flex shrink-0 items-center gap-1 border-b pr-1.5 pl-2">
      <div
        role="tablist"
        aria-label={label}
        className="flex min-w-0 flex-1 items-center gap-1 overflow-x-auto"
      >
        {tabs.map((item) => (
          <button
            key={item.value}
            type="button"
            role="tab"
            aria-selected={tab === item.value}
            onClick={() => onTab(item.value)}
            className={tabClass(tab === item.value)}
          >
            {item.label}
            {item.value === "tests" && tests.status === "ran" && total > 0 && (
              <span
                className={cn(
                  "rounded-badge text-2xs px-1.5 font-mono tabular-nums",
                  allPass
                    ? "bg-success-subtle text-success-subtle-foreground"
                    : "bg-danger-subtle text-danger-subtle-foreground",
                )}
              >
                {tests.passed}/{total}
              </span>
            )}
          </button>
        ))}
        <span className="flex-1" />
        <button
          type="button"
          role="tab"
          aria-selected={tab === "nova"}
          onClick={() => onTab("nova")}
          className={cn(tabClass(tab === "nova"), "fs-nova-tab")}
        >
          <NovaBadge size="xs" className="fs-nova-badge" />
          {t("Nova")}
          {novaBusy && <Spinner className="size-3" />}
        </button>
      </div>
      <Button
        type="button"
        variant="ghost"
        size="icon-sm"
        aria-label={t("Close panel")}
        onClick={onClose}
      >
        <XCloseIcon className="size-4" />
      </Button>
    </div>
  );
}
