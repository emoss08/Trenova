import type { TriageRule, TriageRuleInput } from "@/lib/graphql/inbox";
import { queries } from "@/lib/queries";
import { useTriageStore } from "@/stores/triage-store";
import { useQuery } from "@tanstack/react-query";
import { EyeIcon, PlusIcon, XCloseIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Switch } from "@trenova/shared/components/ui/switch";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatSecondsAgo } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { memo, useEffect, useState, type ReactNode } from "react";
import { useCountUp } from "../briefing/use-briefing-motion";
import { inlineErrorText } from "../inline-error";
import { useTriageActions } from "../use-triage-actions";
import { useTriageRules } from "../use-triage-data";
import {
  EMPTY_DRAFT,
  draftFromRule,
  draftFromSeed,
  missingBlanks,
  ruleInput,
  sameRule,
  type RuleDraft,
} from "./rule-model";
import { RuleSentence } from "./rule-sentence";

/** How long a deleted rule takes to leave before the list lets it go. */
const LEAVE_MS = 280;
const ROW = "grid grid-cols-[52px_minmax(0,1fr)_auto] gap-x-3.5 gap-y-1 @max-2xl:grid-cols-[44px_minmax(0,1fr)]";

function openMatches(target: { ruleId: string } | { input: TriageRuleInput }) {
  useTriageStore.getState().openOverlay({ kind: "ruleMatches", target });
}

/**
 * Every rule as the sentence it is: "When a proof of delivery arrives from
 * drivers.trenova.example on Intake, Nova should file it on the shipment."
 * Each bold blank is a choice in place, a switch pauses the rule, and a
 * quiet line says how often it acted. The add row is the same sentence with
 * its blanks empty; once filled it says what it would have matched over the
 * last thirty days before it is turned on.
 *
 * Its contract with the page: it takes no props, reads its data from
 * `useTriageRules()` and changes rules only through `useTriageActions()`.
 */
