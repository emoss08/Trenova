import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import { MAX_DELEGATES, MEMORY_TOKEN_BUDGET, type ContextProvider } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Chips, SwRow } from "../../edit/fields";
import { Ic } from "../../kit/ic";
import { Tile } from "../../kit/marks";
import { Blk, useDraftField } from "./block";

type TeamBlockProps = {
  /** Every other agent, the ones it could hand work to. */
  agents: readonly AgentDefinitionRow[];
  /** The organization has turned learning off for every agent. */
  learningOffForAll: boolean;
};

/** What it keeps, what it's told up front, and who it can ask. */
export function TeamBlock({ agents, learningOffForAll }: TeamBlockProps) {
  const t = useT();
  const [learns, setLearns] = useDraftField("learnsFromWork");
  const [memory, setMemory] = useDraftField("memoryTokenBudget");
  const [context, setContext] = useDraftField("contextProviders");
  const [delegates, setDelegates] = useDraftField("delegateIds");
  const [trigger] = useDraftField("triggerMode");
  const chat = trigger === "Chat";
  const memoryOff =
    memory !== null && (memory < MEMORY_TOKEN_BUDGET.min || memory > MEMORY_TOKEN_BUDGET.max);

  const contextOptions: [ContextProvider, string][] = [
    ["Organization", t("The organization")],
    ["Clock", t("The current time")],
    ["User", t("The person asking")],
    ["Page", t("The page they are on")],
    ["Tools", t("Its own tools")],
    ["Memory", t("What the organization recorded")],
  ];

  return (
    <Blk
      id="team"
      title={t("Memory and handoffs")}
      note={t("What it keeps, what it's told up front, and who it can ask.")}
    >
      <SwRow
        label={t("Learns from its work")}
        note={t(
          "Keeps a lesson when a conversation settles. Lessons shared beyond one person wait for approval.",
        )}
        on={learns && !learningOffForAll}
        disabled={learningOffForAll}
        why={t("Off for the whole organization")}
        onChange={setLearns}
      />
      <div className="swr">
        <span>
          <b>{t("Memory in the prompt")}</b>
          <em>
            {t(
              "How much remembered context goes in with each question. Empty uses the default.",
            )}
          </em>
        </span>
        <label className="inx mono" style={{ width: 170 }}>
          <input
            inputMode="numeric"
            aria-label={t("Memory in the prompt")}
            value={memory === null ? "" : String(memory)}
            placeholder={t("Default")}
            onChange={(event) => {
              const digits = event.target.value.replace(/\D/g, "");
              setMemory(digits === "" ? null : Number(digits));
            }}
          />
          <span className="inx-a">{t("tokens")}</span>
        </label>
      </div>
      {memoryOff && (
        <p className="f-h t-w">
          {t(
            "Between {0} and {1} tokens.",
            MEMORY_TOKEN_BUDGET.min.toLocaleString(),
            MEMORY_TOKEN_BUDGET.max.toLocaleString(),
          )}
        </p>
      )}
      <div className="tm-s">
        <div className="tm-h">
          <b>{t("Tell it about")}</b>
          <span>
            {context.length
              ? t("{0} of {1}", context.length, contextOptions.length)
              : t("Everything — leave all unchecked to include all of it")}
          </span>
        </div>
        <Chips
          value={context}
          onChange={setContext}
          options={contextOptions}
          label={t("Tell it about")}
        />
      </div>
      <div className="tm-s">
        <div className="tm-h">
          <b>{t("Can ask")}</b>
          <span>
            {chat
              ? t(
                  "{0} of {1} · it passes the conversation along with what it found",
                  delegates.length,
                  MAX_DELEGATES,
                )
              : t("Only agents people talk to can ask others")}
          </span>
        </div>
        {chat ? (
          <div className="hof">
            {agents.map((other) => {
              const on = delegates.includes(other.id);
              const full = !on && delegates.length >= MAX_DELEGATES;
              return (
                <button
                  key={other.id}
                  type="button"
                  aria-pressed={on}
                  disabled={full}
                  className={cn("hof-i", on && "on")}
                  onClick={() =>
                    setDelegates(
                      on ? delegates.filter((id) => id !== other.id) : [...delegates, other.id],
                    )
                  }
                >
                  <Tile agent={other} s={24} />
                  <span>{other.name}</span>
                  <i>
                    <Ic n="check" s={9} w={3} />
                  </i>
                </button>
              );
            })}
          </div>
        ) : (
          <p className="es-note">
            {t('Switch it to "Someone asks" to let it hand work to other agents.')}
          </p>
        )}
      </div>
    </Blk>
  );
}
