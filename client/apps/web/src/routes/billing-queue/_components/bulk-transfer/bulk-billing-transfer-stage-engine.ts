import type {
  BillingTransferFailureCode,
  BillingTransferRun,
} from "@/lib/graphql/billing-transfer";
import {
  BILLING_TRANSFER_FAILURE_REASONS,
  READINESS_STEPS,
  type ReadinessStep,
} from "./bulk-billing-transfer-report";
import {
  SHEET_HEIGHT,
  SHEET_WIDTH,
  type SheetFieldState,
} from "./bulk-billing-transfer-stage-sheet";

/**
 * The transfer stage's choreography.
 *
 * The run reports counts, never the shipments behind them, so the stage animates the
 * difference between what the run has answered and what the stage has already shown. Each
 * sheet carries shipments that share one outcome, and as many of them as keeps the stage
 * within a poll of the run, so a fast run never leaves the animation playing catch-up.
 */

export type StageCounts = {
  transferred: number;
  held: Partial<Record<BillingTransferFailureCode, number>>;
  /** The reason the last held-back sheet was stopped for, which the top of that stack shows. */
  lastHeld: BillingTransferFailureCode | null;
};

export type StageSteps = Record<ReadinessStep, SheetFieldState>;

/** One sheet's worth: a count of shipments with one outcome, `code` null for transferred. */
export type StageGroup = { code: BillingTransferFailureCode | null; count: number };

export const IDLE_STEPS: StageSteps = {
  status: "idle",
  documents: "idle",
  rate: "idle",
  billed: "idle",
};

export const STACK_SCALE = 0.55;

const TIMING = {
  lift: 280,
  travel: 1150,
  land: 360,
  step: 540,
  stamp: 380,
  verdict: 520,
  depart: 1150,
  settle: 300,
  idle: 300,
  handover: 900,
};

/** How long the desk spends on one bill, and so how often a new one leaves the stack. */
const CYCLE_MS =
  READINESS_STEPS.length * TIMING.step + TIMING.verdict + TIMING.stamp + TIMING.land + 200;

const EASE = {
  inOut: "cubic-bezier(0.65, 0, 0.35, 1)",
  settle: "cubic-bezier(0.16, 1, 0.3, 1)",
  spring: "cubic-bezier(0.34, 1.4, 0.64, 1)",
};

export function stageCountsOf(run: BillingTransferRun): StageCounts {
  const held: StageCounts["held"] = {};
  for (const reason of run.failureCounts) {
    held[reason.code] = reason.count;
  }

  return {
    transferred: run.transferredCount,
    held,
    lastHeld: run.failureCounts[0]?.code ?? null,
  };
}

export function heldTotal(counts: StageCounts): number {
  let total = 0;
  for (const count of Object.values(counts.held)) total += count ?? 0;
  return total;
}

/** What the run has answered that the stage has not yet sent off the stack, by outcome. */
export function stageDeficit(target: StageCounts, dispatched: StageCounts): StageGroup[] {
  const deficit: StageGroup[] = [];
  const transferred = target.transferred - dispatched.transferred;
  if (transferred > 0) deficit.push({ code: null, count: transferred });

  for (const [code, count] of Object.entries(target.held) as [
    BillingTransferFailureCode,
    number,
  ][]) {
    const left = count - (dispatched.held[code] ?? 0);
    if (left > 0) deficit.push({ code, count: left });
  }

  return deficit;
}

/**
 * The next sheet. Its outcome is drawn in proportion to what is waiting, so held-back sheets
 * are spread through the run rather than all arriving at the end, and it carries half of
 * everything waiting (at least one, at most what its outcome has), so the backlog halves with
 * every sheet the desk reads.
 */
export function pickStageGroup(deficit: StageGroup[], roll: number): StageGroup | null {
  const total = deficit.reduce((sum, group) => sum + group.count, 0);
  if (total === 0) return null;

  const per = Math.max(1, Math.ceil(total / 2));
  let point = Math.min(Math.max(roll, 0), 0.999_999) * total;
  for (const group of deficit) {
    if (point < group.count) return { code: group.code, count: Math.min(group.count, per) };
    point -= group.count;
  }

  const last = deficit[deficit.length - 1];
  return { code: last.code, count: Math.min(last.count, per) };
}

function addGroup(counts: StageCounts, group: StageGroup): StageCounts {
  if (group.code === null) {
    return { ...counts, transferred: counts.transferred + group.count };
  }

  return {
    ...counts,
    held: { ...counts.held, [group.code]: (counts.held[group.code] ?? 0) + group.count },
    lastHeld: group.code,
  };
}

export type StageAnchors = {
  stage: HTMLElement;
  template: HTMLElement;
  source: HTMLElement;
  desk: HTMLElement;
  billing: HTMLElement;
  attention: HTMLElement;
};

type EngineOptions = {
  anchors: () => StageAnchors | null;
  onSteps: (steps: StageSteps) => void;
  onLanded: (counts: StageCounts) => void;
  onSettled: () => void;
};

