import { AIProviderAutocomplete } from "@/components/autocomplete-fields";
import { ErrorMessage } from "@/components/fields/field-components";
import type { AgentBudgetStatus } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useState, type ReactNode } from "react";
import { useController, useFormContext } from "react-hook-form";
import { aicFieldTrigger } from "../../edit/field-trigger";
import { Ic, type IcName } from "../../kit/ic";
import type { AgentFormValues } from "../agent-form-schema";
import { Blk } from "./block";

type LimitsBlockProps = {
  fresh: boolean;
  /** Where it stands against its caps this month, for an agent already saved. */
  budget: AgentBudgetStatus | undefined;
};

type LimitName = "monthlyBudgetUsd" | "dailyRunLimit" | "runTimeoutSeconds" | "maxToolCalls";

const SECONDS_PER_MINUTE = 60;

/** Where it stops, however well it's going, and how it answers. */
export function LimitsBlock({ fresh, budget }: LimitsBlockProps) {
  const t = useT();
  const { control } = useFormContext<AgentFormValues>();
  const {
    field: { value: monthly },
  } = useController({ control, name: "monthlyBudgetUsd" });
  const {
    field: { value: runsPerDay },
  } = useController({ control, name: "dailyRunLimit" });
  const {
    field: { value: preferred, onChange: setPreferred },
  } = useController({ control, name: "preferredProviderId" });

  const spent = budget ? Number(budget.spentUsd) : null;
  const share = monthly && spent !== null && monthly > 0 ? Math.min(1, spent / monthly) : null;

  return (
    <Blk
      id="limits"
      fresh={fresh}
      title={t("Budget and model")}
      note={t("Where it stops, however well it's going, and how it answers.")}
    >
      <div className="border-border-subtle grid grid-cols-1 border-t md:grid-cols-2">
        <LimitCell
          name="monthlyBudgetUsd"
          label={t("Monthly budget")}
          prefix="$"
          placeholder={t("No cap")}
          step={5}
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
        <LimitCell
          name="dailyRunLimit"
          label={t("Runs per day")}
          suffix={t("runs")}
          step={10}
          hint={runsPerDay ? t("Then it stops for the day and says so") : t("Zero means no cap")}
        />
        <LimitCell
          name="runTimeoutSeconds"
          label={t("Run timeout")}
          suffix={t("min")}
          step={1}
          min={1}
          scale={SECONDS_PER_MINUTE}
          hint={t("A run that takes longer is stopped")}
        />
        <LimitCell
          name="maxToolCalls"
          label={t("Tool calls per run")}
          suffix={t("calls")}
          step={1}
          min={1}
          hint={t("What one run may spend looking things up and acting")}
        />
      </div>
      <div className="mt-5 grid grid-cols-1 gap-5 md:grid-cols-2">
        <div className="flex flex-col gap-2">
          <span className="text-muted-foreground text-sm">{t("Preferred provider")}</span>
          <AIProviderAutocomplete
            label={t("Preferred provider")}
            value={preferred}
            onValueChange={setPreferred}
            placeholder={t("Automatic")}
            triggerClassName={aicFieldTrigger}
          />
          <span className="text-muted-foreground text-xs">
            {t("Tried first. Automatic follows the routing on the Providers tab.")}
          </span>
        </div>
        <RepliesAs />
      </div>
    </Blk>
  );
}

type LimitCellProps = {
  name: LimitName;
  label: string;
  prefix?: string;
  suffix?: string;
  placeholder?: string;
  step: number;
  min?: number;
  /** How many stored units one shown unit is, such as seconds in a minute. */
  scale?: number;
  hint: ReactNode;
  /** How much of the cap is spent, 0 to 1, or null for no bar. */
  bar?: number | null;
  /** An empty box means no cap, rather than zero. */
  allowEmpty?: boolean;
  /** Takes cents as well as whole numbers. */
  decimal?: boolean;
};

