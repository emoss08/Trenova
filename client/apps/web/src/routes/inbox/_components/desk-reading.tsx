import { SectionPanel } from "@/components/section-panel";
import type { InboundMessageDetail } from "@/lib/graphql/inbox";
import { customerPath, shipmentPath } from "@/lib/record-paths";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Building2Icon, TruckIcon } from "lucide-react";
import { Link } from "react-router";
import { CLASSIFICATION_ICON, CLASSIFICATION_VARIANT, classificationLabel } from "./classification";
import type { NextStep } from "./next-step";

export type NextStepHandlers = {
  onReviewDraft: (documentId: string) => void;
  onAttach: (documentId: string, shipmentId: string) => void;
  onAsk: () => void;
  onLink: () => void;
  busy: boolean;
  canAsk: boolean;
};

function percent(value: number): number {
  return Math.round(Math.min(1, Math.max(0, value)) * 100);
}

/**
 * How sure the reading was, against the bar this mailbox sets before the desk
 * acts alone. The tick is the bar; a reading short of it is why the message
 * waited for a person, and the words say so rather than leaving the colour to.
 */
function ConfidenceMeter({ confidence, bar }: { confidence: number; bar: number | null }) {
  const t = useT();
  const value = percent(confidence);
  const threshold = bar === null ? null : percent(bar);
  const short = threshold !== null && value < threshold;

  return (
    <div className="flex flex-col gap-1">
      <div className="bg-muted relative h-1.5 w-full overflow-hidden rounded-full">
        <div
          className={cn("h-full rounded-full", short ? "bg-warning" : "bg-foreground-muted")}
          style={{ width: `${value}%` }}
        />
        {threshold !== null && (
          <span
            aria-hidden
            className="bg-foreground absolute inset-y-0 w-px"
            style={{ left: `${threshold}%` }}
          />
        )}
      </div>
      <span
        className={cn("text-xs tabular-nums", short ? "text-warning" : "text-foreground-subtle")}
      >
        {threshold === null
          ? t("{0}% sure", value)
          : short
            ? t(
                "{0}% sure, short of this mailbox's {1}% bar, so it waited for you",
                value,
                threshold,
              )
            : t("{0}% sure, clear of this mailbox's {1}% bar", value, threshold)}
      </span>
    </div>
  );
}

function NextStepAction({ step, handlers }: { step: NextStep; handlers: NextStepHandlers }) {
  const t = useT();

  switch (step.kind) {
    case "none":
      return null;
    case "reviewDraft":
      return (
        <StepRow
          text={t(
            "The tender came with {0}. The desk has drafted a shipment from it.",
            step.fileName,
          )}
          action={
            <Button size="sm" onClick={() => handlers.onReviewDraft(step.documentId)}>
              {t("Review the draft")}
            </Button>
          }
        />
      );
    case "attachToShipment":
      return (
        <StepRow
          text={t(
            "File {0} on {1}, where billing will look for it.",
            step.fileName,
            step.proNumber,
          )}
          action={
            <Button
              size="sm"
              disabled={handlers.busy}
              onClick={() => handlers.onAttach(step.documentId, step.shipmentId)}
            >
              {t("Attach to {0}", step.proNumber)}
            </Button>
          }
        />
      );
    case "openShipment":
      return (
        <StepRow
          text={t("The answer is on {0}.", step.proNumber)}
          action={
            <Button
              size="sm"
              nativeButton={false}
              render={<Link to={shipmentPath(step.shipmentId)} />}
            >
              {t("Open {0}", step.proNumber)}
            </Button>
          }
        />
      );
    case "askDesk":
      return (
        <StepRow
          text={
            step.reason === "buildLoad"
              ? t("Nothing readable came attached. The desk can build the load from the text.")
              : t("The desk can look the load up and draft the answer for you to send.")
          }
          action={
            <Button size="sm" disabled={!handlers.canAsk || handlers.busy} onClick={handlers.onAsk}>
              {step.reason === "buildLoad"
                ? t("Ask the desk to build it")
                : t("Ask the desk to answer")}
            </Button>
          }
        />
      );
    case "linkByHand":
      return (
        <StepRow
          text={t("The desk could not tell what this is about. Say which record it belongs to.")}
          action={
            <Button size="sm" onClick={handlers.onLink}>
              {t("Link by hand")}
            </Button>
          }
        />
      );
  }
}

