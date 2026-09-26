import type {
  AgentPreviewOperation,
  PreviewRecordChange,
  PreviewWarning,
  ProposalPreview as ProposalPreviewData,
} from "@/lib/graphql/agent-preview";
import {
  Alert,
  AlertAction,
  AlertDescription,
  AlertTitle,
} from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  ArchiveIcon,
  CircleAlertIcon,
  InfoIcon,
  LockIcon,
  PencilIcon,
  PlayIcon,
  PlusIcon,
  RefreshCwIcon,
  SendIcon,
  Trash2Icon,
  TriangleAlertIcon,
  type LucideIcon,
} from "lucide-react";
import { useState, type ReactNode } from "react";
import { Link } from "react-router";
import { humanizeToolName } from "../proposal-state";
import { humanizeKey } from "../readable-values";
import { MessagePreview } from "./message-preview";
import { MoneyPreview } from "./money-preview";
import { operationLabel, previewRecordPath } from "./preview-format";
import {
  previewWarningText,
  previewWarningTone,
  visibleWarnings,
  wouldFailReasons,
  type PreviewWarningTone,
  type WouldFailReason,
} from "./preview-warnings";
import { ValueChange } from "./value-change";

export type PreviewDensity = "compact" | "full";

/** How much a compact preview shows before "Show all changes". */
const COMPACT_RECORDS = 2;
const COMPACT_FIELDS = 4;

const OPERATION_ICONS: Record<AgentPreviewOperation, LucideIcon> = {
  Create: PlusIcon,
  Update: PencilIcon,
  Delete: Trash2Icon,
  Archive: ArchiveIcon,
  Send: SendIcon,
  Run: PlayIcon,
};

const WARNING_VARIANT: Record<PreviewWarningTone, "destructive" | "warning" | "info"> = {
  danger: "destructive",
  warning: "warning",
  info: "info",
};

const WARNING_ICON: Record<PreviewWarningTone, LucideIcon> = {
  danger: CircleAlertIcon,
  warning: TriangleAlertIcon,
  info: InfoIcon,
};

/**
 * What a surface can do about a write that would be refused. Each is
 * offered only where the surface can honour it: a chat card can change a
 * value in the editor and ask its agent; a queue can change a value and
 * reject with the reasons; a plan step can only ask.
 */
export type WouldFailActions = {
  /** Whether the person may change the parameter a reason names. */
  canChange?: (param: string) => boolean;
  /** Opens the editor on the parameter the reason names. */
  onChange?: (reason: WouldFailReason) => void;
  /** Asks the agent for a corrected proposal, naming every reason. */
  onAskAgent?: (reasons: WouldFailReason[]) => void;
};

function WarningAlert({
  warning,
  wouldFail,
}: {
  warning: PreviewWarning;
  wouldFail?: WouldFailActions;
}) {
  const t = useT();
  const tone = previewWarningTone(warning);
  const Icon = WARNING_ICON[tone];
  const reasons = wouldFailReasons(warning);
  if (reasons.length > 0) {
    return <WouldFailAlert reasons={reasons} actions={wouldFail} />;
  }

  return (
    <Alert size="sm" variant={WARNING_VARIANT[tone]}>
      <Icon />
      <AlertDescription>{previewWarningText(warning, t)}</AlertDescription>
    </Alert>
  );
}

/**
 * A write its own rules would refuse, reason by reason. The old notice said
 * only that it would not go through; the person needs what is wrong and a
 * way forward: change the value the rule names, where they may, or have the
 * agent fix it.
 */
function WouldFailAlert({
  reasons,
  actions,
}: {
  reasons: WouldFailReason[];
  actions?: WouldFailActions;
}) {
  const t = useT();
  const changeable = (reason: WouldFailReason) =>
    reason.param !== "" &&
    actions?.onChange !== undefined &&
    (actions.canChange?.(reason.param) ?? false);

  return (
    <Alert size="sm" variant="destructive">
      <CircleAlertIcon />
      <AlertTitle>{t("This would not go through as it stands")}</AlertTitle>
      <AlertDescription>
        <ul className="flex w-full flex-col gap-1">
          {reasons.map((reason, index) => (
            <li
              key={`${reason.param}-${index}`}
              className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-0.5"
            >
              <span className="min-w-0 break-words">
                {reason.label !== "" && <span className="font-medium">{reason.label}: </span>}
                {reason.message}
              </span>
              {changeable(reason) && (
                <Button size="xxs" variant="outline" onClick={() => actions?.onChange?.(reason)}>
                  <PencilIcon className="size-3" />
                  {t("Change {0}", reason.label !== "" ? reason.label : paramLabel(reason.param))}
                </Button>
              )}
            </li>
          ))}
        </ul>
      </AlertDescription>
      {actions?.onAskAgent && (
        <AlertAction>
          <Button size="xs" variant="outline" onClick={() => actions.onAskAgent?.(reasons)}>
            {t("Ask the agent to fix it")}
          </Button>
        </AlertAction>
      )}
    </Alert>
  );
}

