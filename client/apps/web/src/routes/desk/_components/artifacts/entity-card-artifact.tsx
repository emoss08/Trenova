import { DisplayValue } from "@/components/assistant/display-value";
import { humanizeToolName } from "@/components/assistant/proposal-state";
import { isDetailType, isFigureType } from "@/components/assistant/readable-values";
import { ArtifactNotice } from "@/components/assistant/voice/artifact-chrome";
import type { AssistantArtifact } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { useT } from "@trenova/shared/i18n/use-t";
import { ArrowUpRightIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { Link } from "react-router";
import { entityCardFrom } from "./artifact-payloads";

/** The most a card lists before the rest wait behind "Show more". */
const FIELD_LIMIT = 12;

/**
 * One record the assistant looked up, as a person reads one: its values
 * labelled and formatted — a status as a badge, a date in the reader's own
 * timezone, money as money — with the prose and measurements set out below
 * them, and the way into the record itself. Its id is how the card opens and
 * is never shown.
 */
export function EntityCardArtifact({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const card = useMemo(() => entityCardFrom(artifact), [artifact]);
  const [everything, setEverything] = useState(false);

  const facts = card.fields.filter((field) => !isDetailType(field.type));
  const detail = card.fields.filter((field) => isDetailType(field.type));
  const shown = everything ? facts : facts.slice(0, FIELD_LIMIT);
  const more = facts.length - shown.length;

  return (
    <div className="animate-rise flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto p-4">
      <div className="flex min-w-0 items-center gap-2">
        <span className="text-foreground-subtle truncate text-xs">
          {humanizeToolName(card.entity)}
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
      </div>

      {card.fields.length === 0 ? (
        <ArtifactNotice kind={artifact.kind}>
          {card.path === ""
            ? t("This record has nothing more to show here.")
            : t("This record has nothing more to show here; open it to see all of it.")}
        </ArtifactNotice>
      ) : (
        <>
          {facts.length > 0 && (
            <div className="flex flex-col gap-2">
              <DescriptionList layout="inline">
                {shown.map((field, index) => (
                  <DescriptionItem
                    key={field.key}
                    label={field.label}
                    numeric={isFigureType(field.type)}
                    className={index >= FIELD_LIMIT ? "animate-rise" : undefined}
                    valueClassName={index >= FIELD_LIMIT ? "animate-rise" : undefined}
                  >
                    <DisplayValue type={field.type} value={field.value} label={field.label} />
                  </DescriptionItem>
                ))}
              </DescriptionList>
              {(more > 0 || everything) && facts.length > FIELD_LIMIT && (
                <Button
                  size="xs"
                  variant="ghost"
                  className="text-foreground-subtle hover:text-foreground -ml-1.5 h-6 w-fit px-1.5 text-2xs"
                  aria-expanded={everything}
                  onClick={() => setEverything((value) => !value)}
                >
                  {everything
                    ? t("Show fewer")
                    : t("{0, plural, one {Show # more field} other {Show # more fields}}", more)}
                </Button>
              )}
            </div>
          )}

          {detail.length > 0 && (
            <DescriptionList
              layout="stacked"
              columns={1}
              className="border-border-subtle border-t pt-4"
            >
              {detail.map((field) => (
                <DescriptionItem key={field.key} label={field.label}>
                  <DisplayValue type={field.type} value={field.value} label={field.label} />
                </DescriptionItem>
              ))}
            </DescriptionList>
          )}
        </>
      )}
    </div>
  );
}
