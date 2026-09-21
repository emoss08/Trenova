import { AGENT_ACCENTS, resolveAgentIdentity } from "@/components/agent-identity/agent-identity";
import { AgentTile } from "@/components/agent-identity/agent-tile";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { ArrowUpIcon } from "lucide-react";
import { useMemo, useState, type CSSProperties, type FormEvent } from "react";
import { suggestionsFor } from "./suggestions";

const SUGGESTION_COUNT = { comfortable: 3, compact: 2 } as const;

export type AskBoxProps = {
  agents: AgentDefinitionRow[];
  /** Where the picker starts: the agent this person last talked to. */
  defaultAgentId?: string | null;
  disabled?: boolean;
  /** Tighter rows and fewer openers, for the corner panel. */
  compact?: boolean;
  onAsk: (agentId: string, question: string) => void;
};

/**
 * The box you talk into, and who it goes to.
 *
 * It is the same control on the Desk's front page and in the corner panel,
 * because they are the same act. Both surfaces used to open onto a list —
 * agents here, conversations there — which put a directory between a person
 * and the only thing either surface is for. An empty text field is a better
 * first screen than a good menu.
 *
 * The openers under it are the agent's own, so the first click teaches what
 * that agent is for rather than what the product can do in general. They
 * change with the picker, which is most of what makes the picker legible.
 */
export function AskBox({
  agents,
  defaultAgentId,
  disabled = false,
  compact = false,
  onAsk,
}: AskBoxProps) {
  const t = useT();
  const preferred = agents.find((candidate) => candidate.id === defaultAgentId) ?? agents[0];
  const [agentId, setAgentId] = useState(preferred?.id ?? "");
  const [question, setQuestion] = useState("");

  const agent = agents.find((candidate) => candidate.id === agentId) ?? preferred;
  const suggestions = useMemo(
    () =>
      agent
        ? suggestionsFor(agent.template).slice(
            0,
            compact ? SUGGESTION_COUNT.compact : SUGGESTION_COUNT.comfortable,
          )
        : [],
    [agent, compact],
  );

  if (agent === undefined) {
    return null;
  }

  const accent = AGENT_ACCENTS[resolveAgentIdentity(agent).accent];

  const ask = (text: string) => {
    const trimmed = text.trim();
    if (trimmed === "" || disabled) {
      return;
    }
    onAsk(agent.id, trimmed);
    setQuestion("");
  };

  const onSubmit = (event: FormEvent) => {
    event.preventDefault();
    ask(question);
  };

  return (
    <div className="flex flex-col gap-2.5" style={{ "--agent-accent": accent } as CSSProperties}>
      <form
        onSubmit={onSubmit}
        className={cn(
          "bg-field rounded-surface ring-foreground/10 flex flex-col gap-2 p-2.5 ring-1",
          "transition-shadow focus-within:shadow-[0_0_0_1px_var(--agent-accent)]",
        )}
      >
        <textarea
          value={question}
          onChange={(event) => setQuestion(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === "Enter" && !event.shiftKey) {
              event.preventDefault();
              ask(question);
            }
          }}
          rows={compact ? 2 : 3}
          placeholder={t("Ask {0} anything about your operation", agent.name)}
          aria-label={t("Ask {0} anything about your operation", agent.name)}
          className={cn(
            "placeholder:text-muted-foreground w-full resize-none bg-transparent px-1.5 py-1",
            "text-sm outline-none",
            compact ? "max-h-32 min-h-10" : "max-h-48 min-h-14",
          )}
        />

        <div className="flex items-end gap-1.5">
          <div className="flex min-w-0 flex-1 flex-wrap items-center gap-1">
            {agents.map((candidate) => (
              <button
                key={candidate.id}
                type="button"
                aria-pressed={candidate.id === agent.id}
                onClick={() => setAgentId(candidate.id)}
                className={cn(
                  "ui-focus-ring flex items-center gap-1.5 rounded-full py-1 pr-2.5 pl-1",
                  "text-xs transition-colors",
                  candidate.id === agent.id
                    ? "bg-surface-selected text-foreground"
                    : "text-muted-foreground hover:bg-surface-hover",
                )}
              >
                <AgentTile agent={candidate} size="xs" />
                <span className="max-w-32 truncate">{candidate.name}</span>
              </button>
            ))}
          </div>

          <Button
            type="submit"
            size="icon-sm"
            disabled={disabled || question.trim() === ""}
            aria-label={t("Ask")}
            className="shrink-0"
          >
            <ArrowUpIcon className="size-4" />
          </Button>
        </div>
      </form>

      {suggestions.length > 0 && (
        <ul className="flex flex-wrap gap-1.5">
          {suggestions.map((suggestion) => (
            <li key={suggestion.prompt}>
              <button
                type="button"
                disabled={disabled}
                onClick={() => ask(suggestion.prompt)}
                className={cn(
                  "ui-focus-ring text-muted-foreground hover:text-foreground hover:bg-surface-hover",
                  "ring-foreground/10 rounded-full px-2.5 py-1 text-xs ring-1 transition-colors",
                  "disabled:opacity-50",
                )}
              >
                {t(suggestion.label)}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
