import {
  AUTO_COMPACT_SHARE,
  DRAIN_MS,
  kfmt,
  meterView,
  type MeterPart,
} from "@/components/assistant/compaction";
import type { ContextUsage, ThreadBudget } from "@/types/assistant";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixMonthDay, formatUnixTime } from "@trenova/shared/lib/date";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useRef, useState } from "react";
import { DeskIcon } from "../desk-icons";
import { useOutsideDismiss } from "../use-outside-dismiss";

/** The meter's ring: how full the context is, drawn clockwise from the top. */
export function DeskContextRing({ share, size = 16 }: { share: number; size?: number }) {
  const radius = (size - 3) / 2;
  const circumference = 2 * Math.PI * radius;
  const center = size / 2;
  return (
    <svg
      width={size}
      height={size}
      viewBox={`0 0 ${size} ${size}`}
      className="dk-cx-ring"
      aria-hidden
    >
      <circle cx={center} cy={center} r={radius} fill="none" strokeWidth="2" className="dk-cx-rt" />
      <circle
        cx={center}
        cy={center}
        r={radius}
        fill="none"
        strokeWidth="2"
        strokeLinecap="round"
        strokeDasharray={circumference}
        strokeDashoffset={circumference * (1 - Math.min(1, Math.max(0, share)))}
        transform={`rotate(-90 ${center} ${center})`}
        className="dk-cx-rf"
      />
    </svg>
  );
}

/**
 * The figure the context drains to while a conversation compacts: a small
 * ring and the token count, easing from before to after.
 */
export function DeskContextDrain({
  from,
  to,
  window,
  ms = DRAIN_MS,
}: {
  from: number;
  to: number;
  window: number;
  ms?: number;
}) {
  const [value, setValue] = useState(from);
  useEffect(() => {
    let frame = 0;
    const started = performance.now();
    const ease = (x: number) => 1 - Math.pow(1 - x, 3);
    const step = (now: number) => {
      const k = Math.min(1, (now - started) / ms);
      setValue(from + (to - from) * ease(k));
      if (k < 1) {
        frame = requestAnimationFrame(step);
      }
    };
    frame = requestAnimationFrame(step);
    return () => cancelAnimationFrame(frame);
  }, [from, to, ms]);

  return (
    <span className="dk-cx-drain">
      <DeskContextRing share={value / window} size={14} />
      <span className="dk-cx-dn">{kfmt(Math.round(value / 100) * 100)}</span>
    </span>
  );
}

function partLabel(key: MeterPart["key"], t: TranslateFn): string {
  switch (key) {
    case "conv":
      return t("Messages");
    case "tools":
      return t("Tool results & artifacts");
    case "files":
      return t("Files");
    case "sys":
      return t("Agent instructions");
  }
}

/** One usage cap the person is spending against, as the meter lists it. */
type UsageRow = { key: string; title: string; resets: string; share: number };

/** The caps this conversation spends from, those that have a limit set. */
function usageRows(budget: ThreadBudget | undefined, timezone: string, t: TranslateFn): UsageRow[] {
  if (!budget) {
    return [];
  }
  const rows: UsageRow[] = [];
  const month = (at: number) =>
    new Date(at * 1000).toLocaleDateString(undefined, { month: "long", timeZone: "UTC" });
  if (budget.dailyRunLimit > 0) {
    rows.push({
      key: "daily",
      title: t("Today · {0}", budget.agentName),
      resets: t("Resets at {0}", formatUnixTime(budget.dayResetsAt, { timezone })),
      share: budget.runsToday / budget.dailyRunLimit,
    });
  }
  if (budget.limitUsd !== "" && Number(budget.limitUsd) > 0) {
    rows.push({
      key: "budget",
      title: t("{0} · {1}", month(budget.monthStart), budget.agentName),
      resets: t("Resets {0}", formatUnixMonthDay(budget.resetsAt, { timezone: "UTC" })),
      share: budget.share,
    });
  }
  const person = budget.person;
  if (person && person.limit > 0) {
    rows.push({
      key: "person",
      title: t("{0} · you", month(budget.monthStart || person.resetsAt)),
      resets: t("Resets {0}", formatUnixMonthDay(person.resetsAt, { timezone: "UTC" })),
      share: person.used / person.limit,
    });
  }

  return rows;
}

/**
 * The context meter beside the model picker: a ring that fills as the
 * conversation does, its share in figures from three quarters full, turning
 * amber then red. Opening it shows the window by part, compacts the
 * conversation on request, says whether it compacts itself, and lists the
 * usage caps the conversation spends from.
 */
