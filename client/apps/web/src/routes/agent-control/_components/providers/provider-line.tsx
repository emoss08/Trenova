import type { AIProviderRow, AIProviderUsageDay } from "@/lib/graphql/ai-provider";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { DragEvent } from "react";
import { Ic } from "../kit/ic";
import { Switch } from "../kit/layout";
import { Mark } from "../kit/marks";
import { formatLatency, type LiveState } from "./provider-model";

/** What the chain shows of a provider's week: from the usage summary and its daily series. */
export type ProviderWeek = {
  calls: number;
  failed: number;
  latencyP50Ms: number;
  tokens: number;
  days: readonly AIProviderUsageDay[];
};

type ProviderLineProps = {
  provider: AIProviderRow;
  index: number;
  open: boolean;
  live: LiveState;
  needsKey: boolean;
  firsts: number;
  week: ProviderWeek | null;
  dragging: boolean;
  canReorder: boolean;
  canTest: boolean;
  canToggle: boolean;
  onOpen: (focusKey: boolean) => void;
  onTest: () => void;
  onToggle: (enabled: boolean) => void;
  onDragStart: () => void;
  onDragOver: () => void;
  onDragEnd: () => void;
};

/** One provider in the chain: its place, what it takes first, its week, and its switch. */
export function ProviderLine({
  provider,
  index,
  open,
  live,
  needsKey,
  firsts,
  week,
  dragging,
  canReorder,
  canTest,
  canToggle,
  onOpen,
  onTest,
  onToggle,
  onDragStart,
  onDragOver,
  onDragEnd,
}: ProviderLineProps) {
  const t = useT();
  const ok = week && week.calls > 0 ? 1 - week.failed / week.calls : null;
  const most = week ? Math.max(1, ...week.days.map((day) => day.calls)) : 1;

  return (
    <li
      id={`pv-${provider.id}`}
      className={cn("pl", open && "sel", !provider.enabled && "dim", dragging && "drg")}
      draggable={canReorder}
      onDragStart={(event: DragEvent<HTMLLIElement>) => {
        event.dataTransfer.effectAllowed = "move";
        event.dataTransfer.setData("text/plain", provider.id);
        onDragStart();
      }}
      onDragOver={(event) => {
        event.preventDefault();
        onDragOver();
      }}
      onDragEnd={onDragEnd}
      onClick={() => onOpen(false)}
    >
      <span className="pl-n mono">
        <Ic n="grip" s={11} />
        {String(index + 1).padStart(2, "0")}
      </span>
      <span className="pc-mk">
        <Mark provider={provider} s={26} />
        <i className={cn("pc-dot sm", live)} />
      </span>
      <button
        type="button"
        className="pl-t"
        aria-expanded={open}
        onClick={(event) => {
          event.stopPropagation();
          onOpen(false);
        }}
      >
        <b>{provider.name}</b>
        <span className="mono">{provider.model}</span>
      </button>
      <span className="pl-tg">
        {needsKey ? (
          <span className="tg w">{t("Needs a key")}</span>
        ) : !provider.enabled ? (
          <span className="tg">{t("Off")}</span>
        ) : firsts > 0 ? (
          <span className="pl-f">{t("first for {0}", firsts)}</span>
        ) : (
          <span className="pl-f dim">{t("backup")}</span>
        )}
        {provider.trusted && (
          <span className="tg b" title={t("Trusted")}>
            <Ic n="shield" s={10} />
          </span>
        )}
      </span>
      <span className="pl-st">
        {week && week.calls > 0 && ok !== null ? (
          <>
            <span className="pc-bars sm" aria-hidden>
              {week.days.map((day) => (
                <i key={day.day} style={{ height: 3 + (day.calls / most) * 13 }}>
                  {day.failed > 0 && day.calls > 0 && (
                    <u style={{ height: `${Math.max(3, (day.failed / day.calls) * 100)}%` }} />
                  )}
                </i>
              ))}
            </span>
            <span className={cn("mono pl-ok", ok < 0.98 && "t-w")}>{`${(ok * 100).toFixed(1)}%`}</span>
            <span className="mono pl-ms">{formatLatency(week.latencyP50Ms)}</span>
          </>
        ) : (
          <span className="pl-none">{t("No calls this week")}</span>
        )}
      </span>
      <span className="pc-c" onClick={(event) => event.stopPropagation()}>
        {needsKey ? (
          <button type="button" className="btn sm" onClick={() => onOpen(true)}>
            <Ic n="key" s={12} />
            {t("Add key")}
          </button>
        ) : (
          <button
            type="button"
            className="ib"
            title={t("Test connection")}
            aria-label={t("Test connection")}
            disabled={live === "run" || !canTest}
            onClick={onTest}
          >
            {live === "run" ? <i className="spn" /> : <Ic n="plug" s={13} />}
          </button>
        )}
        <Switch
          on={provider.enabled}
          disabled={needsKey || !canToggle}
          label={provider.enabled ? t("Turn off") : t("Turn on")}
          onChange={onToggle}
        />
      </span>
    </li>
  );
}
