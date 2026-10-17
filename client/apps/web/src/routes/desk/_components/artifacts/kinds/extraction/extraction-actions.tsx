import { WorkingDot } from "@/components/assistant/voice/working-dot";
import { recordPath } from "@/config/record-links";
import { CheckIcon, LinkExternal01Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { buttonVariants } from "@trenova/shared/lib/variants/button";
import { Link } from "react-router";
import { canAsk, stageStep, type ShipmentStage } from "./shipment-stage";

/**
 * Where the shipment stands, as one line of four steps under the file's name: not created,
 * drafting, waiting on the person, created, with the shipment linked once there is one.
 */
export function ShipmentSteps({ stage }: { stage: ShipmentStage }) {
  const t = useT();
  const at = stageStep(stage);
  const steps = [t("Not created"), t("Drafting"), t("Waiting on you"), t("Created")];
  const shipmentId =
    stage.stage === "created" || stage.stage === "attached" ? stage.shipmentId : "";

  return (
    <div className="dk-ex-state">
      <span className="dk-ex-state-l">{t("Shipment")}</span>
      <ol className="dk-ex-steps" aria-label={t("Where the shipment stands")}>
        {steps.map((label, index) => (
          <li
            key={label}
            className={cn(
              "dk-ex-step",
              index < at && "dk-done",
              index === at && "dk-now",
              index === at && stage.stage === "drafting" && "dk-wait",
            )}
            aria-current={index === at ? "step" : undefined}
          >
            <i aria-hidden />
            {label}
          </li>
        ))}
      </ol>
      {shipmentId !== "" && (
        <Link className="dk-ex-link ui-focus-ring" to={recordPath("shipment", shipmentId)}>
          {stage.stage === "created" && stage.name !== "" ? stage.name : t("Open the shipment")}
        </Link>
      )}
    </div>
  );
}

type ActionsProps = {
  stage: ShipmentStage;
  checking: boolean;
  /** How many fields Check names; Check is not offered when there are none. */
  checkCount: number;
  /** The sentence naming what Check asks about. */
  checkSummary: string;
  onCheck: () => void;
  onCreate: () => void;
  /** Opens the approval for the drafted shipment, when the surface can. */
  onReview: (() => void) | null;
};

/**
 * What can be done next with what was read. Check asks the agent to look again at the fields
 * that need a person and to look for the missing ones; Create asks it to draft the shipment,
 * which waits for approval like every other write. Neither writes anything from the browser.
 */
export function ExtractionActions({
  stage,
  checking,
  checkCount,
  checkSummary,
  onCheck,
  onCreate,
  onReview,
}: ActionsProps) {
  const t = useT();
  const busy = checking || stage.stage === "replying" || stage.stage === "drafting";
  const askable = !busy && canAsk(stage);

  const note = (() => {
    if (checking) {
      return (
        <span className="dk-ex-note">
          <WorkingDot working />
          {t("The agent is checking the fields")}
        </span>
      );
    }
    switch (stage.stage) {
      case "replying":
        return (
          <span className="dk-ex-note">
            <WorkingDot working />
            {t("The agent is replying; ask again once it is done")}
          </span>
        );
      case "drafting":
        return (
          <span className="dk-ex-note">
            <WorkingDot working />
            {t("Drafting the shipment")}
          </span>
        );
      case "notDrafted":
        return (
          <span className="dk-ex-note">
            {t(
              "The reply ended without a drafted shipment. Read it in the conversation, or ask again.",
            )}
          </span>
        );
      case "waiting":
        return <span className="dk-ex-note">{t("Drafted and waiting on your approval")}</span>;
      case "creating":
        return (
          <span className="dk-ex-note">
            <WorkingDot working />
            {t("Approved; creating the shipment")}
          </span>
        );
      case "created":
        return (
          <span className="dk-ex-note dk-ok">
            <CheckIcon className="size-3.5" />
            {stage.name !== "" ? t("Created shipment {0}", stage.name) : t("Created the shipment")}
          </span>
        );
      case "attached":
        return (
          <span className="dk-ex-note dk-ok">
            <CheckIcon className="size-3.5" />
            {t("This file is already a shipment")}
          </span>
        );
      case "failed":
        return (
          <span className="dk-ex-note dk-bad">
            {stage.reason !== ""
              ? t("It was approved but not created: {0}", stage.reason)
              : t("It was approved but not created.")}
          </span>
        );
      case "declined":
        return <span className="dk-ex-note">{t("You declined the drafted shipment.")}</span>;
      case "expired":
        return (
          <span className="dk-ex-note">
            {t("The drafted shipment expired before it was approved.")}
          </span>
        );
      case "simulated":
        return (
          <span className="dk-ex-note">
            {t("The agent was simulating, so the shipment was previewed and not created.")}
          </span>
        );
      case "noFile":
        return (
          <span className="dk-ex-note">
            {t("The file this was read from is no longer on record.")}
          </span>
        );
      default:
        return null;
    }
  })();

  const shipmentId =
    stage.stage === "created" || stage.stage === "attached" ? stage.shipmentId : "";
  const createLabel =
    stage.stage === "declined" || stage.stage === "expired" || stage.stage === "notDrafted"
      ? t("Draft it again")
      : stage.stage === "failed"
        ? t("Ask again")
        : t("Create shipment");
  const offersCreate =
    stage.stage !== "waiting" &&
    stage.stage !== "creating" &&
    stage.stage !== "created" &&
    stage.stage !== "attached";

  return (
    <div className="dk-ex-acts">
      {askable && checkCount > 0 && <p className="dk-ex-ask">{checkSummary}</p>}
      <div className="dk-ex-btns">
        {note}
        <span className="flex-1" />
        {checkCount > 0 && offersCreate && (
          <Button variant="outline" disabled={!askable} onClick={onCheck}>
            {t("{0, plural, one {Check # field} other {Check # fields}}", checkCount)}
          </Button>
        )}
        {stage.stage === "waiting" && onReview && (
          <Button onClick={onReview}>{t("Review the draft")}</Button>
        )}
        {shipmentId !== "" && (
          <Link
            className={buttonVariants({ variant: "outline" })}
            to={recordPath("shipment", shipmentId)}
          >
            {t("Open the shipment")}
            <LinkExternal01Icon className="size-3.5" />
          </Link>
        )}
        {offersCreate && (
          <Button disabled={!askable} onClick={onCreate}>
            {createLabel}
          </Button>
        )}
      </div>
    </div>
  );
}