export function RulesView() {
  const t = useT();
  const rules = useTriageRules().data;
  const seed = useTriageStore((store) => store.ruleSeed);
  const [draft, setDraft] = useState<RuleDraft | null>(null);

  useEffect(() => {
    if (seed === null) {
      return;
    }
    setDraft(draftFromSeed(seed));
    useTriageStore.getState().seedRule(null);
  }, [seed]);

  const enabled = rules?.filter((rule) => rule.enabled) ?? [];
  const handled = enabled.reduce((sum, rule) => sum + rule.matchesThisMonth, 0);

  return (
    <div className="scrollbar-overlay @container min-h-0 flex-1 overflow-y-auto">
      <div className="animate-rise mx-auto flex w-full max-w-205 flex-col px-9 pt-9 pb-20 @max-2xl:px-4 @max-2xl:pt-6">
        <div className="flex items-baseline gap-3 pb-4">
          <h2 className="text-lg font-semibold">{t("Rules")}</h2>
          {rules !== undefined && (
            <span className="text-foreground-subtle font-mono text-xs tabular-nums">
              {t(
                "{0} on · {1, plural, one {# handled this month} other {# handled this month}}",
                enabled.length,
                handled,
              )}
            </span>
          )}
        </div>

        {draft === null ? (
          <AddRow onStart={() => setDraft(EMPTY_DRAFT)} />
        ) : (
          <NewRule draft={draft} onChange={setDraft} onDone={() => setDraft(null)} />
        )}

        {rules === undefined ? (
          <div aria-busy="true" className="flex flex-col gap-6 py-6">
            <Skeleton className="h-7 w-full" />
            <Skeleton className="h-7 w-5/6" />
            <Skeleton className="h-7 w-4/6" />
          </div>
        ) : (
          <ul aria-label={t("Rules")} className="flex flex-col">
            {rules.map((rule, index) => (
              <RuleRow key={rule.id} rule={rule} index={index} />
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

/** The add row before it is opened: a greyed sentence with dashed blanks. */
function AddRow({ onStart }: { onStart: () => void }) {
  const t = useT();
  const rt = useRichT();

  return (
    <button
      type="button"
      onClick={onStart}
      className={cn(
        ROW,
        "group ui-focus-ring border-border-subtle text-foreground-subtle hover:text-foreground-muted items-center border-t py-4.5 text-left text-2xl transition-colors @max-2xl:text-xl",
      )}
    >
      <span
        aria-hidden
        className="border-border-strong text-foreground-subtle group-hover:border-foreground-muted group-hover:text-foreground flex h-6.5 w-11 items-center justify-center rounded-full border-[1.5px] border-dashed transition-[transform,color,border-color] duration-300 ease-spring group-hover:rotate-90"
      >
        <PlusIcon className="size-3.5" />
      </span>
      <span className="min-w-0">
        {rt("When <u>something</u> arrives, Nova should <u>do something</u>…", {
          u: (chunks: ReactNode) => (
            <span className="bg-[repeating-linear-gradient(90deg,var(--border-strong)_0_4px,transparent_4px_8px)] bg-[length:100%_1.5px] bg-bottom bg-no-repeat pb-0.5">
              {chunks}
            </span>
          ),
        })}
      </span>
      <em className="bg-surface-hover text-foreground-subtle group-hover:bg-ink group-hover:text-ink-foreground rounded-full px-2.5 py-1 text-sm whitespace-nowrap not-italic transition-colors @max-2xl:hidden">
        {t("New rule")}
      </em>
    </button>
  );
}

/** The add row open: the sentence to fill, what it would have matched, and Turn on. */
function NewRule({
  draft,
  onChange,
  onDone,
}: {
  draft: RuleDraft;
  onChange: (draft: RuleDraft) => void;
  onDone: () => void;
}) {
  const t = useT();
  const actions = useTriageActions();
  const input = ruleInput(draft);
  const ready = input !== null;
  const preview = useQuery({ ...queries.inbox.rulePreview(input ?? PLACEHOLDER), enabled: ready });
  const matched = useCountUp(preview.data?.matched ?? 0, preview.data !== undefined, 700);
  const [saving, setSaving] = useState(false);
  const [refusal, setRefusal] = useState<string | null>(null);
  const missing = missingBlanks(draft);

  const turnOn = () => {
    if (input === null) {
      return;
    }
    setSaving(true);
    setRefusal(null);
    actions.createRule(input).then(
      () => {
        setSaving(false);
        onDone();
      },
      (error: unknown) => {
        setSaving(false);
        setRefusal(inlineErrorText(t, error));
      },
    );
  };

  return (
    <section
      aria-label={t("New rule")}
      className={cn(ROW, "border-border-subtle animate-rise border-t py-5")}
      onKeyDown={(event) => {
        if (event.key === "Escape" && event.target === event.currentTarget) {
          onDone();
        }
      }}
    >
      <div className="pt-1">
        <span
          aria-hidden
          className={cn(
            "flex h-6.5 w-11 items-center justify-center rounded-full border-[1.5px] transition-colors",
            ready ? "border-foreground text-foreground border-solid" : "border-border-strong text-foreground-subtle border-dashed",
          )}
        >
          <PlusIcon className="size-3.5" />
        </span>
      </div>
      <div className="min-w-0">
        <RuleSentence draft={draft} onChange={onChange} fresh />
        <div className="text-foreground-muted mt-3.5 flex flex-wrap items-center gap-2 text-sm @max-2xl:flex-col @max-2xl:items-start">
          {ready ? (
            preview.isError ? (
              <span className="text-danger">{inlineErrorText(t, preview.error)}</span>
            ) : preview.data === undefined ? (
              <span className="text-foreground-subtle">{t("Looking back over the last 30 days…")}</span>
            ) : (
              <>
                <b className="text-foreground font-mono text-2xl font-semibold tabular-nums">{matched}</b>
                <span>
                  {t(
                    "{0, plural, one {message last month would have matched} other {messages last month would have matched}}",
                    preview.data.matched,
                  )}
                  {preview.data.off > 0 ? (
                    <>
                      {" · "}
                      <span className="text-warning">
                        {t("{0, plural, one {# looks off} other {# look off}}", preview.data.off)}
                      </span>
                    </>
                  ) : preview.data.matched > 0 ? (
                    <>
                      {" · "}
                      {t("all look right")}
                    </>
                  ) : null}
                </span>
                {preview.data.matched > 0 && (
                  <Button variant="link" size="sm" className="px-1" onClick={() => openMatches({ input })}>
                    {t("See them")}
                  </Button>
                )}
              </>
            )
          ) : (
            <span className="text-foreground-subtle">
              {missing.includes("assignee")
                ? t("Say who to hand it to.")
                : t("Fill in the blanks — it reads like a sentence because it is one.")}
            </span>
          )}
          <span className="flex-1" />
          <span className="flex gap-1.5">
            <Button variant="ghost" size="sm" onClick={onDone}>
              {t("Cancel")}
            </Button>
            <Button size="sm" disabled={!ready} isLoading={saving} loadingText={t("Turning on…")} onClick={turnOn}>
              {t("Turn on")}
            </Button>
          </span>
        </div>
        {refusal !== null && (
          <p role="alert" className="text-danger mt-2 text-sm">
            {refusal}
          </p>
        )}
      </div>
    </section>
  );
}

const PLACEHOLDER: TriageRuleInput = {
  classification: "Other",
  senderScope: "Anyone",
  action: "Ignore",
};

/** One rule: its switch, its sentence (each blank editable) and how often it acted. */
const RuleRow = memo(function RuleRow({ rule, index }: { rule: TriageRule; index: number }) {
  const t = useT();
  const actions = useTriageActions();
  const [leaving, setLeaving] = useState(false);
  const now = Math.floor(Date.now() / 1000);

  const change = (next: RuleDraft) => {
    if (sameRule(rule, next)) {
      return;
    }
    const input = ruleInput(next, rule.enabled);
    if (input !== null) {
      actions.updateRule(rule, input);
    }
  };

  const remove = () => {
    setLeaving(true);
    setTimeout(() => actions.deleteRule(rule), LEAVE_MS);
  };

  let flow: ReactNode;
  if (!rule.enabled) {
    flow = t("Paused");
  } else if (rule.matchCount === 0) {
    flow = t("Waiting for a first match");
  } else {
    flow = (
      <>
        <b className="text-foreground font-semibold tabular-nums">{rule.matchesThisMonth}</b>{" "}
        {t("this month")}
      </>
    );
  }
  const last = rule.lastMatch;

  return (
    <li
      style={{ animationDelay: `${index * 60}ms` }}
      className={cn(
        ROW,
        "group/rule animate-rise border-border-subtle border-t py-5 transition-[opacity,transform] duration-300 ease-settle",
        leaving && "-translate-x-6 opacity-0",
      )}
    >
      <div className="pt-1.5">
        <Switch
          checked={rule.enabled}
          aria-label={rule.enabled ? t("Pause this rule") : t("Turn this rule on")}
          onCheckedChange={(checked) => actions.setRuleEnabled(rule, checked)}
          className="origin-left scale-[1.375]"
        />
      </div>
      <div className="min-w-0">
        <RuleSentence draft={draftFromRule(rule)} onChange={change} quiet={!rule.enabled} />
        <div className="text-foreground-subtle mt-1.5 flex min-h-4.5 items-center gap-2.5 text-sm">
          <span>{flow}</span>
          {rule.enabled && last !== null && (
            <span className="truncate font-mono text-xs opacity-0 transition-opacity duration-200 group-hover/rule:opacity-100 group-focus-within/rule:opacity-100 @max-2xl:hidden">
              {t(
                "Last: {0} · {1}",
                last.reference.trim() === "" ? last.subject : last.reference,
                formatSecondsAgo(now - last.receivedAt),
              )}
            </span>
          )}
        </div>
      </div>
      <div className="flex gap-0.5 pt-1 opacity-0 transition-opacity duration-150 group-hover/rule:opacity-100 group-focus-within/rule:opacity-100 @max-2xl:hidden">
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant="ghost"
                size="icon-sm"
                aria-label={t("See what it matched")}
                onClick={() => openMatches({ ruleId: rule.id })}
              />
            }
          >
            <EyeIcon className="size-3.5" />
          </TooltipTrigger>
          <TooltipContent>{t("See what it matched")}</TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger
            render={
              <Button variant="ghost" size="icon-sm" aria-label={t("Delete rule")} onClick={remove} />
            }
          >
            <XCloseIcon className="size-3.5" />
          </TooltipTrigger>
          <TooltipContent>{t("Delete rule")}</TooltipContent>
        </Tooltip>
      </div>
    </li>
  );
});