/** The last name in a parameter path, in words: shipment.moves[0].bol is "Bol". */
function paramLabel(param: string): string {
  const last = param.split(".").at(-1) ?? param;

  return humanizeKey(last.replace(/\[\d+\]$/u, ""));
}

/**
 * The record a proposal was made against has moved or gone. The server
 * refuses to approve it, so the surface says why before anyone reaches for
 * the button.
 */
export function StaleNotice({ missing }: { missing: boolean }) {
  const t = useT();

  return (
    <Alert size="sm" variant="warning">
      <TriangleAlertIcon />
      <AlertTitle>{t("Changed since it was proposed")}</AlertTitle>
      <AlertDescription>
        {missing
          ? t("The record it would change is gone. It can only be rejected now.")
          : t(
              "The record it would change has been edited since. It can only be rejected now; ask the agent again for a fresh proposal.",
            )}
      </AlertDescription>
    </Alert>
  );
}

/** An approval was refused because what it would do moved while the person read it. */
export function ChangedNotice() {
  const t = useT();

  return (
    <Alert size="sm" variant="warning">
      <RefreshCwIcon />
      <AlertDescription>{t("This change looks different now — review it again.")}</AlertDescription>
    </Alert>
  );
}

/**
 * The preview could not be read. Approving stays possible, without a digest,
 * and the decision is recorded as made without one; the notice says so.
 */
export function PreviewFailedNotice({
  onRetry,
  retrying,
}: {
  onRetry: () => void;
  retrying: boolean;
}) {
  const t = useT();

  return (
    <Alert size="sm" variant="warning">
      <TriangleAlertIcon />
      <AlertDescription>
        {t(
          "What this would change could not be loaded. Approving now is recorded as approved without a preview.",
        )}
      </AlertDescription>
      <AlertAction>
        <Button size="xs" variant="outline" onClick={onRetry} isLoading={retrying}>
          {t("Try again")}
        </Button>
      </AlertAction>
    </Alert>
  );
}

export function PreviewSkeleton({ density = "full" }: { density?: PreviewDensity }) {
  const t = useT();

  return (
    <div
      className="flex flex-col gap-2"
      aria-busy="true"
      aria-label={t("Loading what would change")}
    >
      <Skeleton className="h-3.5 w-1/3" />
      <Skeleton className="h-3.5 w-2/3" />
      {density === "full" && <Skeleton className="h-3.5 w-1/2" />}
    </div>
  );
}

/**
 * The read state of a preview around what the preview shows: a skeleton while
 * the first read runs, the failure with a retry (and whatever the surface can
 * say without it), and once it is in, the notice that it moved under an
 * approval, then the preview itself.
 */
export function PreviewLoadState<T>({
  query,
  changed,
  density = "full",
  fallback,
  children,
}: {
  query: {
    data: T | undefined;
    isError: boolean;
    isFetching: boolean;
    refetch: () => Promise<unknown>;
  };
  changed: boolean;
  density?: PreviewDensity;
  fallback?: ReactNode;
  children: (data: T) => ReactNode;
}) {
  if (query.data !== undefined) {
    return (
      <>
        {changed && <ChangedNotice />}
        {children(query.data)}
      </>
    );
  }
  if (query.isError) {
    return (
      <>
        <PreviewFailedNotice onRetry={() => void query.refetch()} retrying={query.isFetching} />
        {fallback}
      </>
    );
  }

  return <PreviewSkeleton density={density} />;
}

function RecordHeader({ change, tool }: { change: PreviewRecordChange; tool: string }) {
  const t = useT();
  const Icon = OPERATION_ICONS[change.operation];
  const path = change.withheld ? null : previewRecordPath(change.record);
  const kind = change.resource !== "" ? humanizeKey(change.resource) : "";

  return (
    <div className="flex min-w-0 flex-wrap items-center gap-x-1.5 gap-y-0.5 text-xs">
      <Icon aria-hidden className="text-foreground-muted size-3.5 shrink-0" />
      <span className="font-medium">{operationLabel(change.operation, t)}</span>
      {change.operation === "Run" ? (
        <span className="min-w-0 break-words">{humanizeToolName(tool)}</span>
      ) : change.withheld ? (
        <span className="text-foreground-subtle inline-flex items-center gap-1">
          <LockIcon aria-hidden className="size-3 shrink-0" />
          {t("A record hidden by your data access")}
        </span>
      ) : (
        <>
          {kind !== "" && <span className="text-foreground-subtle">{kind}</span>}
          {change.label !== "" &&
            (path !== null ? (
              <Link to={path} className="text-brand min-w-0 break-words hover:underline">
                {change.label}
              </Link>
            ) : (
              <span className="min-w-0 break-words">{change.label}</span>
            ))}
        </>
      )}
    </div>
  );
}

