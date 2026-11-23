import { usePermission } from "@/hooks/use-permission";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { appPath } from "@/lib/app-path";
import {
  workerFormTarget,
  workerPath,
  type WorkerForm,
  type WorkerTarget,
} from "@/routes/worker/use-worker-state";
import { useWorkerAreas } from "@/routes/worker/worker-areas";
import { useNavigate } from "@tanstack/react-router";
import {
  Award01Icon,
  Beaker02Icon,
  CalendarRangeIcon,
  ChevronDownIcon,
  ClockRewindIcon,
  Copy01Icon,
  DotsHorizontalIcon,
  File06Icon,
  FileSearch02Icon,
  GraduationHat01Icon,
  IdCardIcon,
  LinkExternal01Icon,
  LogOut01Icon,
  PlusIcon,
  ShieldAlertIcon,
  Star01Icon,
  SunsetIcon,
  type IconComponent,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { Fragment } from "react";

type AddItem = {
  form: WorkerForm;
  label: (t: TranslateFn) => string;
  icon: IconComponent;
  resource: string;
};

type AddGroup = { label: (t: TranslateFn) => string; items: readonly AddItem[] };

/** What can be added to a worker's file from the header, grouped as the file is. */
export const ADD_TO_RECORD_GROUPS: readonly AddGroup[] = [
  {
    label: (t) => t("Employment"),
    items: [
      {
        form: "record-event",
        label: (t) => t("Employment event"),
        icon: ClockRewindIcon,
        resource: Resource.WorkerEmploymentEvent,
      },
    ],
  },
  {
    label: (t) => t("Qualification"),
    items: [
      {
        form: "add-credential",
        label: (t) => t("Credential"),
        icon: IdCardIcon,
        resource: Resource.WorkerCredential,
      },
      {
        form: "record-collection",
        label: (t) => t("Drug or alcohol collection"),
        icon: Beaker02Icon,
        resource: Resource.WorkerDOTTest,
      },
      {
        form: "log-query",
        label: (t) => t("Clearinghouse query"),
        icon: FileSearch02Icon,
        resource: Resource.WorkerDOTTest,
      },
    ],
  },
  {
    label: (t) => t("Performance"),
    items: [
      {
        form: "record-training-result",
        label: (t) => t("Training result"),
        icon: GraduationHat01Icon,
        resource: Resource.WorkerTraining,
      },
      {
        form: "record-safety-event",
        label: (t) => t("Safety event"),
        icon: ShieldAlertIcon,
        resource: Resource.WorkerSafetyEvent,
      },
      {
        form: "add-recognition",
        label: (t) => t("Recognition"),
        icon: Award01Icon,
        resource: Resource.WorkerRecognition,
      },
      {
        form: "start-review",
        label: (t) => t("Review"),
        icon: Star01Icon,
        resource: Resource.PerformanceReview,
      },
    ],
  },
  {
    label: (t) => t("Time"),
    items: [
      {
        form: "request-time-off",
        label: (t) => t("Time off request"),
        icon: SunsetIcon,
        resource: Resource.WorkerPTO,
      },
      {
        form: "open-leave-case",
        label: (t) => t("Leave case"),
        icon: CalendarRangeIcon,
        resource: Resource.WorkerLeave,
      },
    ],
  },
  {
    label: (t) => t("Files"),
    items: [
      {
        form: "upload-document",
        label: (t) => t("Document"),
        icon: File06Icon,
        resource: Resource.Document,
      },
    ],
  },
];

export type WorkerRecordMenusProps = {
  workerId: string;
  /** The worker's fleet code; "Open fleet code" is offered only with one. */
  hasFleetCode: boolean;
  /** Set once a Terminated event has ended the worker's employment. */
  terminated: boolean;
  /** Opens a place on the page; the page asks first when a work pane holds changes. */
  onOpen: (target: WorkerTarget) => void;
};

/**
 * The record header's actions: "Add to record", which opens each kind of entry's form on the tab
 * it belongs to, and the more menu (copy the page's link, the fleet codes page, end employment).
 * An entry is offered only where its place is on the plan and readable and the person may create
 * it.
 */
export function WorkerRecordMenus({
  workerId,
  hasFleetCode,
  terminated,
  onOpen,
}: WorkerRecordMenusProps) {
  const t = useT();
  const navigate = useNavigate();
  const areas = useWorkerAreas();
  const { copy } = useCopyToClipboard();
  const { allowed: canReadFleetCodes } = usePermission(Resource.FleetCode, Operation.Read);
  const allowed: Record<string, boolean> = {
    [Resource.WorkerEmploymentEvent]: usePermission(
      Resource.WorkerEmploymentEvent,
      Operation.Create,
    ).allowed,
    [Resource.WorkerCredential]: usePermission(Resource.WorkerCredential, Operation.Create).allowed,
    [Resource.WorkerDOTTest]: usePermission(Resource.WorkerDOTTest, Operation.Create).allowed,
    [Resource.WorkerTraining]: usePermission(Resource.WorkerTraining, Operation.Create).allowed,
    [Resource.WorkerSafetyEvent]: usePermission(Resource.WorkerSafetyEvent, Operation.Create)
      .allowed,
    [Resource.WorkerRecognition]: usePermission(Resource.WorkerRecognition, Operation.Create)
      .allowed,
    [Resource.PerformanceReview]: usePermission(Resource.PerformanceReview, Operation.Create)
      .allowed,
    [Resource.WorkerPTO]: usePermission(Resource.WorkerPTO, Operation.Create).allowed,
    [Resource.WorkerLeave]: usePermission(Resource.WorkerLeave, Operation.Create).allowed,
    [Resource.Document]: usePermission(Resource.Document, Operation.Create).allowed,
  };

  const offered = (item: AddItem) => {
    const target = workerFormTarget(item.form);
    return (
      allowed[item.resource] === true &&
      areas.isDrawable({ tab: target.tab, view: target.view ?? null })
    );
  };
  const groups = ADD_TO_RECORD_GROUPS.map((group) => ({
    ...group,
    items: group.items.filter(offered),
  })).filter((group) => group.items.length > 0);

  const canEndEmployment =
    !terminated &&
    allowed[Resource.WorkerEmploymentEvent] === true &&
    areas.isDrawable({ tab: "employment", view: "history" });

  return (
    <>
      {groups.length > 0 ? (
        <DropdownMenu>
          <DropdownMenuTrigger render={<Button type="button" size="sm" />}>
            <PlusIcon className="size-3.5" />
            {t("Add to record")}
            <ChevronDownIcon className="size-3.5 opacity-70" />
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-60">
            {groups.map((group, index) => (
              <Fragment key={group.label(t)}>
                {index > 0 ? <DropdownMenuSeparator /> : null}
                <DropdownMenuGroup>
                  <DropdownMenuLabel>{group.label(t)}</DropdownMenuLabel>
                  {group.items.map((item) => {
                    const Icon = item.icon;
                    return (
                      <DropdownMenuItem
                        key={item.form}
                        title={item.label(t)}
                        startContent={<Icon className="size-3.5" />}
                        onClick={() => onOpen(workerFormTarget(item.form))}
                      />
                    );
                  })}
                </DropdownMenuGroup>
              </Fragment>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>
      ) : null}
      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <Button type="button" variant="outline" size="icon-sm" aria-label={t("More actions")} />
          }
        >
          <DotsHorizontalIcon className="size-4" />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-56">
          <DropdownMenuItem
            title={t("Copy link")}
            startContent={<Copy01Icon className="size-3.5" />}
            onClick={() =>
              void copy(`${window.location.origin}${workerPath(workerId)}`, { withToast: true })
            }
          />
          {hasFleetCode && canReadFleetCodes ? (
            <DropdownMenuItem
              title={t("Open fleet codes")}
              startContent={<LinkExternal01Icon className="size-3.5" />}
              onClick={() =>
                void navigate({ to: appPath("/dispatch/configuration-files/fleet-codes") })
              }
            />
          ) : null}
          {canEndEmployment ? (
            <>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                data-variant="destructive"
                title={t("End employment…")}
                startContent={<LogOut01Icon className="size-3.5" />}
                onClick={() => onOpen(workerFormTarget("record-event", "Terminated"))}
              />
            </>
          ) : null}
        </DropdownMenuContent>
      </DropdownMenu>
    </>
  );
}
