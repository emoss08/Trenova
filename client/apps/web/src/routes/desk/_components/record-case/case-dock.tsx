import {
  DeskHomeComposer,
  useDeskHomeAsk,
  type DeskStartExtras,
} from "@/components/desk-chat/desk-home-ask";
import { useDeskSetting } from "@/stores/desk-settings-store";
import { useT } from "@trenova/shared/i18n/use-t";
import { useDesk } from "../desk-layout";

/**
 * The box at the foot of the case: a few questions worth asking about the
 * record and the Desk's own composer under them. Whatever is asked opens in
 * Nova's margin, beside the case, with the record attached; the same box
 * takes "change the plan" in plain words.
 */
export function CaseDock({
  space,
  label,
  starters,
  agentId,
  agentName,
  onAsk,
}: {
  /** The record's space, so what is typed here and not sent is kept with the record. */
  space: string;
  label: string;
  starters: readonly string[];
  /** The agent on the case: the box is pinned to it, since it is the one that knows the case. */
  agentId: string | null;
  agentName: string | null;
  onAsk: (question: string, agentId?: string, extras?: DeskStartExtras) => void;
}) {
  const t = useT();
  const desk = useDesk();
  const preferredAgent = useDeskSetting("agent");
  const ask = useDeskHomeAsk({
    agents: desk.agents,
    threads: desk.threads,
    isLoading: desk.isLoading,
    preferId: preferredAgent,
    subjectAgentId: agentId ?? undefined,
    pageSource: "screen",
    space,
    onStart: (agentId, question, extras) => onAsk(question ?? "", agentId, extras),
  });

  return (
    <div className="dk-cz-dock">
      {!ask.noAgents && starters.length > 0 && (
        <div className="dk-cz-chips" aria-label={t("Suggested questions")}>
          {starters.map((question) => (
            <button
              key={question}
              type="button"
              className="dk-w-chip ui-focus-ring"
              onClick={() => onAsk(question)}
            >
              {question}
            </button>
          ))}
        </div>
      )}
      <div className="dk-cz-cmp" id="dk-cz-ask">
        <DeskHomeComposer
          ask={ask}
          isStarting={desk.isStarting}
          placeholder={
            agentName
              ? t("Ask about {0}, or tell {1} what to change…", label, agentName)
              : t("Ask about {0}…", label)
          }
        />
      </div>
    </div>
  );
}
