import { describeApiError } from "@/lib/api-error-message";
import type {
  InboundMessageDetail,
  TriageAction,
  TriageExceptionKind,
} from "@/lib/graphql/inbox";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { useId, useMemo, useState } from "react";
import { useTriageActions } from "../../use-triage-actions";
import { readException } from "../exception-reading";
import { useStepSubmit } from "../reader-panel";
import { StepBody, StepError, StepFooter } from "./step-parts";

const KINDS: readonly TriageExceptionKind[] = ["Shortage", "Overage", "Damage"];

function kindLabel(t: TranslateFn, kind: TriageExceptionKind): string {
  switch (kind) {
    case "Shortage":
      return t("Short");
    case "Overage":
      return t("Over");
    case "Damage":
      return t("Damaged");
  }
}

function kindNoun(t: TranslateFn, kind: TriageExceptionKind): string {
  switch (kind) {
    case "Shortage":
      return t("a shortage");
    case "Overage":
      return t("an overage");
    case "Damage":
      return t("a damage");
  }
}

/**
 * Logging what the delivery noted — "2 pallets short" read from the message —
 * as an exception on the shipment, which holds the invoice until it is
 * settled. The count, unit and kind start from what Nova read.
 */
export function ExceptionStep({
  message,
  action,
}: {
  message: InboundMessageDetail;
  action: TriageAction;
}) {
  const t = useT();
  const actions = useTriageActions();
  const formId = useId();
  const read = useMemo(() => {
    const source =
      message.facts.find((fact) => readException(fact.value).kind !== null)?.value ??
      message.attachments
        .flatMap((attachment) => attachment.reading?.fields ?? [])
        .find((field) => readException(field.value).kind !== null)?.value ??
      "";
    return readException(source);
  }, [message.facts, message.attachments]);
  const [kind, setKind] = useState<TriageExceptionKind>(
    action.exceptionKind ?? read.kind ?? "Shortage",
  );
  const [quantity, setQuantity] = useState(read.quantity === null ? "" : String(read.quantity));
  const [unit, setUnit] = useState(read.unit);
  const [note, setNote] = useState("");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const count = Number(quantity);
  const ready = !pending && Number.isInteger(count) && count > 0 && unit.trim() !== "";
  const submit = () => {
    if (!ready) {
      return;
    }
    setPending(true);
    setError(null);
    actions
      .logException(message.id, {
        kind,
        quantity: count,
        unit: unit.trim(),
        note: note.trim() === "" ? null : note.trim(),
      })
      .catch((failure: unknown) => setError(describeApiError(failure)))
      .finally(() => setPending(false));
  };
  useStepSubmit(action.id, action.done || !ready ? null : submit);

  if (action.done) {
    return null;
  }
  const pro = action.shipment?.proNumber ?? message.matchedShipment?.proNumber ?? "";
  const items = KINDS.map((value) => ({ value, label: kindLabel(t, value) }));

  return (
    <StepBody>
      <p className="text-foreground-muted text-sm leading-normal">
        {pro === ""
          ? t("Adds {0} exception to the shipment so billing holds the invoice until it's settled.", kindNoun(t, kind))
          : t(
              "Adds {0} exception to {1} so billing holds the invoice until it's settled.",
              kindNoun(t, kind),
              pro,
            )}
      </p>
      <form
        id={formId}
        className="grid grid-cols-[4rem_minmax(0,1fr)_minmax(0,7rem)] gap-1.5"
        onSubmit={(event) => {
          event.preventDefault();
          submit();
        }}
      >
        <Input
          type="number"
          min={1}
          inputMode="numeric"
          aria-label={t("How many")}
          value={quantity}
          onChange={(event) => setQuantity(event.target.value)}
          className="h-7 text-sm"
        />
        <Input
          aria-label={t("Unit")}
          placeholder={t("pallets, cases, pieces")}
          value={unit}
          onChange={(event) => setUnit(event.target.value)}
          className="h-7 text-sm"
        />
        <Select
          value={kind}
          items={items}
          onValueChange={(next) => next !== null && setKind(next as TriageExceptionKind)}
        >
          <SelectTrigger aria-label={t("Kind of exception")} className="h-7 w-full text-sm">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {items.map((item) => (
              <SelectItem key={item.value} value={item.value}>
                {item.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Input
          aria-label={t("Note")}
          placeholder={t("Note for billing (optional)")}
          value={note}
          onChange={(event) => setNote(event.target.value)}
          className="h-7 text-sm"
          inputContainerClassName="col-span-3"
        />
      </form>
      {error !== null && <StepError>{error}</StepError>}
      <StepFooter>
        <span className="flex-1" />
        <Button
          size="sm"
          type="submit"
          form={formId}
          disabled={!ready}
          isLoading={pending}
          loadingText={t("Logging…")}
        >
          {t("Log exception")}
        </Button>
      </StepFooter>
    </StepBody>
  );
}

/** Blocking the sender: their future mail on this source goes straight to Ignored. */
export function BlockStep({
  message,
  action,
}: {
  message: InboundMessageDetail;
  action: TriageAction;
}) {
  const t = useT();
  const actions = useTriageActions();
  const block = () => actions.blockSender(message.id);
  useStepSubmit(action.id, action.done ? null : block);
  if (action.done) {
    return null;
  }

  return (
    <StepBody>
      <p className="text-foreground-muted text-sm leading-normal">
        {t("Future mail from {0} goes straight to Ignored.", message.senderDomain || message.fromAddress)}
      </p>
      <StepFooter>
        <span className="flex-1" />
        <Button size="sm" onClick={block}>
          {t("Block and ignore")}
        </Button>
      </StepFooter>
    </StepBody>
  );
}
