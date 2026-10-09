import { humanizeToolName } from "@/components/assistant/proposal-state";
import { toolRefusal, type ToolRefusal, type ToolStep } from "@/components/assistant/activity";
import type { AssistantProposal } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useMemo, useState } from "react";
import { DeskErrorButton, DeskErrorCard, DeskErrorLink } from "../desk-error-card";
import { DeskIcon, type DeskIconName } from "../desk-icons";

/** One record a write or a step did not get to, and why. */
export type BulkItem = { id: string; name: string; reason: string };

/**
 * A long list of records that did not go through, short by default: the
 * reasons as chips that filter it, the first few rows, and "Show all" for
 * the rest with a filter box once the list is long.
 */
export function DeskBulkList({
  items,
  limit = 3,
  of = "records",
}: {
  items: readonly BulkItem[];
  limit?: number;
  /** What the list holds, so its search box can name it whole in any language. */
  of?: "records" | "steps";
}) {
  const t = useT();
  const reasons = useMemo(() => {
    const counts = new Map<string, number>();
    for (const item of items) counts.set(item.reason, (counts.get(item.reason) ?? 0) + 1);
    return [...counts.entries()];
  }, [items]);
  const [reason, setReason] = useState<string | null>(null);
  const [all, setAll] = useState(false);
  const [query, setQuery] = useState("");

  let rows = reason ? items.filter((item) => item.reason === reason) : items;
  if (query.trim() !== "") {
    const needle = query.trim().toLowerCase();
    rows = rows.filter((item) => `${item.id} ${item.name}`.toLowerCase().includes(needle));
  }
  const shown = all ? rows : rows.slice(0, limit);

  return (
    <div className="dk-bl">
      {reasons.length > 1 && (
        <div className="dk-bl-rs">
          {reasons.map(([text, count]) => (
            <Button
              key={text}
              variant="bare"
              size="bare"
              className={cn(
                "h-6 gap-1.5 rounded-full pr-2.25 pl-1.75 text-xs text-dsk-fg2 transition-colors duration-120",
                reason === text && "dk-on",
              )}
              aria-pressed={reason === text}
              onClick={() => setReason((value) => (value === text ? null : text))}
            >
              <b>{count}</b>
              {text}
            </Button>
          ))}
        </div>
      )}
      {all && items.length > 8 && (
        <label className="dk-bl-q">
          <DeskIcon name="search" size={12} />
          <input
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder={
              of === "steps"
                ? t("Filter {0, plural, one {# step} other {# steps}}", rows.length)
                : t("Filter {0, plural, one {# record} other {# records}}", rows.length)
            }
            aria-label={of === "steps" ? t("Filter the steps") : t("Filter the records")}
          />
        </label>
      )}
      <div className={cn("dk-bl-l", all && "dk-all")}>
        {shown.map((item, index) => (
          <div key={`${item.id}-${index}`}>
            <span className="dk-ax-id">{item.id}</span>
            <span>{item.name}</span>
            {!reason && reasons.length > 1 ? <em>{item.reason}</em> : <em />}
          </div>
        ))}
        {shown.length === 0 && (
          <div className="dk-bl-none">{t("Nothing matches “{0}”", query)}</div>
        )}
      </div>
      {rows.length > limit && (
        <div className="dk-bl-f">
          <DeskErrorLink onClick={() => setAll((value) => !value)}>
            {all ? t("Show fewer") : t("Show all {0}", rows.length)}
          </DeskErrorLink>
        </div>
      )}
    </div>
  );
}

/**
 * An approved change that did not all go through: how many of how many,
 * that the ones that went through are final and the rest were left as they
 * were, the list of the rest with why, and a way to ask the agent to fix
 * them. A change that failed outright says so with the reason it gave.
 */
export function DeskWriteResultCard({
  proposal,
  onAsk,
}: {
  proposal: Pick<AssistantProposal, "status" | "executionError" | "executionResult">;
  onAsk?: (text: string) => void;
}) {
  const t = useT();
  const result = proposal.executionResult;
  const failed = result?.failed ?? [];

  if (proposal.status === "ExecutionFailed") {
    return (
      <DeskErrorCard
        tone="err"
        title={t("The change didn't go through")}
        sub={proposal.executionError || t("Nothing was changed.")}
        actions={
          onAsk ? (
            <DeskErrorButton
              ink
              onClick={() => onAsk(t("The change didn't go through. Can you fix it?"))}
            >
              {t("Ask the agent to fix it")}
            </DeskErrorButton>
          ) : undefined
        }
      />
    );
  }
  if (!result || failed.length === 0) {
    return null;
  }

  const total = result.total ?? failed.length;
  const done = Math.max(0, total - failed.length);
  // A failed record is named by its label, the way the app names it; the id
  // stands in only when the write could not name it.
  const items = failed.map((item) => ({
    id: item.label || item.id,
    name: "",
    reason: item.reason,
  }));

  return (
    <DeskErrorCard
      tone="err"
      title={t("{0} of {1} went through · {2} didn't", done, total, failed.length)}
      sub={t(
        "{0, plural, one {The one that didn't go through stayed as it was.} other {The # that didn't go through stayed as they were.}} {1, plural, one {The one that went through is final.} other {The # that went through are final.}}",
        failed.length,
        done,
      )}
    >
      <DeskBulkList items={items} />
      {onAsk && (
        <div className="dk-ec-acts">
          <DeskErrorButton
            ink
            onClick={() =>
              onAsk(
                t(
                  "{0, plural, one {# change didn't go through.} other {# changes didn't go through.}} Look at why and fix it: {1}",
                  failed.length,
                  failed
                    .slice(0, 20)
                    .map((item) => `${item.label || item.id} (${item.reason})`)
                    .join("; "),
                ),
              )
            }
          >
            {t("Ask the agent to fix these {0}", failed.length)}
          </DeskErrorButton>
        </div>
      )}
    </DeskErrorCard>
  );
}

