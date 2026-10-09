import { ToggleChipsField } from "@/components/fields/toggle-chips-field";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import { MAX_DELEGATES, MEMORY_TOKEN_BUDGET, type ContextProvider } from "@/types/assistant";
import { Input } from "@trenova/shared/components/ui/input";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useMemo, type ReactNode } from "react";
import { useController, useFormContext } from "react-hook-form";
import { aicFieldTrigger } from "../../edit/field-trigger";
import { SettingRow, SwitchRow } from "../../edit/setting-row";
import { Ic } from "../../kit/ic";
import { Tile } from "../../kit/marks";
import type { AgentFormValues } from "../agent-form-schema";
import { delegateCandidates, type DelegateUnavailable } from "../delegates";
import { Blk } from "./block";

type TeamBlockProps = {
  /** Every agent in the organization; those it could hand work to are picked from them. */
  agents: readonly AgentDefinitionRow[];
  /** The agent being edited, or null for a new one. */
  selfId: string | null;
  /** The organization has turned learning off for every agent. */
  learningOffForAll: boolean;
};

/** What it keeps, what it's told up front, and who it can ask. */
export function TeamBlock({ agents, selfId, learningOffForAll }: TeamBlockProps) {
  const t = useT();
  const { control } = useFormContext<AgentFormValues>();
  const {
    field: { value: trigger },
  } = useController({ control, name: "triggerMode" });

  return (
    <Blk
      id="team"
      title={t("Memory and handoffs")}
      note={t("What it keeps, what it's told up front, and who it can ask.")}
    >
      <LearnsRow learningOffForAll={learningOffForAll} />
      <MemoryRow />
      <ContextChips />
      <Delegates agents={agents} selfId={selfId} chat={trigger === "Chat"} />
    </Blk>
  );
}

function LearnsRow({ learningOffForAll }: { learningOffForAll: boolean }) {
  const t = useT();
  const { control } = useFormContext<AgentFormValues>();

  return (
    <SwitchRow<AgentFormValues>
      control={control}
      name="learnsFromWork"
      label={t("Learns from its work")}
      note={t(
        "Keeps a lesson when a conversation settles. Lessons shared beyond one person wait for approval.",
      )}
      disabled={learningOffForAll}
      why={t("Off for the whole organization")}
    />
  );
}

function MemoryRow() {
  const t = useT();
  const { control } = useFormContext<AgentFormValues>();
  const {
    field: { value: memory, onChange, onBlur, ref, name },
    fieldState,
  } = useController({ control, name: "memoryTokenBudget" });
  const outOfRange =
    memory !== null && (memory < MEMORY_TOKEN_BUDGET.min || memory > MEMORY_TOKEN_BUDGET.max);

  return (
    <SettingRow
      label={t("Memory in the prompt")}
      labelFor="team-memory"
      note={t("How much remembered context goes in with each question. Empty uses the default.")}
      why={
        fieldState.error?.message ??
        (outOfRange
          ? t(
              "Between {0} and {1} tokens.",
              MEMORY_TOKEN_BUDGET.min.toLocaleString(),
              MEMORY_TOKEN_BUDGET.max.toLocaleString(),
            )
          : undefined)
      }
    >
      <Input
        id="team-memory"
        ref={ref}
        name={name}
        inputMode="numeric"
        value={memory === null ? "" : String(memory)}
        placeholder={t("Default")}
        sideText={t("tokens")}
        aria-invalid={fieldState.invalid || outOfRange || undefined}
        inputContainerClassName="w-42.5 shrink-0"
        className={aicFieldTrigger}
        onBlur={onBlur}
        onChange={(event) => {
          const digits = event.target.value.replace(/\D/g, "");
          onChange(digits === "" ? null : Number(digits));
        }}
      />
    </SettingRow>
  );
}

type SectionHeadProps = { title: string; note: ReactNode };

function SectionHead({ title, note }: SectionHeadProps) {
  return (
    <div className="mb-2.5 flex items-baseline gap-2.5">
      <b className="text-sm font-medium">{title}</b>
      <span className="text-muted-foreground text-sm">{note}</span>
    </div>
  );
}

