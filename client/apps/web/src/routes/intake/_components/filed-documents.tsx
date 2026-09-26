import { isRecordEntityType, recordPath } from "@/config/record-links";
import { captureItemStatusAttrs, captureRecordKindLabel, isCaptureRecordKind } from "@/lib/capture";
import type { CaptureItem } from "@/lib/graphql/capture";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import { ExternalLinkIcon } from "lucide-react";
import { Link } from "react-router";

function RecordLink({ item }: { item: CaptureItem }) {
  const t = useT();
  const kind = item.filedType;
  if (item.filedId === null || !isCaptureRecordKind(kind)) {
    return <span className="text-foreground-subtle">{t("A record")}</span>;
  }

  const label = item.filedRecord
    ? item.filedRecord.subtitle === ""
      ? item.filedRecord.title
      : `${item.filedRecord.title} · ${item.filedRecord.subtitle}`
    : captureRecordKindLabel(t, kind);

  if (!isRecordEntityType(kind)) {
    return <span>{label}</span>;
  }

  return (
    <Link
      to={recordPath(kind, item.filedId)}
      className="ui-focus-ring text-brand inline-flex items-center gap-1 hover:underline"
    >
      {label}
      <ExternalLinkIcon className="size-3" aria-hidden />
    </Link>
  );
}

/**
 * Documents from the stack that are filed, or on their way. They are fixed:
 * their pages are on a record now, and the place to change them is there.
 */
export function FiledDocuments({ items }: { items: CaptureItem[] }) {
  const t = useT();
  if (items.length === 0) {
    return null;
  }
  const attrs = captureItemStatusAttrs(t);

  return (
    <section aria-label={t("Filed")} className="flex flex-col gap-2">
      <h3 className="text-sm font-semibold">{t("Filed from this stack")}</h3>
      <ul className="border-border divide-border-subtle bg-card divide-y rounded-lg border">
        {items.map((item) => (
          <li
            key={item.id}
            className="flex flex-wrap items-center gap-x-3 gap-y-1 px-3 py-2 text-sm"
          >
            <Badge variant={phaseTone(attrs[item.status].phase)}>{attrs[item.status].text}</Badge>
            <span className="text-foreground-subtle text-xs tabular-nums">
              {t("{0, plural, one {# page} other {# pages}}", item.pageCount)}
            </span>
            <RecordLink item={item} />
            {item.filedAt !== null && (
              <span className="text-foreground-subtle ml-auto text-xs">
                {formatUnixDateTime(item.filedAt)}
              </span>
            )}
          </li>
        ))}
      </ul>
    </section>
  );
}
