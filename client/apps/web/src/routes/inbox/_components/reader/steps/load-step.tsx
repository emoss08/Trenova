import { recordPath } from "@/config/record-links";
import { describeApiError } from "@/lib/api-error-message";
import type { InboundMessageDetail, TriageAction, TriageLoadDraft } from "@/lib/graphql/inbox";
import type { TriageLoadFieldKey } from "@trenova/graphql/generated/graphql";
import { ArrowRightIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { TextShimmer } from "@trenova/shared/components/ui/text-shimmer";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { cn } from "@trenova/shared/lib/utils";
import { useReducedMotion } from "motion/react";
import { useEffect, useId, useState } from "react";
import { Link } from "react-router";
import { useTriageActions } from "../../use-triage-actions";
import { useReaderPanel, useStepSubmit } from "../reader-panel";
import { StepBody, StepError, StepFooter } from "./step-parts";

/** The fields of a load started by hand, before Nova or anyone has read anything into it. */
const BLANK_KEYS: readonly TriageLoadFieldKey[] = [
  "Customer",
  "Origin",
  "Destination",
  "Pickup",
  "Delivery",
  "Equipment",
  "Weight",
  "Rate",
];
/** How long each drafted row takes to arrive, so the draft reads as Nova filling it in. */
const ROW_MS = 160;
const SERVER_FIELD = /^fields\[(\d+)\]\.value$/;

function fieldLabel(t: TranslateFn, key: TriageLoadFieldKey): string {
  switch (key) {
    case "Customer":
      return t("Customer");
    case "Origin":
      return t("Origin");
    case "Destination":
      return t("Destination");
    case "Pickup":
      return t("Pickup");
    case "Delivery":
      return t("Delivery");
    case "Equipment":
      return t("Equipment");
    case "Weight":
      return t("Weight");
    case "Rate":
      return t("Rate");
    case "Loads":
      return t("Loads");
    case "WeekOf":
      return t("Week of");
    case "RespondBy":
      return t("Respond by");
    case "Reference":
      return t("Reference");
    case "Commodity":
      return t("Commodity");
    case "Instructions":
      return t("Instructions");
  }
}

function fieldHint(t: TranslateFn, key: TriageLoadFieldKey): string {
  switch (key) {
    case "Origin":
    case "Destination":
      return t("Street address or city, state");
    case "Pickup":
    case "Delivery":
    case "RespondBy":
      return t("Date and time");
    case "Rate":
      return t("Rate con to follow");
    case "Weight":
      return t("Pounds");
    case "WeekOf":
      return t("Date");
    default:
      return "";
  }
}

type Row = { key: TriageLoadFieldKey; value: string };

function rowsOf(draft: TriageLoadDraft | null, message: InboundMessageDetail): Row[] {
  if (draft !== null) {
    return draft.fields.map((field) => ({ key: field.key, value: field.value }));
  }

  return BLANK_KEYS.map((key) => ({
    key,
    value: key === "Customer" ? (message.matchedCustomer?.name ?? "") : "",
  }));
}

/** Server field errors at `fields[i].value`, by the field they belong to. */
function fieldErrors(error: unknown, sent: readonly Row[]): Map<TriageLoadFieldKey, string> {
  const byKey = new Map<TriageLoadFieldKey, string>();
  if (!(error instanceof ApiRequestError || error instanceof GraphQLRequestError)) {
    return byKey;
  }
  for (const fieldError of error.normalize().fieldErrors) {
    const match = SERVER_FIELD.exec(fieldError.field);
    const row = match === null ? undefined : sent[Number(match[1])];
    if (row !== undefined && !byKey.has(row.key)) {
      byKey.set(row.key, fieldError.message);
    }
  }

  return byKey;
}

function LoadForm({
  message,
  action,
  draft,
  arriving,
}: {
  message: InboundMessageDetail;
  action: TriageAction;
  draft: TriageLoadDraft | null;
  /** The draft landed while the step was open, so its rows arrive one by one. */
  arriving: boolean;
}) {
  const t = useT();
  const actions = useTriageActions();
  const formId = useId();
  const reduceMotion = useReducedMotion() ?? false;
  const [rows, setRows] = useState<Row[]>(() => rowsOf(draft, message));
  const [shown, setShown] = useState(arriving && !reduceMotion ? 0 : rows.length);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [errors, setErrors] = useState<Map<TriageLoadFieldKey, string>>(new Map());

  useEffect(() => {
    if (shown >= rows.length) {
      return;
    }
    const timer = setTimeout(() => setShown((count) => count + 1), ROW_MS);
    return () => clearTimeout(timer);
  }, [shown, rows.length]);

  const streaming = shown < rows.length;
  const empty = rows.filter((row) => row.value.trim() === "").length;
  const ready = !streaming && !pending;

  const submit = () => {
    if (!ready) {
      return;
    }
    const sent = rows.map((row) => ({ key: row.key, value: row.value.trim() }));
    setPending(true);
    setError(null);
    setErrors(new Map());
    actions
      .createLoad(message.id, sent)
      .catch((failure: unknown) => {
        const byField = fieldErrors(failure, sent);
        setErrors(byField);
        if (byField.size === 0) {
          setError(describeApiError(failure));
        }
      })
      .finally(() => setPending(false));
  };
  useStepSubmit(action.id, ready ? submit : null);

  return (
    <StepBody>
      <form
        id={formId}
        className="grid grid-cols-2 gap-2"
        onSubmit={(event) => {
          event.preventDefault();
          submit();
        }}
      >
        {rows.slice(0, shown).map((row, index) => {
          const missing = row.value.trim() === "";
          const fieldError = errors.get(row.key);
          const inputId = `${formId}-${row.key}`;
          return (
            <div key={row.key} className="animate-rise flex min-w-0 flex-col gap-1">
              <label htmlFor={inputId} className="text-foreground-subtle text-xs">
                {fieldLabel(t, row.key)}
              </label>
              <Input
                id={inputId}
                value={row.value}
                placeholder={fieldHint(t, row.key)}
                aria-invalid={fieldError !== undefined || undefined}
                aria-describedby={fieldError === undefined ? undefined : `${inputId}-error`}
                className={cn(
                  "h-7.5 text-sm",
                  missing &&
                    fieldError === undefined &&
                    "border-warning-border placeholder:text-warning border-dashed",
                )}
                onChange={(event) => {
                  const value = event.target.value;
                  setRows((current) =>
                    current.map((item, at) => (at === index ? { ...item, value } : item)),
                  );
                }}
              />
              {fieldError !== undefined && (
                <span id={`${inputId}-error`} className="text-danger text-xs">
                  {fieldError}
                </span>
              )}
            </div>
          );
        })}
      </form>
      {error !== null && <StepError>{error}</StepError>}
      {!streaming && (
        <StepFooter
          noteTone={empty > 0 ? "warning" : "subtle"}
          note={
            empty > 0
              ? t("{0, plural, one {# empty — fine for a draft} other {# empty — fine for a draft}}", empty)
              : t("All filled in")
          }
        >
          <Button size="sm" variant="ghost" onClick={() => actions.discardLoad(message.id)}>
            {t("Discard")}
          </Button>
          <Button
            size="sm"
            type="submit"
            form={formId}
            isLoading={pending}
            loadingText={t("Creating…")}
          >
            {t("Create")}
          </Button>
        </StepFooter>
      )}
    </StepBody>
  );
}

/**
 * Building the load a message describes: Nova's draft arriving row by row,
 * the form with what it could not read marked, then Create; once created,
 * what was booked and the way to it.
 */
export function LoadStep({
  message,
  action,
}: {
  message: InboundMessageDetail;
  action: TriageAction;
}) {
  const t = useT();
  const panel = useReaderPanel();
  const drafting = panel.state.drafting.load;
  const draftError = panel.state.errors.load;
  const [watched, setWatched] = useState(drafting);
  if (drafting && !watched) {
    setWatched(true);
  }

  const created = action.createdShipment;
  if (created !== null) {
    return (
      <StepBody className="gap-1.5">
        <dl className="flex flex-col text-sm">
          {[
            [t("Shipment"), created.proNumber],
            [t("Lane"), `${created.originCity}, ${created.originState} → ${created.destinationCity}, ${created.destinationState}`],
            [t("BOL"), created.bol === "" ? "—" : created.bol],
          ].map(([label, value]) => (
            <div key={label} className="flex justify-between gap-2.5 py-1">
              <dt className="text-foreground-subtle">{label}</dt>
              <dd className="truncate text-right">{value}</dd>
            </div>
          ))}
        </dl>
        <Link
          to={recordPath("shipment", created.id)}
          className="text-foreground-subtle hover:text-foreground ui-focus-ring inline-flex w-fit items-center gap-1 rounded-sm text-xs"
        >
          {t("Open {0}", created.proNumber)}
          <ArrowRightIcon aria-hidden className="size-2.75" />
        </Link>
      </StepBody>
    );
  }

  if (drafting && message.loadDraft === null) {
    return (
      <StepBody>
        <TextShimmer as="span" className="text-sm">
          {t("Nova is drafting the load…")}
        </TextShimmer>
        <div className="grid grid-cols-2 gap-2">
          {BLANK_KEYS.slice(0, 4).map((key) => (
            <Skeleton key={key} className="h-11" />
          ))}
        </div>
      </StepBody>
    );
  }

  const form = (
    <LoadForm
      key={message.loadDraft?.draftedAt ?? "blank"}
      message={message}
      action={action}
      draft={message.loadDraft}
      arriving={watched && message.loadDraft !== null}
    />
  );
  if (draftError !== null && message.loadDraft === null) {
    return (
      <>
        <StepBody className="pb-1">
          <StepError>{draftError}</StepError>
          <StepFooter note={t("Fill it in by hand below, or ask again.")}>
            <Button size="sm" variant="ghost" onClick={panel.draftLoad}>
              {t("Try again")}
            </Button>
          </StepFooter>
        </StepBody>
        {form}
      </>
    );
  }

  return form;
}