type Point = { x: number; y: number };

class StageDisposed extends Error {}

function centre(element: HTMLElement, stage: HTMLElement): Point {
  const frame = stage.getBoundingClientRect();
  const box = element.getBoundingClientRect();
  return {
    x: box.left - frame.left - stage.clientLeft + box.width / 2,
    y: box.top - frame.top - stage.clientTop + box.height / 2,
  };
}

/** Whole-pixel positions, so a bill at rest sits on the pixel grid and renders crisp. */
function pose(point: Point, { dy = 0, rotate = 0, scale = 1 } = {}): string {
  const x = Math.round(point.x - SHEET_WIDTH / 2);
  const y = Math.round(point.y - SHEET_HEIGHT / 2 + dy);
  return `translate(${x}px, ${y}px) rotate(${rotate}deg) scale(${scale})`;
}

export class TransferStageEngine {
  private target: StageCounts;
  private terminal = false;
  private landed: StageCounts;
  private dispatched: StageCounts;
  private inFlight = 0;
  private desk: Promise<void> = Promise.resolve();
  private readonly timers = new Set<ReturnType<typeof setTimeout>>();
  private readonly animations = new Set<Animation>();
  private readonly sheets = new Set<HTMLElement>();
  private pumping = false;
  private settled = false;
  private disposed = false;
  private readonly options: EngineOptions;

  constructor(options: EngineOptions, initial: StageCounts) {
    this.options = options;
    this.target = initial;
    this.landed = initial;
    this.dispatched = initial;
  }

  update(target: StageCounts, terminal: boolean) {
    this.target = target;
    this.terminal = terminal;
    if (!this.pumping && !this.disposed) {
      this.pumping = true;
      this.options.onLanded(this.landed);
      this.pump();
    }
    this.maybeSettle();
  }

  dispose() {
    this.disposed = true;
    for (const timer of this.timers) clearTimeout(timer);
    this.timers.clear();
    for (const animation of this.animations) animation.cancel();
    this.animations.clear();
    for (const sheet of this.sheets) sheet.remove();
    this.sheets.clear();
  }

  private pump = () => {
    if (this.disposed || this.settled) return;

    const group = pickStageGroup(stageDeficit(this.target, this.dispatched), Math.random());
    if (group) {
      this.dispatched = addGroup(this.dispatched, group);
      this.inFlight += 1;
      void this.fly(group);
    }
    this.later(this.pump, group ? CYCLE_MS : TIMING.idle);
  };

  private async fly(group: StageGroup) {
    const anchors = this.options.anchors();
    const template = anchors?.template.firstElementChild;
    if (!anchors || !(template instanceof HTMLElement)) {
      this.land(group);
      return;
    }

    const sheet = template.cloneNode(true) as HTMLElement;
    Object.assign(sheet.style, {
      position: "absolute",
      left: "0",
      top: "0",
      zIndex: "3",
      pointerEvents: "none",
      transformOrigin: "50% 50%",
    });
    const from = centre(anchors.source, anchors.stage);
    sheet.style.transform = pose(from, { scale: STACK_SCALE });
    anchors.stage.appendChild(sheet);
    this.sheets.add(sheet);

    try {
      const desk = centre(anchors.desk, anchors.stage);
      const peak = { x: from.x + (desk.x - from.x) * 0.5, y: Math.min(from.y, desk.y) - 40 };
      const lifted = pose(from, { dy: -10, rotate: -1.5, scale: STACK_SCALE * 1.05 });

      sheet.classList.add("ui-lift-whisper");
      await this.play(
        sheet,
        [{ transform: pose(from, { scale: STACK_SCALE }) }, { transform: lifted }],
        {
          duration: TIMING.lift,
          easing: EASE.settle,
        },
      );
      sheet.classList.replace("ui-lift-whisper", "ui-lift");
      await this.play(
        sheet,
        [
          { transform: lifted },
          { transform: pose(peak, { rotate: -3, scale: 0.8 }), offset: 0.5 },
          { transform: pose(desk, { dy: -6, rotate: 0.8 }) },
        ],
        { duration: TIMING.travel, easing: EASE.inOut },
      );
      await this.play(
        sheet,
        [{ transform: pose(desk, { dy: -6, rotate: 0.8 }) }, { transform: pose(desk) }],
        {
          duration: TIMING.land,
          easing: EASE.spring,
        },
      );
      sheet.classList.replace("ui-lift", "ui-lift-whisper");

      const turn = this.desk.then(() => this.review(sheet, group, desk));
      this.desk = turn.catch(() => undefined);
      await turn;

      const to = centre(group.code === null ? anchors.billing : anchors.attention, anchors.stage);
      const held = group.code !== null;
      const mid = { x: desk.x + (to.x - desk.x) * 0.5, y: (desk.y + to.y) / 2 - (held ? 12 : 30) };
      const tilt = Math.round((Math.random() - 0.5) * 6) / 2;
      this.later(() => this.options.onSteps(IDLE_STEPS), 460);

      sheet.classList.replace("ui-lift-whisper", "ui-lift");
      await this.play(
        sheet,
        [
          { transform: pose(desk) },
          { transform: pose(desk, { dy: -8, rotate: held ? 1 : -1, scale: 0.95 }), offset: 0.18 },
          { transform: pose(mid, { rotate: held ? 3 : -2.5, scale: 0.72 }), offset: 0.58 },
          { transform: pose(to, { dy: -8, rotate: tilt, scale: STACK_SCALE * 1.05 }) },
        ],
        { duration: TIMING.depart, easing: EASE.inOut },
      );
      sheet.classList.remove("ui-lift");
      await this.play(
        sheet,
        [
          { transform: pose(to, { dy: -8, rotate: tilt, scale: STACK_SCALE * 1.05 }) },
          { transform: pose(to, { rotate: tilt, scale: STACK_SCALE }) },
        ],
        { duration: TIMING.settle, easing: EASE.spring },
      );
    } catch (error) {
      if (!(error instanceof StageDisposed) && !this.disposed) throw error;
    } finally {
      sheet.remove();
      this.sheets.delete(sheet);
      if (!this.disposed) this.land(group);
    }
  }

