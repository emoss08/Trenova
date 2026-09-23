import { JsonViewer } from "@/components/elements/json-viewer";
import { humanizeToolName } from "@/components/assistant/proposal-state";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { AssistantArtifact } from "@/types/assistant";
import { ArrowUpRightIcon, CodeIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { Link } from "react-router";
import { entityCardFrom } from "./artifact-payloads";

/** The most a card lists before the rest is behind the raw view. */
const FACT_LIMIT = 16;

/**
 * One record the assistant looked up, as a person reads one: labelled
 * values first, the whole record a click away. The labels are the record's
 * own field names read as words; the card does not know every entity.
 */
export function EntityCardArtifact({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const card = useMemo(() => entityCardFrom(artifact), [artifact]);
  const [raw, setRaw] = useState(false);
  const shown = card.facts.slice(0, FACT_LIMIT);

  return (
    <div className="animate-rise flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto p-4">
      <div className="flex min-w-0 items-center gap-2">
        <span className="text-muted-foreground flex min-w-0 items-center gap-1.5 text-xs">
          <span className="shrink-0">{humanizeToolName(card.entity)}</span>
          {card.id !== "" && (
            <span className="bg-sunken text-foreground-muted truncate rounded-md px-1.5 py-0.5 font-mono text-2xs tabular-nums">
              {card.id}
            </span>
          )}
        </span>
        {card.path !== "" && (
          <Button
            size="xs"
            variant="ghost"
            className="ml-auto h-6 px-1.5 text-2xs"
            render={<Link to={card.path} />}
          >
            <ArrowUpRightIcon className="size-3" />
            {t("Open")}
          </Button>
        )}
        <Button
          size="xs"
          variant="ghost"
          className={cn("text-muted-foreground h-6 px-1.5 text-2xs", card.path === "" && "ml-auto")}
          onClick={() => setRaw((value) => !value)}
        >
          <CodeIcon className="size-3" />
          {raw ? t("Show fields") : t("Show raw")}
        </Button>
      </div>

      {raw ? (
        <div className="bg-sunken scrollbar-overlay animate-rise max-h-[60vh] overflow-auto rounded-lg p-2">
          <JsonViewer data={card.record as never} collapsed={2} />
        </div>
      ) : shown.length === 0 ? (
        <p className="text-muted-foreground text-sm">
          {t("A structured record; use Show raw to read it.")}
        </p>
      ) : (
        <DescriptionList layout="inline">
          {shown.map((fact) => (
            <DescriptionItem key={fact.key} label={humanizeToolName(fact.key)}>
              <span className="break-words">{fact.value}</span>
            </DescriptionItem>
          ))}
        </DescriptionList>
      )}
      {!raw && card.facts.length > FACT_LIMIT && (
        <p className="text-muted-foreground text-xs">
          {t("{0} more fields in the raw view.", card.facts.length - FACT_LIMIT)}
        </p>
      )}
    </div>
  );
}
