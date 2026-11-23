import { AccountingSyncStateLine } from "@/components/accounting-sync/sync-state-line";
import {
  MetaItem,
  RecordHeader,
  RecordMark,
  RecordTabs,
  TabCount,
  TabDot,
  UpdatedAgo,
} from "@/components/record-header";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import type { WorkerOverview } from "@/lib/graphql/worker-overview";
import type { WorkerRecord, WorkerTab, WorkerTarget } from "@/routes/worker/use-worker-state";
import {
  Briefcase01Icon,
  CheckIcon,
  ClockIcon,
  Copy01Icon,
  MarkerPin01Icon,
  Phone01Icon,
} from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { TabsTab } from "@trenova/shared/components/ui/tabs";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useNow } from "@trenova/shared/hooks/use-now";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatPhoneForDisplay } from "@trenova/shared/lib/phone";
import { WORKER_STANDING_LABELS } from "@trenova/shared/types/worker-overview";
import { WorkerRecordMenus } from "./add-to-record-menu";
import { workerTabLabel, workerTabMarks, workerTenure } from "./overview-model";

type Standing = WorkerOverview["standing"];

const STANDING_TONE: Record<Standing, "success" | "warning" | "danger"> = {
  Good: "success",
  Watch: "warning",
  AtRisk: "danger",
  Blocked: "danger",
};

function standingExplanation(standing: Standing, concerns: number, t: TranslateFn): string {
  const reason =
    standing === "Good"
      ? t("Nothing on the record stands in the way of putting them on a load.")
      : standing === "Watch"
        ? t("Something on the record comes due soon or needs a look.")
        : standing === "AtRisk"
          ? t("Something on the record has already lapsed.")
          : t("They cannot be put on a load today.");
  const count =
    concerns > 0
      ? t(
          "{0, plural, one {# thing on the record needs attention.} other {# things on the record need attention.}}",
          concerns,
        )
      : null;
  return [reason, count].filter(Boolean).join(" ");
}

function driverTypeShort(driverType: WorkerRecord["driverType"], t: TranslateFn): string {
  switch (driverType) {
    case "Local":
      return t("Local");
    case "Regional":
      return t("Regional");
    case "OTR":
      return t("OTR");
    case "Team":
      return t("Team");
  }
}

/** The phone on the meta line, with a copy button that shows on hover without moving anything. */
function PhoneItem({ phone }: { phone: string }) {
  const t = useT();
  const { copy, isCopied } = useCopyToClipboard();
  return (
    <span className="group/phone inline-flex min-w-0 items-center gap-1.5">
      <Phone01Icon aria-hidden className="size-3 shrink-0" />
      <span className="truncate tabular-nums">{formatPhoneForDisplay(phone)}</span>
      <button
        type="button"
        aria-label={t("Copy phone number")}
        title={t("Copy")}
        onClick={() => void copy(phone)}
        className="grid size-4 shrink-0 place-items-center rounded-sm ui-focus-ring text-foreground-subtle opacity-0 transition-opacity group-hover/phone:opacity-100 hover:text-foreground focus-visible:opacity-100 data-[copied=true]:opacity-100"
        data-copied={isCopied}
      >
        {isCopied ? <CheckIcon className="size-3" /> : <Copy01Icon className="size-3" />}
      </button>
    </span>
  );
}

export type WorkerHeaderProps = {
  worker: WorkerRecord;
  overview: WorkerOverview | undefined;
  overviewUpdatedAt: number;
  overviewFetching: boolean;
  onRefresh: () => void;
  /** The tabs the plan and the reader's permissions leave drawable, in order. */
  tabs: readonly WorkerTab[];
  tab: WorkerTab;
  onTab: (tab: WorkerTab) => void;
  /** The pointer is on a tab: start reading its code. */
  onTabIntent: (tab: WorkerTab) => void;
  /** The Profile tab holds edits not saved, so its tab is marked from any other. */
  profileUnsaved: boolean;
  /** Opens a place on the page; the page asks first when a work pane holds changes. */
  onOpen: (target: WorkerTarget) => void;
};

/**
 * The worker's record header: their mark, name, status, worker and driver type and standing,
 * the facts a dispatcher reaches for (fleet, position, home, phone, tenure), the accounting sync
 * line, what can be added to the file, and the page's tabs with what each needs.
 */
