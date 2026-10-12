import { useT } from "@trenova/shared/i18n/use-t";
import { formatShortcut } from "@trenova/shared/lib/shortcuts";
import { TextareaField } from "@/components/fields/textarea-field";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useEffect, useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { Ic, type IcName } from "../../kit/ic";
import type { BuilderStart } from "./builder-model";
import { Button } from "@trenova/shared/components/ui/button";

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
  const form = useForm<{ description: string }>({ defaultValues: { description: "" } });
  const description = useWatch({ control: form.control, name: "description" });
  const [drafting, setDrafting] = useState(false);
  const [step, setStep] = useState(0);

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
    {
      start: "chat",
      icon: "chat",
      label: t("Desk agent"),
      note: t("Answers people in the assistant"),
    },
    {
      start: "scheduled",
      icon: "calendar",
      label: t("Scheduled report"),
      note: t("Runs on a timetable and sends a summary"),
    },
    {
      start: "event",
      icon: "bolt",
      label: t("Event watcher"),
      note: t("Wakes when something happens"),
    },
    {
      start: "blank",
      icon: "edit",
      label: t("Set it up yourself"),
      note: t("An empty agent, no description needed"),
    },
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
    form.clearErrors("description");
    setStep(0);
    setDrafting(true);
    onDraft(text).catch((error: unknown) => {
      setDrafting(false);
      form.setError("description", {
        message: error instanceof Error ? error.message : t("Nova could not draft it. Try again."),
      });
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
            <div className="mt-7">
              <TextareaField
                control={form.control}
                name="description"
                label=""
                aria-label={t("What should it do?")}
                size="lg"
                autoFocus
                placeholder={t("When a truck waits more than two hours at a dock…")}
                onKeyDown={(event) => {
                  if ((event.metaKey || event.ctrlKey) && event.key === "Enter") {
                    event.preventDefault();
                    draft();
                  }
                }}
                footer={
                  <>
                    <ScrollArea
                      className="min-w-0 max-w-sm flex-1"
                      maskVariant="field"
                      maskHeight={24}
                      dragToScroll
                    >
                      <div className="flex w-max gap-1">
                        {examples.map((example) => (
                          <Tooltip key={example}>
                            <TooltipTrigger
                              render={
                                <Button
                                  type="button"
                                  variant="secondary"
                                  size="xxs"
                                  className="text-muted-foreground hover:text-foreground h-6.5 max-w-48 shrink-0 rounded-full px-2.5 text-xs font-normal"
                                  onClick={() =>
                                    form.setValue("description", example, { shouldDirty: true })
                                  }
                                />
                              }
                            >
                              <span className="truncate">{example}</span>
                            </TooltipTrigger>
                            <TooltipContent className="max-w-xs">{example}</TooltipContent>
                          </Tooltip>
                        ))}
                      </div>
                    </ScrollArea>
                    {draftingAvailable ? (
                      <Button
                        type="button"
                        variant="default"
                        className="ml-auto shrink-0"
                        shortcut={formatShortcut("↵")}
                        disabled={!description.trim()}
                        onClick={draft}
                      >
                        <AssistMark className="size-3.5" />
                        {t("Draft it")}
                      </Button>
                    ) : (
                      <span className="text-warning-foreground ml-auto inline-flex shrink-0 items-center gap-1.5 text-sm whitespace-nowrap">
                        <Ic n="plug" s={12} />
                        {t("Connect a provider to draft with AI")}
                      </span>
                    )}
                  </>
                }
              />
            </div>
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
                  {index < step ? <Ic n="check" s={12} w={2.4} /> : <span className="border-border-strong border-t-foreground inline-block size-3 animate-spin rounded-full border-[1.5px]" />}
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