function RecordChangeView({
  change,
  tool,
  fieldLimit,
  density,
}: {
  change: PreviewRecordChange;
  tool: string;
  fieldLimit: number | null;
  density: PreviewDensity;
}) {
  const t = useT();
  const fields = fieldLimit === null ? change.fields : change.fields.slice(0, fieldLimit);
  const hiddenHere = change.fields.length - fields.length;

  return (
    <section className="flex min-w-0 flex-col gap-2">
      <RecordHeader change={change} tool={tool} />
      {!change.withheld && (
        <>
          {fields.length > 0 && (
            <DescriptionList layout="inline" className="pl-5">
              {fields.map((field) => (
                <DescriptionItem key={field.path} label={field.label}>
                  <ValueChange change={field} operation={change.operation} />
                </DescriptionItem>
              ))}
            </DescriptionList>
          )}
          {(hiddenHere > 0 || change.omittedFields > 0) && (
            <p className="text-foreground-subtle pl-5 text-xs">
              {hiddenHere > 0
                ? t(
                    "{0, plural, one {# more value} other {# more values}}",
                    hiddenHere + change.omittedFields,
                  )
                : t(
                    "{0, plural, one {# more value is past what a preview shows} other {# more values are past what a preview shows}}",
                    change.omittedFields,
                  )}
            </p>
          )}
          {change.message && (
            <div className="pl-5">
              <MessagePreview message={change.message} density={density} />
            </div>
          )}
          {change.money && (
            <div className="pl-5">
              <MoneyPreview money={change.money} />
            </div>
          )}
        </>
      )}
    </section>
  );
}

/**
 * What a proposal's write would do, record by record, for the person deciding.
 *
 * Every value is the tool's own answer — the same code the write runs —
 * computed for this reader from the world as it is now. What they may not
 * read is hidden and counted rather than dropped, a value that moved since
 * the proposal says what it was, and a tool that cannot say what it would do
 * shows the values it would run with under a notice that says exactly that.
 *
 * `compact` shows the first records and values with "Show all changes" for
 * the rest; `full` shows everything.
 */
export function ProposalPreview({
  preview,
  density = "full",
  inPlan = false,
  wouldFail,
}: {
  preview: ProposalPreviewData;
  density?: PreviewDensity;
  /** Inside a plan the step draws its own dependency note and the plan its own staleness. */
  inPlan?: boolean;
  /** What the surface offers when the write would be refused; nothing when it cannot act. */
  wouldFail?: WouldFailActions;
}) {
  const t = useT();
  const [expanded, setExpanded] = useState(false);
  const limited = density === "compact" && !expanded;
  const records = limited ? preview.changes.slice(0, COMPACT_RECORDS) : preview.changes;
  const overflow =
    density === "compact" &&
    (preview.changes.length > COMPACT_RECORDS ||
      preview.changes.some((change) => change.fields.length > COMPACT_FIELDS));
  const warnings = visibleWarnings(preview.warnings, { inPlan });

  return (
    <div className="flex min-w-0 flex-col gap-3" data-slot="proposal-preview">
      {preview.stale && <StaleNotice missing={preview.staleness?.missing === true} />}
      {preview.coverage === "Unavailable" && (
        <Alert size="sm" variant="warning">
          <TriangleAlertIcon />
          <AlertDescription>
            {t(
              "This action can't say exactly what it would change. It would run with the values below.",
            )}
          </AlertDescription>
        </Alert>
      )}
      {warnings.map((warning, index) => (
        <WarningAlert key={`${warning.code}-${index}`} warning={warning} wouldFail={wouldFail} />
      ))}

      {preview.changes.length === 0 ? (
        <p className="text-foreground-muted text-xs">{t("Nothing would change.")}</p>
      ) : (
        <div className={cn("flex flex-col", density === "compact" ? "gap-3" : "gap-4")}>
          {records.map((change, index) => (
            <RecordChangeView
              key={`${change.resource}-${change.entityId ?? index}-${index}`}
              change={change}
              tool={preview.tool}
              fieldLimit={limited ? COMPACT_FIELDS : null}
              density={density}
            />
          ))}
        </div>
      )}

      {(preview.withheldCount > 0 ||
        preview.omittedRecords > 0 ||
        preview.coverage === "Partial" ||
        preview.recorded) && (
        <div className="text-foreground-subtle flex flex-col gap-1 text-xs">
          {preview.withheldCount > 0 && (
            <span className="inline-flex items-center gap-1">
              <LockIcon aria-hidden className="size-3 shrink-0" />
              {t(
                "{0, plural, one {# part hidden by your data access} other {# parts hidden by your data access}}",
                preview.withheldCount,
              )}
            </span>
          )}
          {preview.omittedRecords > 0 && (
            <span>
              {t(
                "{0, plural, one {# more record is past what a preview shows} other {# more records are past what a preview shows}}",
                preview.omittedRecords,
              )}
            </span>
          )}
          {preview.coverage === "Partial" && <span>{t("Shows part of what this does.")}</span>}
          {preview.recorded && <span>{t("As it was shown when it was decided.")}</span>}
        </div>
      )}

      {overflow && (
        <Button
          size="xs"
          variant="ghost"
          className="self-start"
          aria-expanded={expanded}
          onClick={() => setExpanded((current) => !current)}
        >
          {expanded ? t("Show fewer changes") : t("Show all changes")}
        </Button>
      )}
    </div>
  );
}
