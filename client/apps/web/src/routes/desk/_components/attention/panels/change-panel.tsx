import type { WatchtowerPanel } from "@/lib/graphql/watchtower";
import { ArrowRightIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatFactTime } from "../../watchtower/lanes";
import { PanelBody, PanelCard } from "./panel-parts";
import { agoOrIn } from "./panel-time";

type ChangePanelData = NonNullable<WatchtowerPanel["change"]>;

/**
 * Something on a record changed: whose record, what changed, and the value it
 * had beside the value it has now. A carrier's authority, an agent's score.
 */
export function ChangePanel({
  change,
  now,
  timezone,
}: {
  change: ChangePanelData;
  now: number;
  timezone: string;
}) {
  const t = useT();
  const values = change.from !== "" || change.to !== "";

  return (
    <PanelCard
      eyebrow={change.ref !== "" ? t("{0} · {1}", change.what, change.ref) : change.what}
      title={change.who}
      badge={
        change.at > 0 ? (
          <span className="flex-none text-xs text-dsk-subtle">{agoOrIn(t, change.at, now)}</span>
        ) : undefined
      }
    >
      <PanelBody>
        {values ? (
          <div className="flex items-center gap-3">
            <Value label={t("Was")} value={change.from || t("Not known")} quiet />
            <ArrowRightIcon aria-hidden className="size-4 flex-none text-dsk-faint" />
            <Value label={t("Now")} value={change.to || t("Not known")} />
          </div>
        ) : (
          change.at > 0 && (
            <p className="text-dsk-base text-dsk-fg2">
              {t("Changed {0}.", formatFactTime(t, change.at, now, timezone))}
            </p>
          )
        )}
      </PanelBody>
    </PanelCard>
  );
}

function Value({ label, value, quiet = false }: { label: string; value: string; quiet?: boolean }) {
  return (
    <div className="flex min-w-0 flex-1 flex-col gap-0.5 rounded-[6px] border border-dsk-b-sub bg-dsk-sunken px-3 py-2">
      <span className="text-xs text-dsk-subtle">{label}</span>
      <span
        className={
          quiet
            ? "truncate text-lg/tight font-medium text-dsk-subtle tabular-nums line-through decoration-1"
            : "truncate text-lg/tight font-semibold text-dsk-fg tabular-nums"
        }
      >
        {value}
      </span>
    </div>
  );
}
