import type { WatchtowerPanel } from "@/lib/graphql/watchtower";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatFactTime } from "../../watchtower/lanes";
import { Figure, Figures, PanelCard } from "./panel-parts";
import { agoOrIn } from "./panel-time";

type BooksPanelData = NonNullable<WatchtowerPanel["books"]>;

/**
 * The accounting system, and what is wrong between it and Trenova: documents
 * or payments waiting on a person, a connection that stopped answering, or an
 * authorization about to run out. It says which, and since when.
 */
export function BooksPanel({
  books,
  now,
  timezone,
}: {
  books: BooksPanelData;
  now: number;
  timezone: string;
}) {
  const t = useT();
  const expiring = books.expiresAt > 0;

  return (
    <PanelCard
      eyebrow={t("Books")}
      title={books.system}
      badge={
        books.down ? (
          <Badge variant="danger" dot>
            {t("Not syncing")}
          </Badge>
        ) : expiring ? (
          <Badge variant="warning" dot>
            {t("Authorization expiring")}
          </Badge>
        ) : (
          <Badge variant="warning" dot>
            {t("{0, plural, one {# waiting} other {# waiting}}", books.count)}
          </Badge>
        )
      }
    >
      <Figures>
        {books.count > 0 && <Figure label={t("Waiting on a person")} value={books.count} />}
        {books.since > 0 && (
          <Figure
            label={books.down ? t("Stopped") : t("Oldest")}
            value={formatFactTime(t, books.since, now, timezone)}
            note={agoOrIn(t, books.since, now)}
            tone={books.down ? "bad" : undefined}
          />
        )}
        {expiring && (
          <Figure
            label={t("Expires")}
            value={formatFactTime(t, books.expiresAt, now, timezone)}
            note={agoOrIn(t, books.expiresAt, now)}
            tone={books.expiresAt <= now ? "bad" : "warn"}
          />
        )}
        {books.down && <Figure label={t("Meanwhile")} value={t("Nothing is lost")} />}
      </Figures>
    </PanelCard>
  );
}
