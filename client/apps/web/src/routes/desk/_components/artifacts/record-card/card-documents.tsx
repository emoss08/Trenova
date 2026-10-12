import { useInView } from "@/hooks/use-in-view";
import { describeApiError } from "@/lib/api-error-message";
import { lazyComponent } from "@/lib/lazy-component";
import {
  hasPendingDocumentWork,
  PENDING_DOCUMENTS_REFETCH_MS,
  resourceDocumentsQuery,
} from "@/lib/resource-documents";
import { useQuery } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { Suspense, useMemo } from "react";
import { DocumentRailSkeleton } from "../skeletons/record-card-skeleton";
import { CARD_DOCUMENT_SECTIONS, useCardAccess } from "./card-access";
import { CardLink, CardSection } from "./record-card";
import { documentTiles } from "./record-documents";

/** How far below the screen the section starts reading, so it is ready as it scrolls in. */
const READ_AHEAD = "320px 0px";

const Rail = lazyComponent(
  () => import("./document-rail"),
  (module) => module.DocumentRail,
);

/**
 * A record's documents at the foot of its card, to look at: a rail of tiles
 * that scrolls sideways (see document-rail.tsx). Nothing is read until the
 * section comes within reach of the screen, neither the documents nor the
 * rail's code, so a card opened and closed again costs nothing for them; the
 * section holds the rail's outline until then. Managing the documents
 * (uploading, reviewing, versions) stays on the record's own page.
 *
 * Any card whose record carries documents draws them with this one line. It is
 * left out for someone who may not read documents; the card stands without it.
 */
export function CardDocuments(props: CardDocumentsProps) {
  const access = useCardAccess(CARD_DOCUMENT_SECTIONS);
  return access.documents ? <DocumentsSection {...props} /> : null;
}

type CardDocumentsProps = {
  /** The record kind the documents are attached under, as the documents API names it. */
  resourceType: string;
  resourceId: string;
  /** The record's own page, where its documents are managed. */
  manageTo?: string;
};

function DocumentsSection({ resourceType, resourceId, manageTo }: CardDocumentsProps) {
  const t = useT();
  const [probe, near] = useInView<HTMLDivElement>({ rootMargin: READ_AHEAD, once: true });
  const query = useQuery({
    ...resourceDocumentsQuery(resourceType, resourceId),
    enabled: near && resourceId !== "",
    refetchInterval: (state) =>
      state.state.data?.some(hasPendingDocumentWork) ? PENDING_DOCUMENTS_REFETCH_MS : false,
  });
  const tiles = useMemo(() => (query.data ? documentTiles(query.data, t) : null), [query.data, t]);

  return (
    <CardSection
      title={t("Documents")}
      aside={
        <span className="dk-rec-docs-aside">
          {tiles && tiles.length > 0 && (
            <span>{t("{0, plural, one {# document} other {# documents}}", tiles.length)}</span>
          )}
          {manageTo && <CardLink to={manageTo}>{t("Manage")}</CardLink>}
        </span>
      }
    >
      <div ref={probe} className="dk-rec-docs-body">
        {tiles ? (
          tiles.length > 0 ? (
            <Suspense fallback={<DocumentRailSkeleton />}>
              <Rail.Component tiles={tiles} label={t("Documents")} />
            </Suspense>
          ) : (
            <p className="dk-rec-docs-note">{t("No documents yet.")}</p>
          )
        ) : query.isError ? (
          <p className="dk-rec-docs-note dk-warn">
            {describeApiError(query.error, t("The documents could not be read."))}
          </p>
        ) : (
          <DocumentRailSkeleton />
        )}
      </div>
    </CardSection>
  );
}
