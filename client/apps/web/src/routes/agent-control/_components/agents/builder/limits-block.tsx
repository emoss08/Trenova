import type { AgentBudgetStatus } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useState, type ReactNode } from "react";
import { Sel } from "../../edit/fields";
import { Ic } from "../../kit/ic";
import { Blk, useDraftField } from "./block";

type Provider = { id: string; name: string; model: string };

type LimitsBlockProps = {
  fresh: boolean;
  /** Providers that can answer the agent, enabled, in routing order. */
  providers: readonly Provider[];
  /** Where it stands against its caps this month, for an agent already saved. */
  budget: AgentBudgetStatus | undefined;
};

/** Where it stops, however well it's going, and how it answers. */
export function LimitsBlock({ fresh, providers, budget }: LimitsBlockProps) {
  const t = useT();
  const [monthly, setMonthly] = useDraftField("monthlyBudgetUsd");
  const [runsPerDay, setRunsPerDay] = useDraftField("dailyRunLimit");
  const [timeout, setTimeoutSeconds] = useDraftField("runTimeoutSeconds");
  const [toolCalls, setToolCalls] = useDraftField("maxToolCalls");
  const [preferred, setPreferred] = useDraftField("preferredProviderId");
  const [output, setOutput] = useDraftField("outputMode");

  const spent = budget ? Number(budget.spentUsd) : null;
  const share = monthly && spent !== null && monthly > 0 ? Math.min(1, spent / monthly) : null;

  return (
    <Blk
      id="limits"
      fresh={fresh}
      title={t("Budget and model")}
      note={t("Where it stops, however well it's going, and how it answers.")}
    >
      <div className="gr">
        <Cell
          label={t("Monthly budget")}
          prefix="$"
          value={monthly === null ? "" : String(monthly)}
          placeholder={t("No cap")}
          step={5}
          onChange={(next) => setMonthly(next)}
          hint={
            monthly === null
              ? t("Empty means no cap")
              : spent !== null
                ? t("${0} spent this month · across every run", spent.toFixed(2))
                : t("Counted across every run this month")
          }
          bar={share}
          allowEmpty
          decimal
        />
        <Cell
          label={t("Runs per day")}
          suffix={t("runs")}
          value={String(runsPerDay)}
          step={10}
          onChange={(next) => setRunsPerDay(next ?? 0)}
          hint={runsPerDay ? t("Then it stops for the day and says so") : t("Zero means no cap")}
        />
        <Cell
          label={t("Run timeout")}
          suffix={t("min")}
          value={String(Math.round(timeout / 60))}
          step={1}
          min={1}
          onChange={(next) => setTimeoutSeconds(Math.max(1, next ?? 1) * 60)}
          hint={t("A run that takes longer is stopped")}
        />
        <Cell
          label={t("Tool calls per run")}
          suffix={t("calls")}
          value={String(toolCalls)}
          step={1}
          min={1}
          onChange={(next) => setToolCalls(Math.max(1, next ?? 1))}
          hint={t("What one run may spend looking things up and acting")}
        />
      </div>
      <div className="mdr">
        <div className="mdr-c">
          <span className="gr-l">{t("Preferred provider")}</span>
          <Sel
            value={preferred}
            onChange={setPreferred}
            label={t("Preferred provider")}
            options={[
              ["", t("Automatic")] as const,
              ...providers.map(
                (provider) => [provider.id, `${provider.name} · ${provider.model}`] as const,
              ),
            ]}
          />
          <span className="gr-h">
            {t("Tried first. Automatic follows the routing on the Providers tab.")}
          </span>
        </div>
        <div className="mdr-c">
          <span className="gr-l">{t("Replies as")}</span>
          <div className="ro" role="radiogroup" aria-label={t("Replies as")}>
            {(
              [
                ["Conversational", "chat", t("A conversation"), t("Answers in prose")],
                ["Report", "receipt", t("A report"), t("Answers in sections")],
              ] as const
            ).map(([mode, icon, label, note]) => (
              <button
                key={mode}
                type="button"
                role="radio"
                aria-checked={output === mode}
                className={cn("ro-o", output === mode && "on")}
                onClick={() => setOutput(mode)}
              >
                <Ic n={icon} s={14} />
                <span>
                  <b>{label}</b>
                  <em>{note}</em>
                </span>
              </button>
            ))}
          </div>
        </div>
      </div>
    </Blk>
  );
}

type CellProps = {
  label: string;
  prefix?: string;
  suffix?: string;
  value: string;
  placeholder?: string;
  step: number;
  min?: number;
  hint: ReactNode;
  /** How much of the cap is spent, 0 to 1, or null for no bar. */
  bar?: number | null;
  /** An empty box means no cap, rather than zero. */
  allowEmpty?: boolean;
  /** Takes cents as well as whole numbers. */
  decimal?: boolean;
  onChange: (value: number | null) => void;
};

/** One big figure with a stepper. */
function Cell({
  label,
  prefix,
  suffix,
  value,
  placeholder = "0",
  step,
  min = 0,
  hint,
  bar = null,
  allowEmpty = false,
  decimal = false,
  onChange,
}: CellProps) {
  const t = useT();
  const [typing, setTyping] = useState<string | null>(null);
  const current = Number(value) || 0;

  return (
    <div className="gr-c">
      <span className="gr-l">{label}</span>
      <div className="gr-v">
        {prefix && <span className="gr-u">{prefix}</span>}
        <input
          className="mono"
          inputMode={decimal ? "decimal" : "numeric"}
          aria-label={label}
          value={typing ?? value}
          placeholder={placeholder}
          onFocus={() => setTyping(value)}
          onBlur={() => setTyping(null)}
          onChange={(event) => {
            const raw = event.target.value.replace(decimal ? /[^\d.]/g : /\D/g, "");
            setTyping(raw);
            const parsed = Number(raw);
            if (raw === "" || !Number.isFinite(parsed)) {
              onChange(allowEmpty ? null : min);
              return;
            }
            onChange(decimal ? Math.round(parsed * 100) / 100 : Math.trunc(parsed));
          }}
        />
        {suffix && <span className="gr-u">{suffix}</span>}
        <span className="gr-st">
          <button
            type="button"
            aria-label={t("Less {0}", label.toLowerCase())}
            onClick={() => onChange(Math.max(min, current - step))}
          >
            −
          </button>
          <button
            type="button"
            aria-label={t("More {0}", label.toLowerCase())}
            onClick={() => onChange(current + step)}
          >
            +
          </button>
        </span>
      </div>
      {bar !== null && (
        <span className="gr-bar">
          <i style={{ width: `${bar * 100}%` }} />
        </span>
      )}
      <span className="gr-h">{hint}</span>
    </div>
  );
}
