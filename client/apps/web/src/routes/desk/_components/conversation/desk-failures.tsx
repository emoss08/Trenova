import type { FailedProvider } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { toast } from "sonner";
import { DeskBrandMark, VENDOR_NAMES } from "../composer/desk-brand-mark";
import { DeskErrorButton, DeskErrorCard } from "../desk-error-card";
import { DeskIcon } from "../desk-icons";

/**
 * Every model the organization set up was asked and none answered. Each one
 * is listed with what it said, and any model the organization has that was
 * never given the task, because giving one the task is the fix. The question
 * is kept, so trying again is one press.
 */
export function DeskNoModelCard({
  providers,
  onRetry,
  onCheckStatus,
}: {
  providers: readonly FailedProvider[];
  onRetry?: () => void;
  onCheckStatus?: () => void;
}) {
  const t = useT();
  return (
    <DeskErrorCard
      tone="err"
      title={t("No model could answer")}
      sub={t("Desk tried every model your organization set up. Your message is saved.")}
    >
      {providers.length > 0 && (
        <div className="dk-ec-provs">
          {providers.map((provider, index) => (
            <div key={`${provider.name}-${index}`}>
              <DeskBrandMark vendor={provider.vendor} size={13} />
              <b>{provider.model || provider.name}</b>
              <span className="dk-ec-pst">{providerStatusLabel(provider.status, t)}</span>
              <em title={provider.detail}>{provider.detail}</em>
            </div>
          ))}
        </div>
      )}
      <div className="dk-ec-acts">
        {onRetry && (
          <DeskErrorButton ink onClick={onRetry}>
            <DeskIcon name="replay" size={12} />
            {t("Try again")}
          </DeskErrorButton>
        )}
        {onCheckStatus && (
          <DeskErrorButton onClick={onCheckStatus}>{t("Check provider status")}</DeskErrorButton>
        )}
      </div>
    </DeskErrorCard>
  );
}

function providerStatusLabel(status: string, t: ReturnType<typeof useT>): string {
  switch (status) {
    case "Overloaded":
      return t("Overloaded");
    case "Timed out":
      return t("Timed out");
    case "Unavailable":
      return t("Unavailable");
    case "Not set up":
      return t("Not set up");
    default:
      return t("Failed");
  }
}

/** How many words a piece of a reply holds, for "dropped after 41 words". */
export function wordCount(text: string): number {
  const words = text.trim().split(/\s+/u);
  return words[0] === "" ? 0 : words.length;
}

/**
 * A reply that stopped before it finished, whether the connection to the
 * model dropped or the person stopped it. What arrived stays above the card;
 * the card says how much arrived and that nothing was changed, and offers to
 * ask again, to carry on from where it stopped, or to copy what is there.
 */
export function DeskCutOffCard({
  arrived,
  vendor,
  stoppedByYou = false,
  changedNothing = true,
  onRetry,
  onContinue,
}: {
  /** The words of the reply that arrived before it stopped. */
  arrived: string;
  /** The company behind the model that was answering, when known. */
  vendor?: string;
  stoppedByYou?: boolean;
  /** False when a step that changes records ran before it stopped. */
  changedNothing?: boolean;
  onRetry?: () => void;
  onContinue?: () => void;
}) {
  const t = useT();
  const words = wordCount(arrived);
  const who = vendor ? (VENDOR_NAMES[vendor] ?? "") : "";
  const what = stoppedByYou
    ? t(
        "{0, plural, one {You stopped it after # word.} other {You stopped it after # words.}}",
        words,
      )
    : who !== ""
      ? t(
          "The connection to {0} dropped after {1}.",
          who,
          t("{0, plural, one {# word} other {# words}}", words),
        )
      : t(
          "{0, plural, one {The connection dropped after # word.} other {The connection dropped after # words.}}",
          words,
        );
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(arrived);
      toast.success(t("Copied what arrived"));
    } catch {
      toast.error(t("Could not copy. Select the text above instead."));
    }
  };

  return (
    <div className="dk-ec-after">
      <DeskErrorCard
        tone="err"
        compact
        title={stoppedByYou ? t("Reply stopped") : t("Reply stopped partway")}
        sub={changedNothing ? `${what} ${t("Nothing was changed.")}` : what}
        actions={
          <>
            {onRetry && (
              <DeskErrorButton ink onClick={onRetry}>
                <DeskIcon name="replay" size={12} />
                {t("Try again")}
              </DeskErrorButton>
            )}
            {onContinue && words > 0 && (
              <DeskErrorButton onClick={onContinue}>{t("Continue from here")}</DeskErrorButton>
            )}
            {words > 0 && (
              <DeskErrorButton onClick={() => void copy()}>
                {t("Copy what arrived")}
              </DeskErrorButton>
            )}
          </>
        }
      />
    </div>
  );
}

/**
 * A reply another model gave because the one asked first did not answer:
 * the first model's mark, faded, then the one that answered, and a line
 * naming both.
 */
export function DeskFallbackLine({
  fromVendor,
  fromModel,
  answeredVendor,
  answeredModel,
}: {
  fromVendor: string;
  fromModel: string;
  answeredVendor: string;
  answeredModel: string;
}) {
  const t = useT();
  return (
    <div className="dk-ec-fb">
      <span className="dk-ec-fbm dk-off">
        <DeskBrandMark vendor={fromVendor} size={11} />
      </span>
      <DeskIcon name="chevR" size={10} />
      <span className="dk-ec-fbm">
        <DeskBrandMark vendor={answeredVendor} size={11} />
      </span>
      <span>
        {t("Answered by")} <b>{answeredModel}</b> · {t("{0} didn't respond", fromModel)}
      </span>
    </div>
  );
}

/**
 * A file on the question that reading made out little of, such as a blurred
 * or skewed photo. The reply says what could be read; the card says the rest
 * could not, and offers a clearer copy or to go on with what there is.
 */
export function DeskPoorlyReadCard({
  fileName,
  onUploadClearer,
  onUseWhatWasRead,
}: {
  fileName: string;
  onUploadClearer?: () => void;
  onUseWhatWasRead?: () => void;
}) {
  const t = useT();
  const photo = /\.(png|jpe?g|heic|tiff?)$/iu.test(fileName);
  return (
    <DeskErrorCard
      tone="warn"
      title={t("I couldn't read most of {0}", fileName)}
      sub={
        photo
          ? t("The photo came through blurred or faint, so only part of it could be made out.")
          : t("The scan came through blurred or faint, so only part of it could be made out.")
      }
      actions={
        <>
          {onUploadClearer && (
            <DeskErrorButton ink onClick={onUploadClearer}>
              {photo ? t("Upload a clearer photo") : t("Upload a clearer copy")}
            </DeskErrorButton>
          )}
          {onUseWhatWasRead && (
            <DeskErrorButton onClick={onUseWhatWasRead}>
              {t("Use what I could read")}
            </DeskErrorButton>
          )}
        </>
      }
    />
  );
}
