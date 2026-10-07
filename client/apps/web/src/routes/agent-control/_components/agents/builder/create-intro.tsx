import { useT } from "@trenova/shared/i18n/use-t";
import { formatShortcut } from "@trenova/shared/lib/shortcuts";
import { useEffect, useState } from "react";
import { Ic, type IcName } from "../../kit/ic";
import type { BuilderStart } from "./builder-model";

/** How long each drafting step shows before the next begins. */
const STEP_MS = 420;

type CreateIntroProps = {
  /** Nova can draft an agent: a provider takes the assistant's work. */
  draftingAvailable: boolean;
  /** Drafts from the description; rejects with a message to show. */
  onDraft: (description: string) => Promise<void>;
  onStart: (start: BuilderStart, description: string) => void;
};

/**
 * Where a new agent begins: say what it should do and Nova drafts it, or start from one
 * of four shapes and set it up by hand.
 */
export function CreateIntro({ draftingAvailable, onDraft, onStart }: CreateIntroProps) {
  const t = useT();
  const [description, setDescription] = useState("");
  const [drafting, setDrafting] = useState(false);
  const [step, setStep] = useState(0);
  const [failure, setFailure] = useState<string | null>(null);

  const steps = [
    t("Naming it"),
    t("Writing instructions"),
    t("Choosing when it runs"),
    t("Picking tools and how free each one is"),
    t("Setting guardrails"),
  ];
  const examples = [
    t("Chase detention when a truck waits more than two hours at a dock"),
    t("Every Friday, email each customer a list of their open invoices"),
    t("Warn drivers 30 days before their medical card expires"),
  ];
  const starts: { start: BuilderStart; icon: IcName; label: string; note: string }[] = [
    { start: "chat", icon: "chat", label: t("Desk agent"), note: t("Answers people in the assistant") },
    {
      start: "scheduled",
      icon: "calendar",
      label: t("Scheduled report"),
      note: t("Runs on a timetable and sends a summary"),
    },
    { start: "event", icon: "bolt", label: t("Event watcher"), note: t("Wakes when something happens") },
    { start: "blank", icon: "edit", label: t("Start blank"), note: t("Set everything up yourself") },
  ];

  useEffect(() => {
    if (!drafting) return;
    const timer = window.setInterval(
      () => setStep((current) => Math.min(current + 1, steps.length - 1)),
      STEP_MS,
    );
    return () => window.clearInterval(timer);
  }, [drafting, steps.length]);

  const draft = () => {
    const text = description.trim();
    if (!text || !draftingAvailable || drafting) return;
    setFailure(null);
    setStep(0);
    setDrafting(true);
    onDraft(text).catch((error: unknown) => {
      setDrafting(false);
      setFailure(error instanceof Error ? error.message : t("Nova could not draft it. Try again."));
    });
  };

  return (
    <div className="ci">
      <div className="ci-in">
        <span className="ci-k">
          <span className="ci-o" />
          {t("New agent")}
        </span>
        <h1>{t("What should it do?")}</h1>
        <p className="ci-s">
          {t("Describe the job in a sentence or two.")}{" "}
          {draftingAvailable
            ? t("Nova drafts the name, instructions, trigger and tools, and you adjust from there.")
            : t("Then pick a starting point below.")}
        </p>
        {!drafting ? (
          <>
            <div className={draftingAvailable ? "ci-box" : "ci-box off"}>
              <textarea
                autoFocus
                rows={3}
                value={description}
                aria-label={t("What should it do?")}
                placeholder={t("When a truck waits more than two hours at a dock…")}
                onChange={(event) => setDescription(event.target.value)}
                onKeyDown={(event) => {
                  if ((event.metaKey || event.ctrlKey) && event.key === "Enter") {
                    event.preventDefault();
                    draft();
                  }
                }}
              />
              <div className="ci-bar">
                <div className="ci-ex">
                  {examples.map((example) => (
                    <button key={example} type="button" onClick={() => setDescription(example)}>
                      {`${example.split(" ").slice(0, 5).join(" ")}…`}
                    </button>
                  ))}
                </div>
                {draftingAvailable ? (
                  <button
                    type="button"
                    className="btn ink"
                    disabled={!description.trim()}
                    onClick={draft}
                  >
                    <Ic n="sparkle" s={13} />
                    {t("Draft it")}
                    <span className="kbd">{formatShortcut("↵")}</span>
                  </button>
                ) : (
                  <span className="ci-na">
                    <Ic n="plug" s={12} />
                    {t("Connect a provider to draft with AI")}
                  </span>
                )}
              </div>
            </div>
            {failure && <p className="f-h t-w">{failure}</p>}
            <div className="ci-or">
              <span>{draftingAvailable ? t("or start from") : t("Start from")}</span>
            </div>
            <div className="ci-g">
              {starts.map((entry) => (
                <button
                  key={entry.start}
                  type="button"
                  className="ci-t"
                  onClick={() => onStart(entry.start, description.trim())}
                >
                  <span className="ci-ti">
                    <Ic n={entry.icon} s={16} />
                  </span>
                  <b>{entry.label}</b>
                  <span>{entry.note}</span>
                  <Ic n="arrowR" s={13} />
                </button>
              ))}
            </div>
          </>
        ) : (
          <div className="ci-run" aria-live="polite">
            <p className="ci-q">“{description.trim()}”</p>
            {steps.slice(0, step + 1).map((label, index) => (
              <div key={label} className={index === step ? "ci-st cur" : "ci-st"}>
                <span className="ck">
                  {index < step ? <Ic n="check" s={12} w={2.4} /> : <i className="spn" />}
                </span>
                <span className="tx">{label}</span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
