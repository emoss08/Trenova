import { WorkingDot } from "@/components/assistant/voice/working-dot";
import { recordPath } from "@/config/record-links";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Link } from "react-router";
import { CardBar, CardBarHint, CardIcon } from "../../record-card/record-card";
import { canAsk, type ShipmentStage } from "./shipment-stage";

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

/** The bar's line: what is under way, what became of the last ask, or what Create will do. */
function StageNote({ stage, checking }: { stage: ShipmentStage; checking: boolean }) {
  const t = useT();
  if (checking) {
    return (
      <>
        <WorkingDot working />
        {t("The agent is checking the fields")}
      </>
    );
  }
  switch (stage.stage) {
    case "replying":
      return (
        <>
          <WorkingDot working />
          {t("The agent is replying; ask again once it is done")}
        </>
      );
    case "drafting":
      return (
        <>
          <WorkingDot working />
          {t("Drafting the shipment")}
        </>
      );
    case "notDrafted":
      return t(
        "The reply ended without a drafted shipment. Read it in the conversation, or ask again.",
      );
    case "waiting":
      return t("The agent drafted it; nothing is created until you approve");
    case "creating":
      return (
        <>
          <WorkingDot working />
          {t("Approved; creating the shipment")}
        </>
      );
    case "created":
      return (
        <>
          <CardIcon name="check" size={13} stroke={2.4} />
          {t("Created from this reading")}
        </>
      );
    case "attached":
      return (
        <>
          <CardIcon name="check" size={13} stroke={2.4} />
          {t("This file is already a shipment")}
        </>
      );
    case "failed":
      return stage.reason !== ""
        ? t("It was approved but not created: {0}", stage.reason)
        : t("It was approved but not created.");
    case "declined":
      return t("You declined the drafted shipment.");
    case "expired":
      return t("The drafted shipment expired before it was approved.");
    case "simulated":
      return t("The agent was simulating, so the shipment was previewed and not created.");
    case "noFile":
      return t("The file this was read from is no longer on record.");
    default:
      return t("Creating drafts the shipment for your approval");
  }
}

/**
 * What can be done next with what was read, along the card's bar. Check asks the agent to
 * look again at the fields that need a person and to look for the missing ones; Create asks
 * it to draft the shipment, which waits for approval like every other write; once drafted the
 * bar opens the approval, and once created it opens the shipment. Nothing is written from the
 * browser.
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
    stage.stage !== "attached" &&
    stage.stage !== "noFile";
  const settled = stage.stage === "created" || stage.stage === "attached";
  const failed = stage.stage === "failed";

  return (
    <CardBar>
      {checkCount > 0 && offersCreate && (
        <Button
          variant="quiet"
          className="dk-rec-step"
          disabled={!askable}
          title={askable ? checkSummary : undefined}
          aria-label={t("Ask the agent to check {0}", checkCount)}
          aria-description={askable ? checkSummary : undefined}
          onClick={onCheck}
        >
          <CardIcon name="reset" size={13} stroke={2} />
          <span className="dk-ex-long" aria-hidden>
            {t("Ask the agent to check {0}", checkCount)}
          </span>
          <span className="dk-ex-short" aria-hidden>
            {t("Check {0}", checkCount)}
          </span>
        </Button>
      )}
      <CardBarHint>
        <span className={cn("dk-ex-note", settled && "dk-ok", failed && "dk-bad")}>
          <StageNote stage={stage} checking={checking} />
        </span>
      </CardBarHint>
      {stage.stage === "waiting" && onReview && (
        <Button onClick={onReview}>{t("Review the draft")}</Button>
      )}
      {shipmentId !== "" && (
        <Link className="dk-ax-btn dk-ink" to={recordPath("shipment", shipmentId)}>
          <CardIcon name="ext" size={13} />
          {stage.stage === "created" && stage.name !== ""
            ? t("Open {0}", stage.name)
            : t("Open the shipment")}
        </Link>
      )}
      {offersCreate && (
        <Button disabled={!askable} onClick={onCreate}>
          {createLabel}
        </Button>
      )}
    </CardBar>
  );
}