/** One limit as a big figure, typed in place or stepped. */
function LimitCell({
  name,
  label,
  prefix,
  suffix,
  placeholder = "0",
  step,
  min = 0,
  scale = 1,
  hint,
  bar = null,
  allowEmpty = false,
  decimal = false,
}: LimitCellProps) {
  const t = useT();
  const { control } = useFormContext<AgentFormValues>();
  const {
    field: { value: stored, onChange, onBlur, ref, name: fieldName },
    fieldState,
  } = useController({ control, name });
  const [typing, setTyping] = useState<string | null>(null);

  const shown = stored === null ? "" : String(Math.round(stored / scale));
  const current = Number(shown) || 0;
  const write = (next: number | null) => {
    if (next === null) {
      onChange(allowEmpty ? null : min * scale);
      return;
    }
    onChange(Math.max(min, next) * scale);
  };
  const stepTo = (next: number) => {
    setTyping(null);
    write(next);
  };
  const inputId = `limit-${name}`;

  return (
    <div className="border-border-subtle flex flex-col gap-1.5 border-b py-4.5 md:odd:pr-5 md:even:border-l md:even:pl-5">
      <label htmlFor={inputId} className="text-muted-foreground text-sm">
        {label}
      </label>
      <div className="flex items-baseline gap-1">
        {prefix && <span className="text-muted-foreground text-lg font-medium">{prefix}</span>}
        <input
          id={inputId}
          name={fieldName}
          ref={ref}
          inputMode={decimal ? "decimal" : "numeric"}
          value={typing ?? shown}
          placeholder={placeholder}
          aria-invalid={fieldState.invalid || undefined}
          className={cn(
            "text-foreground focus:text-brand placeholder:text-muted-foreground w-24 border-0 bg-transparent p-0 font-mono text-3xl font-semibold tracking-tight outline-none placeholder:text-2xl",
            fieldState.invalid && "text-danger-foreground",
          )}
          onFocus={() => setTyping(shown)}
          onBlur={() => {
            setTyping(null);
            onBlur();
          }}
          onChange={(event) => {
            const raw = event.target.value.replace(decimal ? /[^\d.]/g : /\D/g, "");
            setTyping(raw);
            const parsed = Number(raw);
            if (raw === "" || !Number.isFinite(parsed)) {
              write(null);
              return;
            }
            write(decimal ? Math.round(parsed * 100) / 100 : Math.trunc(parsed));
          }}
        />
        {suffix && <span className="text-muted-foreground text-lg font-medium">{suffix}</span>}
        <span className="border-border ml-auto inline-flex self-center overflow-hidden rounded-md border">
          <button
            type="button"
            className="text-muted-foreground hover:bg-field hover:text-foreground ui-focus-ring h-6.5 w-7 text-base transition-colors"
            aria-label={t("Less {0}", label.toLowerCase())}
            onClick={() => stepTo(Math.max(min, current - step))}
          >
            −
          </button>
          <button
            type="button"
            className="border-border text-muted-foreground hover:bg-field hover:text-foreground ui-focus-ring h-6.5 w-7 border-l text-base transition-colors"
            aria-label={t("More {0}", label.toLowerCase())}
            onClick={() => stepTo(current + step)}
          >
            +
          </button>
        </span>
      </div>
      {bar !== null && (
        <span className="bg-border block h-1 rounded-full">
          <i className="bg-brand block h-full rounded-full" style={{ width: `${bar * 100}%` }} />
        </span>
      )}
      {fieldState.error?.message ? (
        <ErrorMessage formError={fieldState.error.message} />
      ) : (
        <span className="text-muted-foreground text-xs">{hint}</span>
      )}
    </div>
  );
}

/** How its answers read: a conversation, or a report in sections. */
function RepliesAs() {
  const t = useT();
  const { control } = useFormContext<AgentFormValues>();
  const {
    field: { value: mode, onChange },
  } = useController({ control, name: "outputMode" });
  const options: readonly [AgentFormValues["outputMode"], IcName, string, string][] = [
    ["Conversational", "chat", t("A conversation"), t("Answers in prose")],
    ["Report", "receipt", t("A report"), t("Answers in sections")],
  ];

  return (
    <div className="flex flex-col gap-2">
      <span className="text-muted-foreground text-sm">{t("Replies as")}</span>
      <div className="grid grid-cols-2 gap-1.5" role="radiogroup" aria-label={t("Replies as")}>
        {options.map(([option, icon, label, note]) => {
          const on = mode === option;
          return (
            <button
              key={option}
              type="button"
              role="radio"
              aria-checked={on}
              className={cn(
                "border-border hover:border-border-strong ui-focus-ring flex items-center gap-2.5 rounded-lg border px-2.5 py-2 text-left transition-colors",
                on && "border-brand/55 bg-brand/6 hover:border-brand/55",
              )}
              onClick={() => onChange(option)}
            >
              <span className={on ? "text-brand" : "text-muted-foreground"}>
                <Ic n={icon} s={14} />
              </span>
              <span className="flex flex-col">
                <b className="text-sm font-medium">{label}</b>
                <span className="text-muted-foreground text-xs">{note}</span>
              </span>
            </button>
          );
        })}
      </div>
    </div>
  );
}
