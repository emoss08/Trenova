import { DocumentFileTypeIcon } from "@/components/documents/document-file-type-icon";
import { canObserveIntersections, useInView } from "@/hooks/use-in-view";
import { documentContentUrl } from "@/services/document";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { memo, useCallback, useEffect, useRef, useState } from "react";
import { CardIcon } from "./record-card";
import { documentReviewLabel, type DocumentTile } from "./record-documents";

/** How many tiles the rail draws at a time; the next join as the reader nears its end. */
export const RAIL_BATCH = 12;
/** How far ahead of the rail's end the next tiles are drawn, so scrolling never meets a gap. */
const RAIL_LOOKAHEAD = "0px 480px 0px 0px";
/** How much of the rail one press of an arrow scrolls. */
const RAIL_STEP = 0.85;

type Edges = { start: boolean; end: boolean };

function edgesOf(rail: HTMLElement): Edges {
  return {
    start: rail.scrollLeft <= 1,
    end: rail.scrollLeft + rail.clientWidth >= rail.scrollWidth - 1,
  };
}

/** The same edges kept as they were, so a scroll that reaches neither end draws nothing again. */
function sameEdges(current: Edges, next: Edges): Edges {
  return current.start === next.start && current.end === next.end ? current : next;
}

/**
 * A record's documents in one row that scrolls sideways, however many there
 * are: each a tile with its preview, or its file's mark until the server has
 * drawn one, named for its type and opening the document itself in a new tab.
 * Every tile is the same size whatever it says: three lines of one line each,
 * cut short with an ellipsis, and its review marked on the preview.
 * The rail draws a batch of tiles and the next as the reader scrolls toward
 * its end, and each preview is fetched only as its tile comes near, so a
 * record with hundreds of documents costs what the reader looks at. Arrows
 * at either end step through it for a pointer without a sideways wheel.
 */
export function DocumentRail({ tiles, label }: { tiles: readonly DocumentTile[]; label: string }) {
  const t = useT();
  const [rail, setRail] = useState<HTMLUListElement | null>(null);
  const [drawn, setDrawn] = useState(RAIL_BATCH);
  const [edges, setEdges] = useState<Edges>({ start: true, end: true });
  const drawNext = useCallback(() => setDrawn((count) => count + RAIL_BATCH), []);
  const [endRef] = useInView<HTMLLIElement>({
    root: rail,
    rootMargin: RAIL_LOOKAHEAD,
    onEnter: drawNext,
  });
  // A browser that cannot tell when the end comes near draws every tile.
  const shown = canObserveIntersections() ? Math.min(drawn, tiles.length) : tiles.length;

  // Scrolling reports far more often than the screen draws, so the rail is
  // measured at most once a frame, and the card redraws only when an end is
  // reached or left, never while the rail glides between them.
  const frame = useRef<number | null>(null);
  const measure = useCallback(() => {
    if (!rail || frame.current !== null) return;
    frame.current = requestAnimationFrame(() => {
      frame.current = null;
      setEdges((current) => sameEdges(current, edgesOf(rail)));
    });
  }, [rail]);

  useEffect(
    () => () => {
      if (frame.current !== null) cancelAnimationFrame(frame.current);
    },
    [],
  );

  // The rail is measured as the browser reports it: when it is first laid out
  // and each time it is resized, as it scrolls, and as a pointer or the focus
  // comes to it, which is when its arrows are wanted.
  useEffect(() => {
    if (!rail || typeof ResizeObserver === "undefined") return undefined;
    const observer = new ResizeObserver(measure);
    observer.observe(rail);
    return () => observer.disconnect();
  }, [rail, measure]);

  // The rail's own scroll-behavior glides the step, or jumps under reduced motion.
  const step = (direction: 1 | -1) => {
    rail?.scrollBy?.({ left: direction * rail.clientWidth * RAIL_STEP });
  };

  return (
    <div
      className={cn("dk-rec-docs", !edges.start && "dk-from", !edges.end && "dk-more")}
      onPointerEnter={measure}
      onFocus={measure}
    >
      <div className="dk-rec-docs-clip">
        <ul ref={setRail} className="dk-rec-docs-rail" aria-label={label} onScroll={measure}>
          {tiles.slice(0, shown).map((tile, index) => (
            <li key={tile.id}>
              <DocumentTileLink tile={tile} delay={(index % RAIL_BATCH) * 30} />
            </li>
          ))}
          {shown < tiles.length && (
            // A new marker for each batch: the browser reports each one as it is first watched,
            // so the next batch is drawn while the end is still in reach, and no sooner.
            <li key={shown} ref={endRef} className="dk-rec-docs-end" aria-hidden />
          )}
        </ul>
      </div>
      {!edges.start && (
        <Button
          variant="quiet"
          size="icon-xs"
          className="dk-rec-docs-arrow dk-prev"
          title={t("Earlier documents")}
          aria-label={t("Earlier documents")}
          onClick={() => step(-1)}
        >
          <CardIcon name="back" size={12} stroke={2.2} />
        </Button>
      )}
      {!edges.end && (
        <Button
          variant="quiet"
          size="icon-xs"
          className="dk-rec-docs-arrow dk-next"
          title={t("More documents")}
          aria-label={t("More documents")}
          onClick={() => step(1)}
        >
          <CardIcon name="forward" size={12} stroke={2.2} />
        </Button>
      )}
    </div>
  );
}

/** One tile; unchanged tiles are not drawn again as the rail scrolls or grows. */
const DocumentTileLink = memo(function DocumentTileLink({
  tile,
  delay,
}: {
  tile: DocumentTile;
  delay: number;
}) {
  const t = useT();
  const [broken, setBroken] = useState(false);
  const review = documentReviewLabel(tile.review, t);
  return (
    <a
      className={cn("dk-rec-doc", tile.pending && "dk-pending")}
      href={documentContentUrl(tile.id, "view")}
      target="_blank"
      rel="noopener noreferrer"
      title={t("Open {0}", tile.name)}
      style={{ animationDelay: `${delay}ms` }}
    >
      <span className="dk-rec-doc-p">
        {tile.preview && !broken ? (
          <img
            src={documentContentUrl(tile.id, "preview")}
            alt=""
            loading="lazy"
            decoding="async"
            draggable={false}
            onError={() => setBroken(true)}
          />
        ) : (
          <DocumentFileTypeIcon fileType={tile.fileType} fileName={tile.name} size="lg" />
        )}
        {tile.code && <em className="dk-rec-doc-code">{tile.code}</em>}
        {tile.pending && <em className="dk-rec-doc-wait">{t("Preparing…")}</em>}
        {review && <i className={cn("dk-rec-doc-rv", `dk-${tile.review}`)}>{review}</i>}
      </span>
      <span className="dk-rec-doc-b">
        <b>{tile.type || tile.name}</b>
        <span className={cn(!tile.type && "dk-faint")}>
          {tile.type ? tile.name : t("Unclassified")}
        </span>
        <em>{tile.meta}</em>
      </span>
    </a>
  );
});
