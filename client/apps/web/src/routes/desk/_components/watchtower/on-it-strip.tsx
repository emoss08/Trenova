import type { WatchtowerItem } from "@/lib/graphql/watchtower";
import { Button } from "@trenova/shared/components/ui/button";
import { useElementSize } from "@trenova/shared/hooks/use-element-size";
import { useT } from "@trenova/shared/i18n/use-t";
import { AgentRing } from "./agent-ring";

/** A ring's button and the gap after it, as `.dk-w3-on` draws them. */
const RING_PX = 26;
const GAP_PX = 8;

/**
 * How many rings fit in `width`, and how many agents are left over for the
 * "+N" chip. The chip takes a ring's slot, so when they do not all fit one
 * ring gives way to it.
 */
export function ringSlots(width: number, working: number): { shown: number; more: number } {
  const fit = Math.max(0, Math.floor((width + GAP_PX) / (RING_PX + GAP_PX)));
  if (working <= fit) {
    return { shown: working, more: 0 };
  }
  const shown = Math.max(0, fit - 1);
  return { shown, more: working - shown };
}

/**
 * The agents at work, above the queue: a ring for each that fits, filling as
 * its run goes on, then "+N" for the rest, and how many are working. A ring
 * opens the item its agent is on; "+N" opens the next of the ones not drawn.
 */
export function OnItStrip({
  working,
  selectedId,
  onSelect,
}: {
  working: readonly WatchtowerItem[];
  selectedId: string | null;
  onSelect: (id: string) => void;
}) {
  const t = useT();
  const [ringsRef, { width }] = useElementSize<HTMLDivElement>();
  const { shown, more } = ringSlots(width, working.length);
  const hidden = working.slice(shown);
  const nextHidden = hidden.find((item) => item.id !== selectedId) ?? hidden[0] ?? null;

  return (
    <div className="dk-w3-on">
      <span className="dk-w3-onl">{t("On it")}</span>
      <div ref={ringsRef} className="dk-w3-onr">
        {working.slice(0, shown).map((item) => {
          const name = item.agentWork?.agent?.name ?? t("An agent");
          return (
            <Button
              key={item.id}
              variant="bare"
              size="bare"
              title={[name, item.title].join(" · ")}
              aria-label={t("{0} is on {1}", name, item.title)}
              onClick={() => onSelect(item.id)}
            >
              {item.agentWork && (
                <AgentRing work={item.agentWork} agent={item.agentWork.agent} size={18} />
              )}
            </Button>
          );
        })}
        {more > 0 && nextHidden !== null && (
          <Button
            variant="bare"
            size="bare"
            className="dk-w3-onm"
            title={hidden.map((item) => item.agentWork?.agent?.name ?? t("An agent")).join(", ")}
            aria-label={t(
              "{0, plural, one {# more agent working; open its item} other {# more agents working; open the next item}}",
              more,
            )}
            onClick={() => onSelect(nextHidden.id)}
          >
            {t("+{0}", more)}
          </Button>
        )}
      </div>
      <span className="dk-w3-onn">
        {t("{0, plural, one {# working} other {# working}}", working.length)}
      </span>
    </div>
  );
}
