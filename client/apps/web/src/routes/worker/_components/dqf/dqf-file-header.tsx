import { InfoPopover } from "@/components/info-popover";
import type { DQFFile } from "@/lib/graphql/worker-dqf";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { dqfSectionProgress, dqfSectionTab, type DQFSectionValue } from "@trenova/shared/lib/dqf";
import { cn } from "@trenova/shared/lib/utils";
import { PlusIcon } from "lucide-react";
import { useMemo } from "react";

type DQFFileHeaderProps = {
  file: DQFFile;
  canCreate: boolean;
  onAddEmployer: () => void;
  onOpenTab: (tab: string) => void;
};

const SPINE_LABELS: Record<DQFSectionValue, string> = {
  Credentials: "Licences & reviews",
  Documents: "Documents",
  SafetyHistory: "Employers",
  DrugAlcohol: "Drug & alcohol",
};

/**
 * The verdict, then the spine of the file: one block per section with how
 * much of it is on file and a thin bar of what is settled, warning and
 * missing. The file is assembled on read, so every figure here is the truth
 * of the areas it points at, never a copy.
 */
export function DQFFileHeader({ file, canCreate, onAddEmployer, onOpenTab }: DQFFileHeaderProps) {
  const spine = useMemo(() => dqfSectionProgress(file.items), [file.items]);

  return (
    <div data-testid="dqf-header" className="flex flex-col gap-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="text-sm font-semibold">Driver qualification file</h3>
            <Badge variant={file.complete ? "active" : "inactive"}>
              {file.complete ? "Complete" : "Incomplete"}
            </Badge>
            <InfoPopover title="Driver qualification file">
              <p>
                Complete means every required item is on file and in date. Expired, missing and
                outstanding items block; expiring soon only warns, because the document on file is
                still valid today.
              </p>
              <p>
                A previous employer who never answers still settles once the chases are on record:
                the rule asks for a good-faith effort and a record of it, not an answer nobody can
                compel.
              </p>
            </InfoPopover>
          </div>
          <p className="text-muted-foreground mt-0.5 text-xs">
            Assembled on read from the credentials, documents, previous-employer investigations and
            testing record. 49 CFR 391.51.
          </p>
        </div>
        {canCreate ? (
          <Button size="sm" onClick={onAddEmployer}>
            <PlusIcon className="size-3.5" />
            Add previous employer
          </Button>
        ) : null}
      </div>

      <ol data-testid="dqf-spine" className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        {spine.map((section) => {
          const tab = dqfSectionTab(section.section);
          const body = (
            <>
              <div className="flex items-baseline justify-between gap-2">
                <span className="text-muted-foreground truncate text-[11px] font-semibold uppercase">
                  {SPINE_LABELS[section.section]}
                </span>
                <span className="text-sm font-semibold tabular-nums">
                  {section.satisfied}/{section.total}
                </span>
              </div>
              <SpineBar section={section} />
            </>
          );
          const className =
            "border-border/80 flex w-full flex-col gap-2 rounded-lg border p-3 text-left transition-colors";
          return (
            <li key={section.section} data-testid={`dqf-spine-${section.section}`}>
              {tab ? (
                <button
                  type="button"
                  className={cn(className, "hover:border-border hover:bg-muted/30")}
                  onClick={() => onOpenTab(tab)}
                  aria-label={`Open ${SPINE_LABELS[section.section].toLowerCase()}`}
                >
                  {body}
                </button>
              ) : (
                <div className={className}>{body}</div>
              )}
            </li>
          );
        })}
      </ol>

      <dl className="flex flex-wrap items-center gap-x-6 gap-y-2 rounded-lg border px-4 py-3 text-xs">
        <Count label="Missing" value={file.missingRequired} />
        <Count label="Expired" value={file.expired} />
        <Count label="Outstanding" value={file.outstanding} />
        <Count label="Expiring soon" value={file.expiringSoon} />
        <div className="ml-auto flex min-w-0 flex-col gap-0.5 text-right">
          {file.safetyHistoryDueAt > 0 ? (
            <span className="text-muted-foreground flex items-center justify-end gap-1.5">
              Previous-employer investigation was due {formatUnixDate(file.safetyHistoryDueAt)}
              {file.safetyHistoryLate ? (
                <Badge variant="inactive" title="49 CFR 391.23(c)(1)">
                  Late
                </Badge>
              ) : null}
            </span>
          ) : null}
          {file.retentionExpiresAt ? (
            <span className="text-muted-foreground">
              {file.purgeEligible
                ? `Held past retention since ${formatUnixDate(file.retentionExpiresAt)}`
                : `Hold until ${formatUnixDate(file.retentionExpiresAt)} · 49 CFR 391.51(d)`}
            </span>
          ) : null}
        </div>
      </dl>
    </div>
  );
}

function SpineBar({
  section,
}: {
  section: { total: number; satisfied: number; warning: number; blocking: number };
}) {
  if (section.total === 0) {
    return <div className="bg-muted h-1.5 rounded-sm" aria-hidden />;
  }
  const pct = (value: number) => `${(value / section.total) * 100}%`;
  return (
    <div className="bg-muted flex h-1.5 gap-px overflow-hidden rounded-sm" aria-hidden>
      {section.satisfied > 0 ? (
        <span className="bg-primary/60 h-full" style={{ width: pct(section.satisfied) }} />
      ) : null}
      {section.warning > 0 ? (
        <span className="bg-warning h-full" style={{ width: pct(section.warning) }} />
      ) : null}
      {section.blocking > 0 ? (
        <span className="bg-destructive/70 h-full" style={{ width: pct(section.blocking) }} />
      ) : null}
    </div>
  );
}

function Count({ label, value }: { label: string; value: number }) {
  return (
    <div className="flex flex-col">
      <dt className="text-2xs text-muted-foreground uppercase">{label}</dt>
      <dd className="font-medium tabular-nums">{value}</dd>
    </div>
  );
}
