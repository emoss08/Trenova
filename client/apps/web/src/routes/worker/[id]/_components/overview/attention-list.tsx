import { HoverGlideGroup } from "@/components/hover-glide";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import type { WorkerConcern } from "@/lib/graphql/worker-overview";
import {
  workerConcernTarget,
  workerConcernWords,
  workerDestinationTarget,
} from "@/lib/worker-concerns";
import type { WorkerTarget } from "@/routes/worker/use-worker-state";
import type { WorkerConcernCode, WorkerConcernSeverity } from "@trenova/graphql/generated/graphql";
import {
  Beaker02Icon,
  Building02Icon,
  ClipboardCheckIcon,
  FileSearch02Icon,
  GraduationHat01Icon,
  IdCardIcon,
  ShieldAlertIcon,
  SlashCircle01Icon,
  Star01Icon,
  UserX01Icon,
  type IconComponent,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";

const CODE_ICONS: Record<WorkerConcernCode, IconComponent> = {
  NotEmployed: UserX01Icon,
  NotAssignable: SlashCircle01Icon,
  CredentialExpired: IdCardIcon,
  CredentialMissing: IdCardIcon,
  CredentialExpiring: IdCardIcon,
  TrainingExpired: GraduationHat01Icon,
  TrainingMissing: GraduationHat01Icon,
  TrainingOverdue: GraduationHat01Icon,
  TrainingDue: GraduationHat01Icon,
  SafetyRating: ShieldAlertIcon,
  OutOfService: ShieldAlertIcon,
  ActiveDiscipline: ShieldAlertIcon,
  OpenSafetyEvent: ShieldAlertIcon,
  ChecklistOverdue: ClipboardCheckIcon,
  ChecklistOpen: ClipboardCheckIcon,
  ReviewOverdue: Star01Icon,
  ReviewDue: Star01Icon,
  ClearinghouseQueryOverdue: FileSearch02Icon,
  ClearinghouseQueryDue: FileSearch02Icon,
  DrugAlcoholProhibited: Beaker02Icon,
  EmploymentVerificationOutstanding: Building02Icon,
};

const SEVERITY_ORDER: readonly WorkerConcernSeverity[] = ["Critical", "Warning", "Info"];

const WELL_TONES: Record<WorkerConcernSeverity, string> = {
  Critical: "border-danger-border bg-danger-subtle text-danger-subtle-foreground",
  Warning: "border-warning-border bg-warning-subtle text-warning-subtle-foreground",
  Info: "border-neutral-border bg-neutral-subtle text-neutral-subtle-foreground",
};

const DOT_TONES: Record<WorkerConcernSeverity, string> = {
  Critical: "bg-danger",
  Warning: "bg-warning",
  Info: "bg-neutral",
};

function groupLabel(severity: WorkerConcernSeverity, t: TranslateFn): string {
  switch (severity) {
    case "Critical":
      return t("Needs action");
    case "Warning":
      return t("Coming due");
    case "Info":
      return t("For information");
  }
}

function ConcernRow({
  concern,
  primary,
  onOpen,
}: {
  concern: WorkerConcern;
  primary: boolean;
  onOpen: (target: WorkerTarget) => void;
}) {
  const t = useT();
  const words = workerConcernWords(concern, t);
  const Icon = CODE_ICONS[concern.code];
  return (
    <li
      data-glide-target
      data-concern={concern.code}
      className="relative grid grid-cols-[1.75rem_minmax(0,1fr)] items-start gap-x-3 gap-y-2 rounded-md px-2 py-2.5 @[34rem]/attention:grid-cols-[1.75rem_minmax(0,1fr)_auto]"
    >
      <span
        aria-hidden
        className={cn(
          "grid size-7 place-items-center rounded-md border",
          WELL_TONES[concern.severity],
        )}
      >
        <Icon className="size-3.5" />
      </span>
      <span className="flex min-w-0 flex-col gap-0.5">
        <button
          type="button"
          onClick={() => onOpen(workerDestinationTarget(concern.destination))}
          className="w-fit rounded-xs ui-focus-ring text-left text-sm font-medium"
        >
          {words.title}
        </button>
        {words.detail ? (
          <span className="max-w-[72ch] text-xs text-muted-foreground">{words.detail}</span>
        ) : null}
      </span>
      <Button
        type="button"
        size="sm"
        variant={primary ? "default" : "secondary"}
        className="col-start-2 justify-self-start @[34rem]/attention:col-start-auto"
        onClick={() => onOpen(workerConcernTarget(concern))}
      >
        {words.action}
      </Button>
    </li>
  );
}

/**
 * What on the record needs someone, grouped by how soon: Needs action, Coming due, For
 * information, the server's worst-first order kept inside each group. Each row names the
 * thing, says what its dates mean and offers the one step that moves it; the first row's step
 * is the solid one.
 */
export function AttentionList({
  concerns,
  concernCount,
  onOpen,
}: {
  concerns: readonly WorkerConcern[];
  concernCount: number;
  onOpen: (target: WorkerTarget) => void;
}) {
  const t = useT();
  const groups = SEVERITY_ORDER.map((severity) => ({
    severity,
    items: concerns.filter((concern) => concern.severity === severity),
  })).filter((group) => group.items.length > 0);
  const first = groups[0]?.items[0];
  const hidden = Math.max(0, concernCount - concerns.length);

  return (
    <SectionPanel title={t("Needs attention")} count={concernCount}>
      {concerns.length === 0 ? (
        <SectionPanelQuiet>{t("Nothing needs attention right now.")}</SectionPanelQuiet>
      ) : (
        <ScrollArea className="max-h-[34rem]">
          <HoverGlideGroup className="@container/attention bg-card px-1.5 pb-1.5">
            {groups.map((group) => (
              <section key={group.severity} aria-label={groupLabel(group.severity, t)}>
                <h4 className="flex items-center gap-1.5 px-2 pt-3 pb-1 text-xs font-medium text-foreground-subtle">
                  <span
                    aria-hidden
                    className={cn("size-1.5 rounded-full", DOT_TONES[group.severity])}
                  />
                  {groupLabel(group.severity, t)}
                </h4>
                <ul className="flex flex-col gap-px">
                  {group.items.map((concern, index) => (
                    <ConcernRow
                      key={`${concern.code}-${concern.destination.recordId ?? index}`}
                      concern={concern}
                      primary={concern === first}
                      onOpen={onOpen}
                    />
                  ))}
                </ul>
              </section>
            ))}
            {hidden > 0 ? (
              <p className="px-2 pt-2 pb-1 text-xs text-muted-foreground">
                {t(
                  "{0, plural, one {# more is not shown here} other {# more are not shown here}}. Each tab lists its own.",
                  hidden,
                )}
              </p>
            ) : null}
          </HoverGlideGroup>
        </ScrollArea>
      )}
    </SectionPanel>
  );
}