/** What it is told up front, each a chip that toggles; none chosen tells it everything. */
function ContextChips() {
  const t = useT();
  const { control } = useFormContext<AgentFormValues>();
  const {
    field: { value: context },
  } = useController({ control, name: "contextProviders" });
  const options: { value: ContextProvider; label: string }[] = [
    { value: "Organization", label: t("The organization") },
    { value: "Clock", label: t("The current time") },
    { value: "User", label: t("The person asking") },
    { value: "Page", label: t("The page they are on") },
    { value: "Tools", label: t("Its own tools") },
    { value: "Memory", label: t("What the organization recorded") },
  ];

  return (
    <div className="mt-4.5">
      <SectionHead
        title={t("Tell it about")}
        note={
          context.length
            ? t("{0} of {1}", context.length, options.length)
            : t("Everything — leave all unchecked to include all of it")
        }
      />
      <ToggleChipsField<AgentFormValues, ContextProvider>
        control={control}
        name="contextProviders"
        label={t("Tell it about")}
        options={options}
        checkedTone="success"
      />
    </div>
  );
}

type DelegatesProps = {
  agents: readonly AgentDefinitionRow[];
  selfId: string | null;
  chat: boolean;
};

/** The agents it may hand a conversation to, up to the limit, each a tile that toggles. */
function Delegates({ agents, selfId, chat }: DelegatesProps) {
  const t = useT();
  const { control } = useFormContext<AgentFormValues>();
  const {
    field: { value: delegates, onChange },
  } = useController({ control, name: "delegateIds" });
  const candidates = useMemo(
    () => delegateCandidates(agents, { selfId, chosen: delegates }),
    [agents, delegates, selfId],
  );
  const unavailableNote: Record<DelegateUnavailable, string> = {
    disabled: t("Disabled · take it off or enable it"),
    background: t("Runs on its own · it can't be handed work"),
    page: t("Works on its own page · it can't be handed work"),
  };

  return (
    <div className="mt-4.5">
      <SectionHead
        title={t("Can ask")}
        note={
          chat
            ? t(
                "{0} of {1} · it passes the conversation along with what it found",
                delegates.length,
                MAX_DELEGATES,
              )
            : t("Only agents people talk to can ask others")
        }
      />
      {chat && candidates.length > 0 ? (
        <div className="grid grid-cols-[repeat(auto-fill,minmax(190px,1fr))] gap-1.5">
          {candidates.map(({ agent: other, unavailable }) => {
            const on = delegates.includes(other.id);
            const full = !on && delegates.length >= MAX_DELEGATES;
            const warn = on && unavailable !== null;
            return (
              <button
                key={other.id}
                type="button"
                aria-pressed={on}
                disabled={full}
                title={unavailable ? unavailableNote[unavailable] : undefined}
                className={cn(
                  "group ui-focus-ring border-border-subtle hover:border-border-strong relative flex items-center gap-2.5 rounded-lg border py-1.5 pr-7.5 pl-1.5 text-left text-base transition-colors disabled:cursor-not-allowed disabled:opacity-40",
                  on && !warn && "border-brand/45 bg-brand/6",
                  warn && "border-warning/45 bg-warning/6",
                  !on && "[&_.tile]:opacity-75 [&_.tile]:grayscale",
                )}
                onClick={() =>
                  onChange(
                    on ? delegates.filter((id) => id !== other.id) : [...delegates, other.id],
                  )
                }
              >
                <Tile agent={other} s={24} />
                <span className="flex min-w-0 flex-col">
                  <span className="truncate">{other.name}</span>
                  {unavailable && (
                    <em className="text-warning-foreground truncate text-xs not-italic">
                      {unavailableNote[unavailable]}
                    </em>
                  )}
                </span>
                <i
                  className={cn(
                    "absolute right-2.5 grid size-4 place-items-center rounded-full border-[1.5px] transition-colors",
                    on
                      ? warn
                        ? "border-warning bg-warning text-foreground-on-solid"
                        : "border-brand bg-brand text-foreground-on-solid"
                      : "border-border-strong text-transparent",
                  )}
                >
                  <Ic n="check" s={9} w={3} />
                </i>
              </button>
            );
          })}
        </div>
      ) : (
        <p className="text-muted-foreground mt-1.5 text-base text-pretty">
          {chat
            ? t("No other agent can be asked yet. Agents people talk to that are on show up here.")
            : t('Switch it to "Someone asks" to let it hand work to other agents.')}
        </p>
      )}
    </div>
  );
}
