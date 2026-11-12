import type { WatchtowerPanel } from "@/lib/graphql/watchtower";
import { File06Icon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatFactTime } from "../../watchtower/lanes";
import { PanelBody, PanelCard } from "./panel-parts";

type FilePanelData = NonNullable<WatchtowerPanel["file"]>;

/** A file that arrived and could not be processed: which file, when, and what stopped it. */
export function FilePanel({
  file,
  now,
  timezone,
}: {
  file: FilePanelData;
  now: number;
  timezone: string;
}) {
  const t = useT();

  return (
    <PanelCard
      eyebrow={
        file.receivedAt > 0
          ? t("File · received {0}", formatFactTime(t, file.receivedAt, now, timezone))
          : t("File")
      }
      title={
        <span className="flex items-center gap-2">
          <File06Icon aria-hidden className="size-4 flex-none text-dsk-subtle" />
          <span className="truncate">{file.name || t("A file with no name")}</span>
        </span>
      }
      badge={
        <Badge variant="warning" dot>
          {t("Set aside")}
        </Badge>
      }
    >
      {file.reason !== "" && (
        <PanelBody>
          <p className="text-dsk-base text-dsk-fg2">{file.reason}</p>
        </PanelBody>
      )}
    </PanelCard>
  );
}
