import type { WorkerOverview } from "@/lib/graphql/worker-overview";
import type { WorkerTarget } from "@/routes/worker/use-worker-state";
import { WithheldPanel } from "@trenova/shared/components/withheld-value";
import { ShieldOffIcon, ShieldTickIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import type { WorkerClearanceBlock } from "@trenova/graphql/generated/graphql";
import { ClearanceRunway, clearanceItemName } from "./clearance-runway";
import {
  clearanceItemAction,
  clearanceSummary,
  endsClearance,
  type ClearanceItemAction,
  type WorkerClearance,
  type WorkerClearanceItem,
} from "./runway-model";

const DAY_S = 86_400;
const CREDENTIALS: WorkerTarget = { tab: "qualification", view: "credentials" };

type BlockStep = { label: string; target: WorkerTarget };

function blockReason(block: WorkerClearanceBlock, t: TranslateFn): string {
  switch (block) {
    case "NotEmployed":
      return t("Not currently employed.");
    case "DispatchHold":
      return t("Dispatch is on hold.");
    case "RequiredCredentialLapsed":
      return t("A required credential has lapsed.");
    case "RequiredCredentialMissing":
      return t("A required credential is not on file.");
    case "DispatchCompliance":
      return t("The dispatch compliance check refuses this driver.");
  }
}

function blockStep(block: WorkerClearanceBlock, t: TranslateFn): BlockStep | null {
  switch (block) {
    case "NotEmployed":
      return { label: t("Open history"), target: { tab: "employment", view: "history" } };
    case "DispatchHold":
      return { label: t("Open profile"), target: { tab: "profile" } };
    case "RequiredCredentialMissing":
      return {
        label: t("Add credential"),
        target: { tab: "qualification", view: "credentials", form: "add-credential" },
      };
    case "RequiredCredentialLapsed":
    case "DispatchCompliance":
      return null;
  }
}

/** The band's primary button for an item: the action, naming the item where it reads well. */
function itemStepLabel(action: ClearanceItemAction, name: string, t: TranslateFn): string {
  switch (action) {
    case "renew":
      return t("Renew {0}", name);
    case "add-credential":
      return t("Add {0}", name);
    case "log-query":
      return t("Log a query");
    case "open-training":
      return t("Open {0}", name);
    case "open-review":
      return t("Open review");
    case "start-review":
      return t("Start review");
  }
}

function itemStep(item: WorkerClearanceItem, t: TranslateFn): BlockStep {
  const { action, target } = clearanceItemAction(item);
  return { label: itemStepLabel(action, clearanceItemName(item, t), t), target };
}

function clearedSentence(clearance: WorkerClearance, t: TranslateFn): string {
  const summary = clearanceSummary(clearance);
  if (!summary.stop) {
    return summary.dueAhead > 0
      ? t(
          "Nothing on the record is dated to stop driving. {0, plural, one {One thing comes due} other {# things come due}} ahead.",
          summary.dueAhead,
        )
      : t("Nothing on the record is dated to stop driving.");
  }
  const name = clearanceItemName(summary.stop, t);
  if (summary.dueBefore === 0) return t("{0} lapses first, and nothing comes due before it.", name);
  if (summary.hazmatStopsBefore === 0) {
    return t(
      "{0} lapses first. {1, plural, one {One thing comes due} other {# things come due}} before then, and {1, plural, one {it does not stop driving} other {none of them stops driving}}.",
      name,
      summary.dueBefore,
    );
  }
  return t(
    "{0} lapses first. {1, plural, one {One thing comes due} other {# things come due}} before then; {2, plural, one {# stops hazmat loads} other {# stop hazmat loads}}.",
    name,
    summary.dueBefore,
    summary.hazmatStopsBefore,
  );
}

function ClearedVerdict({
  clearance,
  canOpenCredentials,
  onOpen,
}: {
  clearance: WorkerClearance;
  canOpenCredentials: boolean;
  onOpen: (target: WorkerTarget) => void;
}) {
  const t = useT();
  const { stop } = clearanceSummary(clearance);
  const stopItem = stop ? clearance.items.find((item) => item.key === stop.key) : undefined;
  const step = stopItem ? itemStep(stopItem, t) : null;
  return (
    <>
      <p className="flex items-center gap-1.5 text-xs font-medium text-success">
        <ShieldTickIcon aria-hidden className="size-3.5" />
        {t("Cleared to drive")}
      </p>
      {stop ? (
        <>
          <p className="mt-2.5 text-3xl font-semibold tabular-nums">
            {t("{0, plural, one {# day} other {# days}}", Math.max(0, stop.daysUntil))}
          </p>
          <p className="text-base text-muted-foreground tabular-nums">
            {t("through {0}", formatUnixDateMedium(clearance.clearedThrough ?? stop.dueAt))}
          </p>
        </>
      ) : null}
      <p className="mt-3 max-w-[34ch] text-sm text-pretty text-muted-foreground">
        {clearedSentence(clearance, t)}
      </p>
      <Actions step={step} canOpenCredentials={canOpenCredentials} onOpen={onOpen} />
    </>
  );
}

function NotClearedVerdict({
  clearance,
  asOf,
  canOpenCredentials,
  onOpen,
}: {
  clearance: WorkerClearance;
  asOf: number;
  canOpenCredentials: boolean;
  onOpen: (target: WorkerTarget) => void;
}) {
  const t = useT();
  const lapsed = clearance.items.find((item) => item.daysUntil < 0 && endsClearance(item));
  const step =
    (lapsed ? itemStep(lapsed, t) : null) ??
    clearance.blocks.map((block) => blockStep(block, t)).find((found) => found !== null) ??
    null;
  const since = clearance.notClearedSince;
  return (
    <>
      <p className="flex items-center gap-1.5 text-xs font-medium text-danger">
        <ShieldOffIcon aria-hidden className="size-3.5" />
        {t("Not cleared to drive")}
      </p>
      {since ? (
        <>
          <p className="mt-2.5 text-3xl font-semibold text-danger tabular-nums">
            {t(
              "{0, plural, one {# day} other {# days}}",
              Math.max(0, Math.floor((asOf - since) / DAY_S)),
            )}
          </p>
          <p className="text-base text-muted-foreground tabular-nums">
            {t("since {0}", formatUnixDateMedium(since))}
          </p>
        </>
      ) : null}
      {clearance.blocks.length > 0 ? (
        <ul className="mt-3 flex max-w-[34ch] flex-col gap-1 text-sm text-pretty text-muted-foreground">
          {clearance.blocks.map((block) => (
            <li key={block}>{blockReason(block, t)}</li>
          ))}
        </ul>
      ) : null}
      <Actions step={step} canOpenCredentials={canOpenCredentials} onOpen={onOpen} />
    </>
  );
}

function Actions({
  step,
  canOpenCredentials,
  onOpen,
}: {
  step: BlockStep | null;
  canOpenCredentials: boolean;
  onOpen: (target: WorkerTarget) => void;
}) {
  const t = useT();
  if (!step && !canOpenCredentials) return null;
  return (
    <div className="mt-auto flex flex-wrap gap-2 pt-4">
      {step ? (
        <Button type="button" size="sm" onClick={() => onOpen(step.target)}>
          {step.label}
        </Button>
      ) : null}
      {canOpenCredentials ? (
        <Button type="button" size="sm" variant="quiet" onClick={() => onOpen(CREDENTIALS)}>
          {t("Open credentials")}
        </Button>
      ) : null}
    </div>
  );
}

/**
 * Whether the worker may be put on a load, until when and why, beside the runway of
 * everything that comes due. Without credentials read there is no verdict to give: the band
 * says so instead of drawing one built on evidence the reader may not see.
 */
export function ClearanceBand({
  overview,
  canOpenCredentials,
  onOpen,
}: {
  overview: WorkerOverview;
  canOpenCredentials: boolean;
  onOpen: (target: WorkerTarget) => void;
}) {
  const t = useT();
  const { clearance } = overview;

  if (!clearance) {
    return (
      <section
        aria-label={t("Clearance")}
        data-testid="clearance-withheld"
        className="flex rounded-lg border bg-card"
      >
        <WithheldPanel>
          {t(
            "You don't have access to credentials, so this page gives no verdict on whether the driver is cleared to drive.",
          )}
        </WithheldPanel>
      </section>
    );
  }

  return (
    <section
      aria-label={t("Clearance")}
      data-cleared={clearance.cleared ? "" : undefined}
      className="grid min-w-0 rounded-lg border bg-card @[1000px]/overview:grid-cols-[19rem_minmax(0,1fr)]"
    >
      <div className="flex min-w-0 flex-col border-b border-border-subtle p-4 @[1000px]/overview:border-r @[1000px]/overview:border-b-0">
        {clearance.cleared ? (
          <ClearedVerdict
            clearance={clearance}
            canOpenCredentials={canOpenCredentials}
            onOpen={onOpen}
          />
        ) : (
          <NotClearedVerdict
            clearance={clearance}
            asOf={overview.asOf}
            canOpenCredentials={canOpenCredentials}
            onOpen={onOpen}
          />
        )}
      </div>
      <div className="min-w-0 px-4 py-3">
        <ClearanceRunway
          clearance={clearance}
          asOf={overview.asOf}
          variant="lanes"
          onOpen={onOpen}
        />
      </div>
    </section>
  );
}
