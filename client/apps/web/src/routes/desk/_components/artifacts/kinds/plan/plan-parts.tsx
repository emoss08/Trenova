import { MessagePreview } from "@/components/assistant/proposal-preview/message-preview";
import { MoneyPreview } from "@/components/assistant/proposal-preview/money-preview";
import {
  operationLabel,
  previewRecordPath,
} from "@/components/assistant/proposal-preview/preview-format";
import {
  previewWarningText,
  previewWarningTone,
} from "@/components/assistant/proposal-preview/preview-warnings";
import { HiddenValue, ValueChange } from "@/components/assistant/proposal-preview/value-change";
import type { PreviewRecordChange } from "@/lib/graphql/agent-preview";
import {
  AlertTriangleIcon,
  ChevronRightIcon,
  Edit02Icon,
  File02Icon,
  Link01Icon,
  Mail01Icon,
  PlusIcon,
  Trash01Icon,
  type IconComponent,
} from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useId, useState, type ReactNode } from "react";
import { Link } from "react-router";
import { CardIcon } from "../../record-card/record-card";
import { cardMomentAfter, cardTime } from "../../record-card/record-card-values";
import {
  foldedLine,
  recordLine,
  stepWarnings,
  type PlanRecordView,
  type PlanStepTone,
  type PlanStepView,
} from "./plan-card-model";

/** The plan's steps as one segmented bar under its name: a segment a step, coloured by how it stands. */
export function PlanTrack({ tones }: { tones: readonly PlanStepTone[] }) {
  return (
    <div className="dk-pln-track" aria-hidden>
      {tones.map((tone, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- one segment a step, in step order
        <i key={index} className={`dk-${tone}`} />
      ))}
    </div>
  );
}

const OPERATION_ICONS: Record<PreviewRecordChange["operation"], IconComponent> = {
  Create: PlusIcon,
  Update: Edit02Icon,
  Delete: Trash01Icon,
  Archive: File02Icon,
  Send: Mail01Icon,
  Run: File02Icon,
};

/** The record a change is about, linked where it has a page, and what is done to it. */
function ChangeRecord({ change }: { change: PreviewRecordChange }) {
  const t = useT();
  const Icon = OPERATION_ICONS[change.operation];
  const path = previewRecordPath(change.record);
  const label = change.withheld ? null : change.label.trim();

  return (
    <div className="dk-pln-rec">
      <Icon aria-hidden className="size-3.5" />
      {label === null ? (
        <HiddenValue />
      ) : path !== null && label !== "" ? (
        <Link to={path} className="dk-rec-lnk">
          <b>{label}</b>
        </Link>
      ) : (
        <b>{label === "" ? t("A record") : label}</b>
      )}
      <em>{operationLabel(change.operation, t)}</em>
    </div>
  );
}

/** One record a step changes: its fields from → to, the money it moves, the message it sends. */
function StepChange({ change }: { change: PreviewRecordChange }) {
  const t = useT();
  const fields = change.withheld ? [] : change.fields;

  return (
    <div className="dk-pln-chg">
      <ChangeRecord change={change} />
      {fields.length > 0 && (
        <dl className="dk-pln-f">
          {fields.map((field) => (
            <div key={field.path} className="contents">
              <dt>{field.label}</dt>
              <dd>
                <ValueChange change={field} operation={change.operation} />
              </dd>
            </div>
          ))}
        </dl>
      )}
      {change.omittedFields > 0 && (
        <div className="dk-pln-more">
          {t("{0, plural, one {# more field} other {# more fields}}", change.omittedFields)}
        </div>
      )}
      {!change.withheld && change.money && (
        <div className="dk-pln-x">
          <MoneyPreview money={change.money} />
        </div>
      )}
      {!change.withheld && change.message && (
        <div className="dk-pln-x">
          <MessagePreview message={change.message} />
        </div>
      )}
    </div>
  );
}

/** What a step will change, or changed: every record it touches, open or folded to one line. */
function StepChanges({ step, folded }: { step: PlanStepView; folded: boolean }) {
  const t = useT();
  const [open, setOpen] = useState(!folded);
  const regionId = useId();
  const preview = step.preview;
  if (preview === null) {
    return null;
  }
  const changes = preview.changes;
  if (changes.length === 0) {
    return null;
  }

  if (!folded) {
    return (
      <>
        {changes.map((change, index) => (
          // oxlint-disable-next-line react/no-array-index-key -- a preview's changes keep their order
          <StepChange key={index} change={change} />
        ))}
      </>
    );
  }

  return (
    <>
      <button
        type="button"
        className="dk-pln-fold"
        aria-expanded={open}
        aria-controls={regionId}
        onClick={() => setOpen((current) => !current)}
      >
        <ChevronRightIcon aria-hidden className="size-3.5" />
        <span>{foldedLine(preview, t)}</span>
        <span className="sr-only">{open ? t("Hide the changes") : t("Show the changes")}</span>
      </button>
      {open && (
        <div id={regionId} className="contents">
          {changes.map((change, index) => (
            // oxlint-disable-next-line react/no-array-index-key -- a preview's changes keep their order
            <StepChange key={index} change={change} />
          ))}
        </div>
      )}
    </>
  );
}