  /** The desk reads one bill at a time, a section per step, and stops on the one that fails. */
  private async review(sheet: HTMLElement, group: StageGroup, desk: Point) {
    const failedAt = group.code ? BILLING_TRANSFER_FAILURE_REASONS[group.code].step : null;
    const steps: StageSteps = { ...IDLE_STEPS };
    this.options.onSteps({ ...steps });

    for (const { key } of READINESS_STEPS) {
      const field = sheet.querySelector<HTMLElement>(`[data-field="${key}"]`);
      steps[key] = "active";
      if (field) field.dataset.state = "active";
      this.options.onSteps({ ...steps });
      this.sweep(field);
      await this.wait(TIMING.step);

      const outcome: SheetFieldState = key === failedAt ? "fail" : "pass";
      steps[key] = outcome;
      if (field) field.dataset.state = outcome;
      this.options.onSteps({ ...steps });
      if (outcome === "fail") break;
    }

    if (failedAt) {
      await this.wait(TIMING.verdict);
      return;
    }

    await this.play(
      sheet,
      [
        { transform: pose(desk) },
        { transform: pose(desk, { scale: 0.965 }), offset: 0.45 },
        { transform: pose(desk) },
      ],
      { duration: TIMING.stamp, easing: EASE.inOut },
    );
    sheet.dataset.stamped = "true";
    await this.wait(TIMING.verdict - 160);
  }

  private sweep(field: HTMLElement | null) {
    const sweep = field?.querySelector<HTMLElement>("[data-sweep]");
    if (!sweep) return;

    const animation = sweep.animate(
      [
        { transform: "translateX(-100%)", opacity: 1 },
        { transform: "translateX(170%)", opacity: 1 },
      ],
      { duration: TIMING.step - 80, easing: EASE.inOut },
    );
    this.animations.add(animation);
    animation.finished.then(
      () => this.animations.delete(animation),
      () => this.animations.delete(animation),
    );
  }

  private land(group: StageGroup) {
    this.inFlight -= 1;
    this.landed = addGroup(this.landed, group);
    this.options.onLanded(this.landed);
    this.maybeSettle();
  }

  private maybeSettle() {
    if (!this.terminal || this.settled || this.disposed || this.inFlight > 0) return;
    if (stageDeficit(this.target, this.dispatched).length > 0) return;

    this.settled = true;
    this.later(this.options.onSettled, TIMING.handover);
  }

  /** Runs one move, then writes its end onto the bill and drops the animation, so the bill is
   * re-rendered at its resting size instead of being held as a scaled bitmap. */
  private async play(
    element: HTMLElement,
    keyframes: Keyframe[],
    options: KeyframeAnimationOptions,
  ) {
    if (this.disposed) throw new StageDisposed();

    const animation = element.animate(keyframes, { fill: "forwards", ...options });
    this.animations.add(animation);
    try {
      await animation.finished;
    } catch {
      throw new StageDisposed();
    } finally {
      this.animations.delete(animation);
    }
    if (this.disposed) throw new StageDisposed();

    try {
      animation.commitStyles();
    } catch {
      element.style.transform = String(keyframes[keyframes.length - 1]?.transform ?? "");
    }
    animation.cancel();
  }

  private wait(ms: number) {
    return new Promise<void>((resolve, reject) => {
      this.later(() => (this.disposed ? reject(new StageDisposed()) : resolve()), ms);
    });
  }

  private later(callback: () => void, ms: number) {
    const timer = setTimeout(() => {
      this.timers.delete(timer);
      if (!this.disposed) callback();
    }, ms);
    this.timers.add(timer);
  }
}
