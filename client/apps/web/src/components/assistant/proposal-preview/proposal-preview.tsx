import type {
  AgentPreviewOperation,
  PreviewRecordChange,
  PreviewWarning,
  ProposalPreview as ProposalPreviewData,
} from "@/lib/graphql/agent-preview";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  ArchiveIcon,
  Edit02Icon,
  type IconComponent,
  Lock01Icon,
  PlayIcon,
  PlusIcon,
  Send01Icon,
  Trash01Icon,
} from "@trenova/shared/components/icons";
import { useState, type ReactNode } from "react";
import { Link } from "react-router";
import { humanizeToolName } from "../proposal-state";
import { humanizeKey, isRecordId } from "../readable-values";
import { MessagePreview } from "./message-preview";
import { MoneyPreview } from "./money-preview";
import { operationLabel, previewRecordPath } from "./preview-format";
import { PreviewNote, PreviewNotes } from "./preview-notes";
import {
  previewNeedsAttention,
  previewWarningText,
  previewWarningTone,
  reasonSentence,
  visibleWarnings,
  wouldFailReasons,
  type WouldFailReason,
} from "./preview-warnings";
import { ValueChange } from "./value-change";

export type PreviewDensity = "compact" | "full";

/** How much a compact preview shows before "Show all changes". */
const COMPACT_RECORDS = 2;
const COMPACT_FIELDS = 4;

const OPERATION_ICONS: Record<AgentPreviewOperation, IconComponent> = {
  Create: PlusIcon,
  Update: Edit02Icon,
  Delete: Trash01Icon,
  Archive: ArchiveIcon,
  Send: Send01Icon,
  Run: PlayIcon,
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

function WarningNote({
  warning,
  wouldFail,
}: {
  warning: PreviewWarning;
  wouldFail?: WouldFailActions;
}) {
  const t = useT();
  const reasons = wouldFailReasons(warning);
  if (reasons.length > 0) {
    return <WouldFailNote reasons={reasons} actions={wouldFail} />;
  }

  return (
    <PreviewNote tone={previewWarningTone(warning)}>{previewWarningText(warning, t)}</PreviewNote>
  );
}

/**
 * A write its own rules would refuse, reason by reason, each in words a
 * person would use for the field, with a way forward: change the value the
 * rule names, where they may, or have the agent fix it.
 */
function WouldFailNote({
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
    <PreviewNote
      tone="danger"
      title={t("This can't run as it stands")}
      action={
        actions?.onAskAgent && (
          <Button size="xs" variant="secondary" onClick={() => actions.onAskAgent?.(reasons)}>
            {t("Ask the agent to fix it")}
          </Button>
        )
      }
    >
      <ul className="flex w-full flex-col gap-1">
        {reasons.map((reason, index) => (
          <li
            key={`${reason.param}-${index}`}
            className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-0.5"
          >
            <span className="min-w-0 break-words">
              {reason.label !== "" && (
                <span className="text-foreground font-medium">{reason.label}: </span>
              )}
              {reasonSentence(reason, t)}
            </span>
            {changeable(reason) && (
              <Button size="xxs" variant="secondary" onClick={() => actions?.onChange?.(reason)}>
                <Edit02Icon className="size-3" />
                {t("Change {0}", reason.label !== "" ? reason.label : paramLabel(reason.param))}
              </Button>
            )}
          </li>
        ))}
      </ul>
    </PreviewNote>
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
    <PreviewNote tone="warning" title={t("Changed since it was proposed")}>
      {missing
        ? t("The record it would change is gone. It can only be rejected now.")
        : t(
            "The record it would change has been edited since. It can only be rejected now; ask the agent again for a fresh proposal.",
          )}
    </PreviewNote>
  );
}

/** An approval was refused because what it would do moved while the person read it. */
export function ChangedNotice() {
  const t = useT();

  return (
    <PreviewNote tone="warning">
      {t("This change looks different now — review it again.")}
    </PreviewNote>
  );
}

/**
 * The preview could not be read. An internal or personal change may still be
 * approved, without a digest, and the decision is recorded as made without
 * one; the notice says so. A change that must be reviewed cannot be approved
 * until it loads, and the notice says that instead.
 */
export function PreviewFailedNotice({
  onRetry,
  retrying,
  reviewRequired = false,
}: {
  onRetry: () => void;
  retrying: boolean;
  reviewRequired?: boolean;
}) {
  const t = useT();

  return (
    <PreviewNote
      tone="warning"
      action={
        <Button size="xs" variant="secondary" onClick={onRetry} isLoading={retrying}>
          {t("Try again")}
        </Button>
      }
    >
      {reviewRequired
        ? t(
            "What this would change could not be loaded. It has to be reviewed before it can be approved, so try again.",
          )
        : t(
            "What this would change could not be loaded. Approving now is recorded as approved without a preview.",
          )}
    </PreviewNote>
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
  loading,
  reviewRequired = false,
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
  /** What stands in while the first read runs; a skeleton unless the surface shows its own. */
  loading?: ReactNode;
  /** The change cannot be approved without this preview, which a failure says. */
  reviewRequired?: boolean;
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
        <PreviewFailedNotice
          onRetry={() => void query.refetch()}
          retrying={query.isFetching}
          reviewRequired={reviewRequired}
        />
        {fallback}
      </>
    );
  }

  return loading === undefined ? <PreviewSkeleton density={density} /> : loading;
}

