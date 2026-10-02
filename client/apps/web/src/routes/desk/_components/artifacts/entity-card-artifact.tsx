import { DisplayValue } from "@/components/assistant/display-value";
import { humanizeToolName } from "@/components/assistant/proposal-state";
import { isDetailType, isFigureType } from "@/components/assistant/readable-values";
import { ArtifactKindIcon, ArtifactNotice } from "@/components/assistant/voice/artifact-chrome";
import type { AssistantArtifact } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { useT } from "@trenova/shared/i18n/use-t";
import { ArrowUpRightIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { Link } from "react-router";
import { entityCardFrom } from "./artifact-payloads";
import { ArtifactScroll, ArtifactSection } from "./artifact-section";

/** The most a card lists before the rest wait behind "Show more". */
const FIELD_LIMIT = 12;

/**
 * One record the assistant looked up, as a person reads one: its name at the
 * head, which opens the record when it has a page, its values labelled and
 * formatted in two columns — a status as a badge, a date in the reader's own
 * timezone, money as money — and the prose and measurements set out below
 * them. Its id is how the card opens and is never shown.
 */
export function EntityCardArtifact({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const card = useMemo(() => entityCardFrom(artifact), [artifact]);
  const [everything, setEverything] = useState(false);

  const facts = card.fields.filter((field) => !isDetailType(field.type));
  const detail = card.fields.filter((field) => isDetailType(field.type));
  const shown = everything ? facts : facts.slice(0, FIELD_LIMIT);
  const more = facts.length - shown.length;
  const entity = humanizeToolName(card.entity);

  return (
    <ArtifactScroll>
      <header className="flex items-center gap-3">
        <span className="bg-sunken text-foreground-muted flex size-9 shrink-0 items-center justify-center rounded-md">
          <ArtifactKindIcon kind="entity_card" className="size-4" />
        </span>
        <div className="flex min-w-0 flex-1 flex-col gap-0.5">
          {card.path !== "" ? (
            <Link
              to={card.path}
              className="ui-focus-ring inline-flex max-w-full items-center gap-1 self-start rounded-control text-sm font-semibold underline-offset-2 hover:underline"
            >
              <span className="truncate">{artifact.title}</span>
              <ArrowUpRightIcon aria-hidden className="text-foreground-subtle size-3.5 shrink-0" />
            </Link>
          ) : (
            <p className="truncate text-sm font-semibold">{artifact.title}</p>
          )}
          {entity !== "" && <p className="text-foreground-subtle truncate text-xs">{entity}</p>}
        </div>
        {card.path !== "" && (
          <Button size="xs" variant="outline" className="shrink-0" render={<Link to={card.path} />}>
            {t("Open")}
            <ArrowUpRightIcon className="size-3" />
          </Button>
        )}
      </header>

      {card.fields.length === 0 ? (
        <ArtifactNotice kind={artifact.kind}>
          {card.path === ""
            ? t("This record has nothing more to show here.")
            : t("This record has nothing more to show here; open it to see all of it.")}
        </ArtifactNotice>
      ) : (
        <>
          {facts.length > 0 && (
            <ArtifactSection
              title={t("Details")}
              hint={t("{0, plural, one {# field} other {# fields}}", facts.length)}
              action={
                facts.length > FIELD_LIMIT ? (
                  <Button
                    size="xxs"
                    variant="ghost"
                    className="text-foreground-subtle hover:text-foreground -mr-1.5 text-2xs"
                    aria-expanded={everything}
                    onClick={() => setEverything((value) => !value)}
                  >
                    {everything
                      ? t("Show fewer")
                      : t("{0, plural, one {Show # more} other {Show # more}}", more)}
                  </Button>
                ) : undefined
              }
            >
              <DescriptionList layout="stacked" columns={2} className="gap-y-2.5">
                {shown.map((field, index) => (
                  <DescriptionItem
                    key={field.key}
                    label={field.label}
                    numeric={isFigureType(field.type)}
                    className={index >= FIELD_LIMIT ? "animate-rise" : undefined}
                    valueClassName="text-xs"
                  >
                    <DisplayValue type={field.type} value={field.value} label={field.label} />
                  </DescriptionItem>
                ))}
              </DescriptionList>
            </ArtifactSection>
          )}

          {detail.map((field) => (
            <ArtifactSection key={field.key} title={field.label}>
              <div className="text-xs leading-relaxed">
                <DisplayValue type={field.type} value={field.value} label={field.label} />
              </div>
            </ArtifactSection>
          ))}
        </>
      )}
    </ArtifactScroll>
  );
}
