import { documentPage } from "@/lib/queries/document-page";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { useObjectUrl } from "@trenova/shared/hooks/use-object-url";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { PageSize } from "./extraction-payload";
import {
  boxInRegion,
  boxStyle,
  pageAspect,
  regionAspect,
  regionImageStyle,
  snippetRegion,
  type PageBox,
} from "./page-geometry";

/** How long a page the server drew is kept once nothing shows it, as the viewer keeps it. */
const PAGE_GC_MS = 5 * 60 * 1000;

type SnippetProps = {
  documentId: string;
  page: number;
  pages: readonly PageSize[];
  /** The boxes to outline, all on this page. */
  boxes: readonly PageBox[];
  /** What the snippet shows, for a screen reader and the button's name. */
  label: string;
  /** Doubtful values are outlined amber; settled ones in the selection's colour. */
  doubtful: boolean;
  onShow: () => void;
};

/**
 * The part of the page a value was read from, at the size its text is read at, with the
 * value outlined: the reading and its source side by side without leaving the list. It draws
 * the same page image the viewer does, so opening the page after costs nothing. Pressing it
 * finds the value on the page.
 */
export function ExtractionSnippet({
  documentId,
  page,
  pages,
  boxes,
  label,
  doubtful,
  onShow,
}: SnippetProps) {
  const t = useT();
  const query = useQuery({
    ...documentPage.image(documentId, page),
    staleTime: Number.POSITIVE_INFINITY,
    gcTime: PAGE_GC_MS,
    retry: 1,
  });
  const url = useObjectUrl(query.data);
  if (boxes.length === 0 || query.isError) return null;

  const region = snippetRegion(boxes);
  const size = pages.find((candidate) => candidate.number === page);
  const aspect = regionAspect(region, pageAspect(size));

  return (
    <Button
      variant="bare"
      size="bare"
      className={cn("dk-ex-snip ui-focus-ring", url === null && "ui-shimmer")}
      style={{ aspectRatio: aspect }}
      aria-label={t("Show {0} on page {1}", label, page)}
      onClick={onShow}
    >
      {url !== null && <img src={url} alt="" draggable={false} style={regionImageStyle(region)} />}
      {url !== null &&
        boxes.map((box, index) => (
          <i
            key={index}
            aria-hidden
            className={cn("dk-ex-snip-box", doubtful && "dk-warn")}
            style={boxStyle(boxInRegion(box, region))}
          />
        ))}
      <span className="dk-ex-snip-pg" aria-hidden>
        {t("p. {0}", page)}
      </span>
    </Button>
  );
}