/**
 * What a decider has to see before answering, whatever else is folded away:
 * the record moved since the proposal, and every warning the write carries,
 * a refusal reason by reason with its way forward.
 */
export function PreviewAttention({
  preview,
  inPlan = false,
  wouldFail,
}: {
  preview: ProposalPreviewData;
  inPlan?: boolean;
  wouldFail?: WouldFailActions;
}) {
  const t = useT();
  if (!previewNeedsAttention(preview, { inPlan })) {
    return null;
  }
  const warnings = visibleWarnings(preview.warnings, { inPlan });

  if (!preview.stale && warnings.length === 0) {
    return null;
  }

  return (
    <PreviewNotes label={t("Before you decide")}>
      {preview.stale && <StaleNotice missing={preview.staleness?.missing === true} />}
      {warnings.map((warning, index) => (
        <WarningNote key={`${warning.code}-${index}`} warning={warning} wouldFail={wouldFail} />
      ))}
    </PreviewNotes>
  );
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
          <Lock01Icon aria-hidden className="size-3 shrink-0" />
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

/** A value that is a record's id (or a list of them) with no record sent to name it. */
function isBareId(field: PreviewRecordChange["fields"][number]): boolean {
  if (field.withheld || field.beforeRef || field.afterRef) {
    return false;
  }
  const values = [field.before, field.after].filter(
    (value) => value !== null && value !== undefined && value !== "",
  );

  return (
    values.length > 0 &&
    values.every((value) =>
      Array.isArray(value) ? value.length > 0 && value.every(isRecordId) : isRecordId(value),
    )
  );
}

function RecordChangeView({
  change,
  tool,
  fieldLimit,
  density,
  bare,
}: {
  change: PreviewRecordChange;
  tool: string;
  fieldLimit: number | null;
  density: PreviewDensity;
  /** The preview is only the values the action would run with. */
  bare: boolean;
}) {
  const t = useT();
  // A value that is only a record's id names nothing a person can read. The
  // server sends the record itself beside it; one that arrives bare is left out.
  const readable = change.fields.filter((field) => !isBareId(field));
  const fields = fieldLimit === null ? readable : readable.slice(0, fieldLimit);
  const hiddenHere = readable.length - fields.length;
  // The values a run would use are captioned once by the preview; a header
  // repeating the action's name over them says nothing more.
  const headed = !(change.operation === "Run" && bare);

  return (
    <section className="flex min-w-0 flex-col gap-2">
      {headed && <RecordHeader change={change} tool={tool} />}
      {!change.withheld && (
        <>
          {fields.length > 0 && (
            <DescriptionList layout="inline" className={headed ? "pl-5" : undefined}>
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
  attention = true,
  wouldFail,
}: {
  preview: ProposalPreviewData;
  density?: PreviewDensity;
  /** Inside a plan the step draws its own dependency note and the plan its own staleness. */
  inPlan?: boolean;
  /**
   * Whether the preview leads with what needs the decider's attention. A
   * surface that keeps the record changes folded draws `PreviewAttention`
   * itself, where it stays in view, and turns it off here.
   */
  attention?: boolean;
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

  return (
    <div className="flex min-w-0 flex-col gap-3" data-slot="proposal-preview">
      {attention && <PreviewAttention preview={preview} inPlan={inPlan} wouldFail={wouldFail} />}
      {/* A change that cannot be previewed is not an alarm: it is the values
          it would run with, said once as their caption. */}
      {preview.coverage === "Unavailable" && (
        <p className="text-foreground-subtle text-xs">
          {preview.changes.length > 0
            ? t("Its result can't be previewed. It would run with these values.")
            : t("Its result can't be previewed.")}
        </p>
      )}

      {preview.changes.length === 0 ? (
        preview.coverage !== "Unavailable" && (
          <p className="text-foreground-muted text-xs">{t("Nothing would change.")}</p>
        )
      ) : (
        <div className={cn("flex flex-col", density === "compact" ? "gap-3" : "gap-4")}>
          {records.map((change, index) => (
            <RecordChangeView
              key={`${change.resource}-${change.entityId ?? index}-${index}`}
              change={change}
              tool={preview.tool}
              fieldLimit={limited ? COMPACT_FIELDS : null}
              density={density}
              bare={preview.coverage === "Unavailable"}
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
              <Lock01Icon aria-hidden className="size-3 shrink-0" />
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
