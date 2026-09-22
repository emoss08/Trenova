import type { InboundMessageDetail } from "@/lib/graphql/inbox";
import { queries } from "@/lib/queries";
import { agentRunPath } from "@/lib/record-paths";
import { useQuery } from "@tanstack/react-query";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { Link } from "react-router";
import { classificationLabel } from "./classification";

type TrailStep = {
  key: string;
  label: React.ReactNode;
  at?: number | null;
  state: "done" | "current" | "failed";
  assisted?: boolean;
};

function Reviewer({ userId }: { userId: string }) {
  const t = useT();
  const { data } = useQuery({ ...queries.user.detail(userId), retry: false });

  return <>{data?.name ?? t("a teammate")}</>;
}

/**
 * The message's life so far, in the order it happened: when it arrived, what
 * it was read as, what it was matched to, and who closed it — a person, or
 * the desk on its own. The last step is the one it is waiting at.
 */
export function MessageTrail({ message }: { message: InboundMessageDetail }) {
  const t = useT();
  const classification = message.classification ?? null;
  const shipment = message.matchedShipment ?? null;
  const reviewedBy = message.reviewedBy ?? null;

  const steps: TrailStep[] = [
    {
      key: "arrived",
      label:
        message.mailbox === null || message.mailbox === undefined
          ? t("Arrived")
          : t("Arrived at {0}", message.mailbox.address),
      at: message.receivedAt,
      state: "done",
    },
    {
      key: "read",
      label:
        message.failureText !== ""
          ? t("Could not be read")
          : classification === null
            ? t("Not read yet")
            : t("Read as {0}", classificationLabel(t, classification)),
      state: message.failureText !== "" ? "failed" : classification === null ? "current" : "done",
      assisted: classification !== null,
    },
  ];

  if (classification !== null) {
    steps.push({
      key: "matched",
      label:
        shipment !== null
          ? t("Matched to {0}", shipment.proNumber)
          : message.matchedCustomer
            ? t("Matched to {0}", message.matchedCustomer.name)
            : t("Not matched to a record"),
      state: shipment !== null || message.matchedCustomer ? "done" : "current",
      assisted: shipment !== null || Boolean(message.matchedCustomer),
    });
  }

  switch (message.status) {
    case "Actioned":
    case "Ignored": {
      const verb = message.status === "Actioned" ? t("Handled") : t("Ignored");
      steps.push({
        key: "closed",
        label:
          reviewedBy === null ? (
            t("{0} by the desk", verb)
          ) : (
            <>
              {t("{0} by", verb)} <Reviewer userId={reviewedBy} />
            </>
          ),
        at: message.reviewedAt ?? null,
        state: "done",
        assisted: reviewedBy === null,
      });
      break;
    }
    case "InReview":
    case "Quarantined":
      steps.push({ key: "waiting", label: t("Waiting on you"), state: "current" });
      break;
    default:
      steps.push({ key: "reading", label: t("Being read"), state: "current" });
  }

  return (
    <div className="flex flex-col gap-2">
      <ol className="flex flex-col">
        {steps.map((step, index) => (
          <li key={step.key} className="relative flex gap-3 pb-3 last:pb-0">
            {index < steps.length - 1 && (
              <span
                aria-hidden
                className="bg-border absolute top-3 bottom-0 left-[0.3125rem] w-px"
              />
            )}
            <span
              aria-hidden
              className={cn(
                "relative mt-1 size-2.5 shrink-0 rounded-full border-2",
                step.state === "done" && "border-foreground-muted bg-foreground-muted",
                step.state === "current" && "border-warning bg-background",
                step.state === "failed" && "border-danger bg-danger",
              )}
            />
            <div className="flex min-w-0 flex-1 flex-wrap items-baseline justify-between gap-x-3">
              <span className="flex items-center gap-1.5 text-sm">
                {step.assisted && <AssistMark className="text-foreground-subtle size-3.5" />}
                {step.label}
              </span>
              {step.at ? (
                <span className="text-foreground-subtle text-xs tabular-nums">
                  {formatUnixDateTime(step.at)}
                </span>
              ) : null}
            </div>
          </li>
        ))}
      </ol>
      {message.runId && (
        <Link
          to={agentRunPath(message.runId)}
          className="ui-focus-ring text-brand w-fit text-xs underline-offset-4 hover:underline"
        >
          {t("See the run that read it")}
        </Link>
      )}
    </div>
  );
}