export function DeskContextMeter({
  usage,
  auto,
  onAutoChange,
  onCompact,
  compacting,
  budget,
}: {
  usage: ContextUsage | null;
  auto: boolean;
  onAutoChange: (on: boolean) => void;
  onCompact: () => void;
  compacting: boolean;
  budget?: ThreadBudget;
}) {
  const t = useT();
  const timezone = useAuthStore((state) => state.user?.timezone) || "UTC";
  const [open, setOpen] = useState(false);
  const [more, setMore] = useState(false);
  const ref = useRef<HTMLSpanElement>(null);
  useOutsideDismiss(ref, open, () => setOpen(false));

  const view = meterView(usage);
  const rows = usageRows(budget, timezone, t);

  return (
    <span className={cn("dk-cx", view.tone && `dk-${view.tone}`)} ref={ref}>
      <button
        type="button"
        className={cn("dk-cx-b", open && "dk-on", compacting && "dk-busy")}
        title={compacting ? t("Compacting…") : t("Context {0}% used", view.pct)}
        aria-label={compacting ? t("Compacting…") : t("Context {0}% used", view.pct)}
        aria-expanded={open}
        onClick={() => setOpen((value) => !value)}
      >
        <DeskContextRing share={view.share} />
        {view.showPct && <span className="dk-cx-pc">{view.pct}%</span>}
      </button>
      {open && (
        <div className="dk-cx-pop" role="dialog" aria-label={t("Context window")}>
          <button type="button" className="dk-cx-row dk-cx-h" onClick={() => setMore((m) => !m)}>
            <span>{t("Context window")}</span>
            <span className="dk-cx-v">
              {kfmt(view.used)} / {kfmt(view.window)} ({view.pct}%)
            </span>
            <span className={cn("dk-cx-cv", more && "dk-on")}>
              <DeskIcon name="chevR" size={12} stroke={2} />
            </span>
          </button>
          <div className="dk-cx-bar">
            {view.parts.map((part) =>
              part.tokens > 0 ? (
                <i
                  key={part.key}
                  className={`dk-cx-${part.tone}`}
                  style={{ width: `${(part.tokens / view.window) * 100}%` }}
                />
              ) : null,
            )}
            <b style={{ left: `${AUTO_COMPACT_SHARE * 100}%` }} title={t("Auto-compacts here")} />
          </div>
          {more && (
            <div className="dk-cx-leg">
              {view.parts.map((part) => (
                <div key={part.key}>
                  <i className={`dk-cx-${part.tone}`} />
                  <span>{partLabel(part.key, t)}</span>
                  <span className="dk-cx-v">{kfmt(part.tokens)}</span>
                </div>
              ))}
            </div>
          )}
          <div className="dk-cx-cmp">
            <div className="dk-cx-ct">
              <b>{t("Compact conversation")}</b>
              <span>
                {view.canCompact ? (
                  <>
                    {t("Summarizes older turns and frees about")} <em>{kfmt(view.frees)}</em>.{" "}
                    {t("Pinned artifacts and pending approvals stay in full.")}
                  </>
                ) : (
                  t("Nothing to compact yet.")
                )}
              </span>
            </div>
            <button
              type="button"
              className="dk-cx-go"
              disabled={compacting || !view.canCompact}
              onClick={() => {
                setOpen(false);
                onCompact();
              }}
            >
              {compacting ? t("Compacting…") : t("Compact")}
            </button>
          </div>
          <label className="dk-cx-auto">
            <span>{t("Compact automatically at {0}%", Math.round(AUTO_COMPACT_SHARE * 100))}</span>
            <button
              type="button"
              role="switch"
              aria-checked={auto}
              className={cn("dk-cx-sw", auto && "dk-on")}
              onClick={() => onAutoChange(!auto)}
            >
              <i />
            </button>
          </label>
          {rows.length > 0 && (
            <>
              <div className="dk-cx-sep" />
              <div className="dk-cx-row dk-cx-sub">
                <span>{t("Usage limits")}</span>
                <DeskIcon name="ext" size={12} />
              </div>
              {rows.map((row) => {
                const pct = Math.round(row.share * 100);
                return (
                  <div className="dk-cx-u" key={row.key}>
                    <div className="dk-cx-row">
                      <b>{row.title}</b>
                      <span className="dk-cx-r">{row.resets}</span>
                      <span className="dk-cx-v">{pct}%</span>
                    </div>
                    <div className="dk-cx-bar dk-thin">
                      <i
                        className={
                          pct >= 90 ? "dk-cx-c-hot" : pct >= 75 ? "dk-cx-c-warm" : "dk-cx-c1"
                        }
                        style={{ width: `${Math.min(100, pct)}%` }}
                      />
                    </div>
                  </div>
                );
              })}
            </>
          )}
          <div className="dk-cx-f">
            <button type="button" className="dk-cx-fb" onClick={() => setMore((m) => !m)}>
              {t("See detailed breakdown")}
            </button>
          </div>
        </div>
      )}
    </span>
  );
}