export function WorkerHeader({
  worker,
  overview,
  overviewUpdatedAt,
  overviewFetching,
  onRefresh,
  tabs,
  tab,
  onTab,
  onTabIntent,
  profileUnsaved,
  onOpen,
}: WorkerHeaderProps) {
  const t = useT();
  const now = useNow("minute");
  const name = worker.wholeName || `${worker.firstName} ${worker.lastName}`.trim();
  const marks = workerTabMarks(overview, t);
  const place = [worker.city, worker.state?.abbreviation].filter(Boolean).join(", ");
  const tenure = workerTenure(
    {
      hireDate: worker.profile?.hireDate ?? null,
      terminationDate: worker.profile?.terminationDate ?? null,
    },
    now,
    t,
  );

  return (
    <RecordHeader
      mark={<RecordMark name={name} />}
      title={name}
      badges={
        <>
          <Badge variant={worker.status === "Active" ? "success" : "neutral"} dot>
            {worker.status === "Active" ? t("Active") : t("Inactive")}
          </Badge>
          <Badge variant="neutral">
            {worker.type === "Employee" ? t("Employee") : t("Contractor")}
          </Badge>
          <Badge variant="neutral">{driverTypeShort(worker.driverType, t)}</Badge>
          {overview ? (
            <Tooltip>
              <TooltipTrigger
                render={
                  <Badge
                    variant={STANDING_TONE[overview.standing]}
                    dot
                    tabIndex={0}
                    className="cursor-default"
                  />
                }
              >
                {WORKER_STANDING_LABELS[overview.standing]}
              </TooltipTrigger>
              <TooltipContent className="flex max-w-72 flex-col gap-0.5">
                <span className="font-medium">{WORKER_STANDING_LABELS[overview.standing]}</span>
                <span className="opacity-80">
                  {standingExplanation(overview.standing, overview.concernCount, t)}
                </span>
                <span className="opacity-80">
                  {t(
                    "The standing reads Good standing, Worth watching, At risk or Cannot be assigned.",
                  )}
                </span>
              </TooltipContent>
            </Tooltip>
          ) : null}
        </>
      }
      meta={
        <>
          {worker.fleetCode ? (
            <MetaItem
              icon={
                <span
                  aria-hidden
                  className="size-2 shrink-0 rounded-[2px]"
                  style={{ backgroundColor: worker.fleetCode.color || undefined }}
                />
              }
            >
              <span className="font-mono">{worker.fleetCode.code}</span>
            </MetaItem>
          ) : null}
          {worker.position ? (
            <MetaItem icon={<Briefcase01Icon />}>{t(worker.position.title)}</MetaItem>
          ) : null}
          {place ? <MetaItem icon={<MarkerPin01Icon />}>{place}</MetaItem> : null}
          {worker.phoneNumber ? <PhoneItem phone={worker.phoneNumber} /> : null}
          {tenure ? <MetaItem icon={<ClockIcon />}>{tenure}</MetaItem> : null}
        </>
      }
      actions={
        <WorkerRecordMenus
          workerId={worker.id}
          hasFleetCode={worker.fleetCode != null}
          terminated={Boolean(worker.profile?.terminationDate)}
          onOpen={onOpen}
        />
      }
    >
      <AccountingSyncStateLine objectId={worker.id} className="-mt-1 text-xs" />
      <RecordTabs<WorkerTab>
        value={tab}
        onValueChange={onTab}
        trailing={
          overviewUpdatedAt > 0 ? (
            <UpdatedAgo at={overviewUpdatedAt} onRefresh={onRefresh} busy={overviewFetching} />
          ) : null
        }
      >
        {tabs.map((key) => {
          const mark = marks[key];
          return (
            <TabsTab key={key} value={key} onPointerEnter={() => onTabIntent(key)}>
              {workerTabLabel(key, t)}
              {key === "profile" && profileUnsaved ? (
                <TabDot tone="warning" label={t("Unsaved changes")} />
              ) : mark?.kind === "dot" ? (
                <TabDot tone={mark.tone} label={mark.label} />
              ) : mark?.kind === "count" ? (
                <TabCount>{mark.value}</TabCount>
              ) : null}
            </TabsTab>
          );
        })}
      </RecordTabs>
    </RecordHeader>
  );
}