function StepRow({ text, action }: { text: string; action: React.ReactNode }) {
  const t = useT();

  return (
    <div className="bg-sunken flex flex-wrap items-center justify-between gap-3 rounded-md px-3 py-2.5">
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span className="text-foreground-subtle text-xs font-medium">{t("Suggested next")}</span>
        <span className="text-sm">{text}</span>
      </div>
      {action}
    </div>
  );
}

/**
 * What the desk made of the message, beside the message and never instead of
 * it: the kind, how sure, what it was matched to and why, and the one thing to
 * do next. Every word of it is the machine's, so it wears the assist mark.
 */
export function DeskReading({
  message,
  step,
  handlers,
}: {
  message: InboundMessageDetail;
  step: NextStep;
  handlers: NextStepHandlers;
}) {
  const t = useT();
  const classification = message.classification ?? null;
  const KindIcon = classification === null ? null : CLASSIFICATION_ICON[classification];
  const bar =
    message.mailbox?.reviewPolicy === "ReviewBelowConfidence"
      ? message.mailbox.minConfidence
      : null;
  const shipment = message.matchedShipment ?? null;
  const customer = message.matchedCustomer ?? null;
  const lane =
    shipment === null
      ? ""
      : [
          [shipment.originCity, shipment.originState].filter(Boolean).join(", "),
          [shipment.destinationCity, shipment.destinationState].filter(Boolean).join(", "),
        ]
          .filter(Boolean)
          .join(" → ");

  return (
    <SectionPanel title={t("What the desk made of it")} icon={<AssistMark />}>
      <div className="flex flex-col gap-4 p-3">
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="flex flex-col gap-2">
            <span className="text-foreground-subtle text-xs font-medium">{t("Read as")}</span>
            {classification === null || KindIcon === null ? (
              <span className="text-foreground-muted text-sm">{t("Not read yet")}</span>
            ) : (
              <>
                <Badge variant={CLASSIFICATION_VARIANT[classification]} className="w-fit gap-1">
                  <KindIcon className="size-3" aria-hidden />
                  {classificationLabel(t, classification)}
                </Badge>
                <ConfidenceMeter confidence={message.confidence} bar={bar} />
              </>
            )}
          </div>

          <div className="flex flex-col gap-2">
            <span className="text-foreground-subtle text-xs font-medium">{t("About")}</span>
            {shipment === null && customer === null ? (
              <span className="text-foreground-muted text-sm">{t("Nothing matched")}</span>
            ) : (
              <div className="flex flex-col gap-1.5">
                {shipment !== null && (
                  <Link
                    to={shipmentPath(shipment.id)}
                    className="ui-focus-ring hover:bg-surface-hover border-border flex items-start gap-2 rounded-md border px-2.5 py-2 transition-colors"
                  >
                    <TruckIcon
                      className="text-foreground-subtle mt-0.5 size-4 shrink-0"
                      aria-hidden
                    />
                    <span className="flex min-w-0 flex-col">
                      <span className="text-brand truncate text-sm font-medium">
                        {shipment.proNumber}
                      </span>
                      {lane !== "" && (
                        <span className="text-foreground-subtle truncate text-xs">{lane}</span>
                      )}
                      {(shipment.bol !== "" || shipment.poNumber !== "") && (
                        <span className="text-foreground-subtle truncate text-xs">
                          {[
                            shipment.bol !== "" ? t("BOL {0}", shipment.bol) : "",
                            shipment.poNumber !== "" ? t("PO {0}", shipment.poNumber) : "",
                          ]
                            .filter(Boolean)
                            .join(" · ")}
                        </span>
                      )}
                    </span>
                  </Link>
                )}
                {customer !== null && (
                  <Link
                    to={customerPath(customer.id)}
                    className="ui-focus-ring text-brand flex w-fit items-center gap-1.5 text-sm underline-offset-4 hover:underline"
                  >
                    <Building2Icon className="text-foreground-subtle size-3.5" aria-hidden />
                    {customer.name}
                  </Link>
                )}
              </div>
            )}
          </div>
        </div>

        {message.matchReason !== "" && (
          <p className="text-foreground-muted border-border border-l-2 pl-3 text-sm leading-relaxed">
            {message.matchReason}
          </p>
        )}

        <NextStepAction step={step} handlers={handlers} />
      </div>
    </SectionPanel>
  );
}
