import { Lock01Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@trenova/shared/components/ui/hover-card";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { CardIcon } from "../../record-card/record-card";
import {
  sourceLabel,
  valueState,
  type DraftCandidate,
  type DraftChange,
  type DraftCredential,
  type DraftRequirement,
  type DraftSource,
  type DraftValue,
} from "./connection-draft";

/** A transaction set as a stamp: its number over the way it travels. */
export function SetStamp({
  transactionSet,
  direction,
}: {
  transactionSet: string;
  direction: string;
}) {
  const t = useT();
  return (
    <span className="dk-ecd-set">
      <b>{transactionSet || "—"}</b>
      <i>{direction === "Inbound" ? t("In") : direction === "Outbound" ? t("Out") : ""}</i>
    </span>
  );
}

/** Where a value came from as a tab in the margin; the guide's own words open from it. */
export function SourceTab({ source }: { source: DraftSource }) {
  const t = useT();
  const label = sourceLabel(source, t);
  const quoted = source.quote !== "" || source.section !== "" || source.note !== "";
  if (!quoted) {
    return <span className={cn("dk-ecd-src", `dk-${source.kind.toLowerCase()}`)}>{label}</span>;
  }
  return (
    <HoverCard>
      <HoverCardTrigger
        delay={200}
        render={
          <button
            type="button"
            className={cn("dk-ecd-src dk-quoted", `dk-${source.kind.toLowerCase()}`)}
            aria-label={t("{0}: what the guide says", label)}
          />
        }
      >
        {label}
      </HoverCardTrigger>
      <HoverCardContent align="end" className="w-80">
        <div className="flex items-baseline justify-between gap-3 text-xs text-muted-foreground">
          <span>{label}</span>
          {source.section !== "" && <span>{t("Section {0}", source.section)}</span>}
        </div>
        {source.quote !== "" && (
          <blockquote className="mt-1.5 border-l border-border pl-2 font-mono text-xs break-words whitespace-pre-wrap">
            {source.quote}
          </blockquote>
        )}
        {source.note !== "" && <p className="mt-1.5 text-xs">{source.note}</p>}
      </HoverCardContent>
    </HoverCard>
  );
}

/** How sure the reading is, as a short meter. */
export function Sureness({ confidence }: { confidence: number }) {
  const t = useT();
  if (confidence <= 0) {
    return <span className="dk-ecd-sure dk-none" aria-hidden />;
  }
  const percent = Math.round(Math.min(1, confidence) * 100);
  return (
    <span className="dk-ecd-sure" title={t("{0}% confident", percent)}>
      <i style={{ width: `${percent}%` }} />
    </span>
  );
}

function changeWord(change: DraftChange, t: TranslateFn): string {
  switch (change) {
    case "Add":
      return t("New");
    case "Change":
      return t("Changes");
    default:
      return "";
  }
}

/** Whether saving adds or changes this, as a word beside it; nothing when it stays. */
export function ChangeMark({ change }: { change: DraftChange }) {
  const t = useT();
  const word = changeWord(change, t);
  return word ? (
    <span className={cn("dk-ecd-chg", `dk-${change.toLowerCase()}`)}>{word}</span>
  ) : null;
}

function statusWord(value: DraftValue, t: TranslateFn): string {
  switch (value.status) {
    case "Held":
      return t("Held for you");
    case "Unverified":
      return t("Not found on its page");
    case "Invalid":
      return t("Will not save");
    case "Missing":
      return value.required ? t("Required, not in the guide") : t("Not in the guide");
    case "Defaulted":
      return t("Default");
    case "Kept":
      return t("Kept");
    case "Accepted":
      return "";
  }
}

/** One setting the draft would write: what it is, its value over what it was, and where it came from. */
function ValueRow({ value }: { value: DraftValue }) {
  const t = useT();
  const state = valueState(value.status);
  const word = statusWord(value, t);
  return (
    <div className={cn("dk-ecd-v", `dk-${state}`)}>
      <span className="dk-ecd-vl">{value.label || value.field}</span>
      <span className="dk-ecd-vv">
        {value.value !== "" ? <b title={value.value}>{value.value}</b> : <em>{t("Not set")}</em>}
        {value.change === "Change" && value.current !== "" && (
          <s title={value.current}>{value.current}</s>
        )}
        {(word !== "" || value.reason !== "") && (
          <span className="dk-ecd-vn">{[word, value.reason].filter(Boolean).join(" · ")}</span>
        )}
      </span>
      <ChangeMark change={value.change} />
      <Sureness confidence={value.confidence} />
      <SourceTab source={value.source} />
    </div>
  );
}

/** A section's settings, each on its own line. */
export function ValueRows({ values }: { values: readonly DraftValue[] }) {
  return (
    <div className="dk-ecd-vs">
      {values.map((value) => (
        <ValueRow key={value.field} value={value} />
      ))}
    </div>
  );
}

/**
 * A value held back for the person: why, and each reading the guide gave with
 * its page. Picking one asks the agent to draft again with it.
 */
export function DecisionAsk({
  decision,
  disabled,
  onChoose,
}: {
  decision: DraftValue;
  disabled: boolean;
  onChoose: (candidate: DraftCandidate) => void;
}) {
  const t = useT();
  return (
    <div className="dk-ecd-ask">
      <div className="dk-ecd-ask-h">
        <CardIcon name="warn" size={12} stroke={2} />
        <b>{decision.label || decision.field}</b>
        {decision.reason !== "" && <span>{decision.reason}</span>}
      </div>
      {decision.candidates.length > 0 ? (
        <div className="dk-ecd-opts">
          {decision.candidates.map((candidate, index) => (
            <div
              // oxlint-disable-next-line react/no-array-index-key -- two readings may share a value and differ only by page
              key={`${candidate.value}-${index}`}
              className="dk-ecd-opt"
            >
              <Button
                variant="quiet"
                size="sm"
                disabled={disabled || candidate.value === ""}
                onClick={() => onChoose(candidate)}
              >
                {t("Use {0}", candidate.value || t("nothing"))}
              </Button>
              {candidate.problem !== "" && <span className="dk-ecd-vn">{candidate.problem}</span>}
              <Sureness confidence={candidate.confidence} />
              <SourceTab source={candidate.source} />
            </div>
          ))}
        </div>
      ) : (
        <span className="dk-ecd-vn">
          {t("The guide gives no value. Tell the agent what it is.")}
        </span>
      )}
    </div>
  );
}

/** The credentials the person enters after saving, none of them taken from the guide. */
export function CredentialList({ credentials }: { credentials: readonly DraftCredential[] }) {
  const t = useT();
  return (
    <ul className="dk-ecd-keys">
      {credentials.map((credential) => (
        <li key={credential.label} className={cn(credential.required && "dk-req")}>
          {credential.missing ? (
            <Lock01Icon className="size-3 shrink-0" aria-hidden />
          ) : (
            <CardIcon name="check" size={12} stroke={2.4} />
          )}
          <span>{credential.label}</span>
          <em>
            {!credential.missing
              ? t("Already set")
              : credential.required
                ? t("Required before it can be turned on")
                : t("If the partner uses it")}
          </em>
        </li>
      ))}
    </ul>
  );
}

function requirementName(requirement: DraftRequirement, t: TranslateFn): string {
  return requirement.elementPosition > 0
    ? t("{0}, element {1}", requirement.segmentId, requirement.elementPosition)
    : requirement.segmentId;
}

/** What the guide asks of a document type that the drafted map does not write. */
export function RequirementList({ requirements }: { requirements: readonly DraftRequirement[] }) {
  const t = useT();
  return (
    <ul className="dk-ecd-reqs">
      {requirements.map((requirement, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- one segment may carry several rules
        <li key={`${requirement.segmentId}-${requirement.elementPosition}-${index}`}>
          <code>{requirementName(requirement, t)}</code>
          <span>
            {requirement.problem}
            {requirement.codes.length > 0 && (
              <em>{t("Codes {0}", requirement.codes.join(", "))}</em>
            )}
          </span>
          <SourceTab source={requirement.source} />
        </li>
      ))}
    </ul>
  );
}
