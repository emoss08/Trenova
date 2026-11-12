import type { WatchtowerPanel } from "@/lib/graphql/watchtower";
import { Badge } from "@trenova/shared/components/ui/badge";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { workNoteGist } from "../work-note";
import { PanelBody, PanelCard } from "./panel-parts";

type AskPanelData = NonNullable<WatchtowerPanel["ask"]>;

/**
 * An agent that stopped and asked for a person: what kind of thing stopped
 * it, and what it had tried, in one plain sentence of its own words.
 */
export function AskPanel({ ask }: { ask: AskPanelData }) {
  const t = useT();
  const tried = workNoteGist(ask.tried);

  return (
    <PanelCard
      eyebrow={t("An agent stopped")}
      title={ask.category || t("It needs a person to go on")}
      badge={
        <Badge variant="warning" dot>
          {t("Needs a person")}
        </Badge>
      }
    >
      {tried !== "" && (
        <PanelBody>
          <p className="flex items-start gap-2 border-l-2 border-dsk-b-strong pl-3 text-dsk-base text-dsk-fg2">
            <NovaBadge size="xs" className="mt-0.5 flex-none" />
            <span>{tried}</span>
          </p>
        </PanelBody>
      )}
    </PanelCard>
  );
}