function stepTime(step: PlanStepView, t: ReturnType<typeof useT>): string {
  switch (step.tone) {
    case "done":
      return step.ranAt ? t("Ran {0}", cardTime(step.ranAt)) : t("Ran");
    case "fail":
      return step.ranAt ? t("Failed {0}", cardTime(step.ranAt)) : t("Failed");
    case "run":
      return t("Running…");
    case "next":
      return t("Up next");
    case "skip":
      return t("Not run");
    case "wait":
      return t("Waiting");
  }
}

/** One step on the track: its node, what it is for, when it ran, and what it changes. */
export function PlanStep({
  step,
  folded,
  dependencyLabel,
}: {
  step: PlanStepView;
  folded: boolean;
  /** The sentence for the earlier steps this one starts from. */
  dependencyLabel: (steps: readonly number[]) => string;
}) {
  const t = useT();
  const warnings = stepWarnings(step);

  return (
    <li className={cn("dk-pln-step", `dk-${step.tone}`)}>
      <span className="dk-pln-n">
        {step.tone === "done" ? (
          <CardIcon name="check" size={11} stroke={2.8} />
        ) : step.tone === "fail" ? (
          <CardIcon name="x" size={10} stroke={2.8} />
        ) : (
          step.number
        )}
      </span>
      <div className="dk-pln-body">
        <div className="dk-pln-h">
          <div>
            <b>{step.title}</b>
            <span>
              <code>{step.toolName}</code>
            </span>
          </div>
          <span className="dk-pln-at">{stepTime(step, t)}</span>
        </div>
        {step.dependsOn.length > 0 && (
          <span className="dk-pln-dep">
            <Link01Icon aria-hidden className="size-3" />
            {dependencyLabel(step.dependsOn)}
          </span>
        )}
        {step.tone !== "done" &&
          step.tone !== "skip" &&
          warnings.map((warning, index) => (
            <div
              // oxlint-disable-next-line react/no-array-index-key -- a preview's warnings keep their order
              key={index}
              className={cn("dk-pln-warn", previewWarningTone(warning) === "danger" && "dk-bad")}
            >
              <AlertTriangleIcon aria-hidden className="size-3" />
              <span>{previewWarningText(warning, t)}</span>
            </div>
          ))}
        <StepChanges step={step} folded={folded} />
        {step.tone === "fail" && (
          <div className="dk-pln-err" role="alert">
            <b>{t("This step did not go through")}</b>
            {step.error !== "" && <span>{step.error}</span>}
          </div>
        )}
        {step.partial !== null && step.tone !== "fail" && (
          <div className="dk-pln-warn">
            <AlertTriangleIcon aria-hidden className="size-3" />
            <span>{t("{0} of {1} went through", step.partial.done, step.partial.total)}</span>
          </div>
        )}
        {step.tone === "done" && step.result !== "" && (
          <span className="dk-pln-res">
            <CardIcon name="check" size={11} stroke={2.6} />
            {step.result}
          </span>
        )}
      </div>
    </li>
  );
}

/** Every record the plan touches, once each, with the steps that change it. */
export function PlanRecords({
  records,
  withheld,
}: {
  records: readonly PlanRecordView[];
  withheld: number;
}) {
  const t = useT();

  return (
    <div className="dk-pln-recs">
      {records.map((record) => {
        const path = previewRecordPath(record.record);
        const body = (
          <>
            {record.sends > 0 && record.fields.length === 0 ? (
              <Mail01Icon aria-hidden className="size-3.5" />
            ) : (
              <Edit02Icon aria-hidden className="size-3.5" />
            )}
            <div>
              <b>{record.label === "" ? t("A record") : record.label}</b>
              <span>{recordLine(record, t)}</span>
            </div>
            <span
              className="dk-pln-sn"
              aria-label={
                record.steps.length === 1
                  ? t("Step {0}", record.steps[0])
                  : t("Steps {0}", record.steps.join(", "))
              }
            >
              {record.steps.map((step) => (
                <i key={step} aria-hidden>
                  {step}
                </i>
              ))}
            </span>
          </>
        );
        return path !== null ? (
          <Link key={record.key} to={path} className="dk-rec-row">
            {body}
          </Link>
        ) : (
          <div key={record.key} className="dk-rec-row">
            {body}
          </div>
        );
      })}
      {withheld > 0 && (
        <div className="dk-pln-hidden">
          {t(
            "{0, plural, one {# record you cannot see} other {# records you cannot see}}",
            withheld,
          )}
        </div>
      )}
    </div>
  );
}

/** A moment in the plan's history, as its line in the approval log says it. */
export function logMoment(first: number, at: number | null): string {
  return at === null || at === 0 ? "" : cardMomentAfter(first, at);
}

/** A note the person wrote, drawn as their plain words. */
export function PlanNote({ children }: { children: ReactNode }) {
  return <div className="dk-pln-note">{children}</div>;
}