type StepKind = "failed" | ToolRefusal;

const KIND_ORDER: StepKind[] = ["failed", "denied", "over_budget", "invalid", "duplicate"];

const KIND_ICON: Record<StepKind, DeskIconName> = {
  failed: "x",
  denied: "lock",
  over_budget: "alert",
  invalid: "info",
  duplicate: "undo",
};

const KIND_CLASS: Record<StepKind, string> = {
  failed: "dk-s-failed",
  denied: "dk-s-denied",
  over_budget: "dk-s-budget",
  invalid: "dk-s-invalid",
  duplicate: "dk-s-dup",
};

function kindLabel(kind: StepKind, count: number, t: TranslateFn): string {
  switch (kind) {
    case "failed":
      return t("{0, plural, one {# didn't work} other {# didn't work}}", count);
    case "denied":
      return t("{0, plural, one {# not permitted} other {# not permitted}}", count);
    case "over_budget":
      return t("{0, plural, one {# out of budget} other {# out of budget}}", count);
    case "invalid":
      return t("{0, plural, one {# not accepted} other {# not accepted}}", count);
    case "duplicate":
      return t("{0, plural, one {# skipped} other {# skipped}}", count);
  }
}

/**
 * Why a step did not go through, from the first line of its result, without
 * the runtime's stock opening ("Tool \"x\" was not run:"), which the row
 * already says by naming the step.
 */
function stepReason(step: ToolStep): string {
  const line = step.content.split("\n").find((part) => part.trim() !== "") ?? "";
  const reason = line.replace(/^Tool\s+"[^"]+"\s+(was not run|failed)\s*[:.]\s*/u, "").trim();
  const text = reason.charAt(0).toUpperCase() + reason.slice(1);
  return text.length > 140 ? `${text.slice(0, 139)}…` : text;
}

/**
 * The steps of a reply that did not go through, folded into one line under
 * it: how many of how many, a dot for each kind. Opened, each kind is a row
 * with what the steps were and why, and a kind with several steps opens to
 * list them.
 */
export function DeskStepFailures({ steps }: { steps: readonly ToolStep[] }) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const [selected, setSelected] = useState<StepKind | null>(null);

  const groups = useMemo(() => {
    const byKind = new Map<StepKind, ToolStep[]>();
    for (const step of steps) {
      if (step.status !== "failed") continue;
      const kind: StepKind = toolRefusal(step) ?? "failed";
      byKind.set(kind, [...(byKind.get(kind) ?? []), step]);
    }
    return KIND_ORDER.flatMap((kind) => {
      const list = byKind.get(kind);
      return list ? [{ kind, steps: list }] : [];
    });
  }, [steps]);

  const failedCount = groups.reduce((sum, group) => sum + group.steps.length, 0);
  if (failedCount === 0) {
    return null;
  }

  return (
    <div className={cn("dk-sf", open && "dk-open")}>
      <Button
        variant="bare"
        size="bare"
        className="dk-sf-h flex h-9 w-full justify-start gap-2.25 pr-2.5 pl-2.75 text-left text-sm transition-colors duration-120 hover:bg-dsk-hover [&_b]:font-medium [&_b]:text-dsk-fg"
        aria-expanded={open}
        onClick={() => {
          setOpen((value) => !value);
          setSelected(null);
        }}
      >
        <span className="dk-sf-ic">
          <DeskIcon name="alert" size={12} stroke={2.2} />
        </span>
        <span>
          <b>{t("{0} of {1} steps didn't go through", failedCount, steps.length)}</b>
        </span>
        <span className="dk-sf-dots">
          {groups.map((group) => (
            <i key={group.kind} className={KIND_CLASS[group.kind]} />
          ))}
        </span>
        <span className="dk-sf-cv">
          <DeskIcon name="chevR" size={11} stroke={2.2} />
        </span>
      </Button>
      {open && (
        <div className="dk-sf-l">
          {groups.map((group) => {
            const phrases = [...new Set(group.steps.map((step) => humanizeToolName(step.name)))];
            const reasons = [...new Set(group.steps.map(stepReason).filter(Boolean))];
            const many = group.steps.length > 1;
            return (
              <div
                key={group.kind}
                className={cn(
                  "dk-sf-r",
                  KIND_CLASS[group.kind],
                  selected === group.kind && "dk-on",
                )}
              >
                <Button
                  variant="bare"
                  size="bare"
                  className={cn(
                    "dk-sf-rh flex min-h-7.5 w-full justify-start gap-2 pr-2.5 pl-2.75 text-left text-sm hover:bg-dsk-hover [&_b]:font-medium [&_b]:whitespace-nowrap",
                    many ? "cursor-pointer" : "cursor-default",
                  )}
                  onClick={() =>
                    many && setSelected((value) => (value === group.kind ? null : group.kind))
                  }
                >
                  <span className="dk-ec-sic">
                    <DeskIcon name={KIND_ICON[group.kind]} size={10} stroke={2.4} />
                  </span>
                  <b>{kindLabel(group.kind, group.steps.length, t)}</b>
                  <span>{[phrases.join(", "), reasons[0]].filter(Boolean).join(" · ")}</span>
                  {many && (
                    <span className="dk-sf-cv">
                      <DeskIcon name="chevR" size={10} stroke={2.2} />
                    </span>
                  )}
                </Button>
                {selected === group.kind && (
                  <div className="dk-sf-items">
                    <DeskBulkList
                      limit={5}
                      of="steps"
                      items={group.steps.map((step) => ({
                        id: step.name,
                        name: humanizeToolName(step.name),
                        reason: stepReason(step),
                      }))}
                    />
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
