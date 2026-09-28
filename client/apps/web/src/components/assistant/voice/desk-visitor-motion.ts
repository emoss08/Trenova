/**
 * The desk visitor's motion, written once as a pose over time and fitted to
 * CSS keyframes.
 *
 * Every part of the drawing reads the same clock: the sheet rides in the
 * hand, the steps follow the distance covered so the feet never slide, the
 * body leans into the walk and into the reach, and the lamp's pool answers
 * the sheet landing. Written as independent keyframes those drifted apart,
 * so the keyframes are derived instead: `pose` says where everything is at
 * any instant and `deskVisitorKeyframes` fits each part with as few
 * keyframes as hold it within a tolerance, joining them with cubic curves
 * that carry the true speed through each keyframe rather than stopping on
 * it.
 *
 * The drawing reads its geometry from here too, so the pile sits exactly
 * where the hand sets the sheet down. The fitted CSS is committed as
 * `desk-visitor.css`; `pnpm --filter @trenova/web desk-visitor:generate`
 * writes it and a test fails when it is stale.
 *
 * The module imports nothing, so node can run it directly.
 */

/** Where the visitor stands at the desk, and where the ground is. */
export const VISITOR_X = 23.4;
export const GROUND_Y = 23;
export const SHOULDER_Y = 11;
export const HIP_Y = 16.6;

const WALK_DISTANCE = 30;
const ARM_REACH = 4.4;
const LEG_LENGTH = 6.4;
const CARRY_ANGLE = 95;
const SHEET_WIDTH = 5.4;
const SHEET_CENTRE_Y = 10.5;

/** The loop, in seconds, as the design was tuned. */
const TIMING = { walk: 1.8, atDesk: 1.5, empty: 0.8 } as const;

/** How the visitor walks, as the design was tuned. */
const GAIT = {
  swing: 20,
  stepLength: 1,
  arms: 0.8,
  bob: 0.22,
  walkLean: 2,
  reachLean: 5,
  pulse: 0.25,
} as const;

const CRUISE = 0.55;
const CRUISE_AREA = CRUISE + (1 - CRUISE) / 2;

function clamp01(x: number): number {
  return Math.max(0, Math.min(1, x));
}

function smoothstep(x: number): number {
  const u = clamp01(x);

  return u * u * (3 - 2 * u);
}

function easeInOut(x: number): number {
  const u = clamp01(x);

  return u < 0.5 ? 4 * u * u * u : 1 - Math.pow(-2 * u + 2, 3) / 2;
}

function lerp(a: number, b: number, u: number): number {
  return a + (b - a) * u;
}

function radians(degrees: number): number {
  return (degrees * Math.PI) / 180;
}

/** Share of the walk covered by `u` of its time: cruise, then ease to a stop. */
function travelled(u: number): number {
  const x = clamp01(u);
  const covered =
    x < CRUISE ? x : CRUISE + (x - CRUISE) - Math.pow(x - CRUISE, 2) / (2 * (1 - CRUISE));

  return covered / CRUISE_AREA;
}

/** Speed at `u` of the walk, as a share of the cruising speed. */
function pace(u: number): number {
  const x = clamp01(u);

  return x < CRUISE ? 1 : (1 - x) / (1 - CRUISE);
}

type Point = { x: number; y: number };

/** The centre of a sheet held flat on the hand, for an arm angle and a lean. */
function heldSheet(armAngle: number, lean: number, walkX: number, bobY: number): Point {
  const px = VISITOR_X - ARM_REACH * Math.sin(radians(armAngle));
  const py = SHOULDER_Y + ARM_REACH * Math.cos(radians(armAngle));
  const cos = Math.cos(radians(lean));
  const sin = Math.sin(radians(lean));
  const x = VISITOR_X + (px - VISITOR_X) * cos - (py - GROUND_Y) * sin;
  const y = GROUND_Y + (px - VISITOR_X) * sin + (py - GROUND_Y) * cos;

  return { x: x + walkX - 0.6, y: y + bobY - 1 };
}

/** The arm angle that lays the sheet on the pile, leaning into the reach. */
function solveReachAngle(): number {
  let low = 60;
  let high = 125;
  for (let i = 0; i < 48; i++) {
    const mid = (low + high) / 2;
    if (heldSheet(mid, -GAIT.reachLean, 0, 0).y > SHEET_CENTRE_Y) {
      low = mid;
    } else {
      high = mid;
    }
  }

  return (low + high) / 2;
}

function round(n: number, places = 3): number {
  const m = 10 ** places;

  return Math.round(n * m) / m + 0;
}

/** The arm angle the sheet is set down at, and the lean that goes with it. */
export const REACH_ANGLE = round(solveReachAngle(), 2);
export const REACH_LEAN = -GAIT.reachLean;

const LANDING = heldSheet(solveReachAngle(), -GAIT.reachLean, 0, 0);

/** The pile under the lamp: its left edge, its sheets and the one set on it. */
export const PILE = {
  x: round(LANDING.x - SHEET_WIDTH / 2),
  centre: round(LANDING.x),
  width: SHEET_WIDTH,
  sheetY: round(LANDING.y - 0.4),
  rows: [11.1, 12.1, 13.1],
} as const;

/** How far the half-size lamp is shifted so its light falls on the pile. */
export const LAMP_X = round(PILE.centre - 11.75);

const BEATS = (() => {
  const { walk, atDesk, empty } = TIMING;
  const k = atDesk / 1.5;
  const outStart = walk + 1.5 * k;
  const sinkStart = outStart + 0.35 * walk;

  return {
    walk,
    lowerStart: walk + 0.1 * k,
    lowerEnd: walk + 0.5 * k,
    releaseStart: walk + 0.65 * k,
    releaseEnd: walk + 1.05 * k,
    turnStart: walk + 1.15 * k,
    turnEnd: walk + 1.45 * k,
    outStart,
    outEnd: outStart + walk,
    sinkStart,
    sinkEnd: sinkStart + 0.5,
    loop: outStart + walk + empty,
  };
})();

/** One loop, in seconds. */
export const LOOP_SECONDS = round(BEATS.loop, 2);

export type VisitorPose = {
  walk: number;
  fade: number;
  facing: number;
  bob: number;
  lean: number;
  legFront: number;
  legBack: number;
  armFront: number;
  armBack: number;
  sheetX: number;
  sheetY: number;
  sheetFade: number;
  pile: number;
  glow: number;
};

type Stride = { leg: number; bob: number; lean: number };

function stride(walked: number, speed: number): Stride {
  const cycle = 4 * LEG_LENGTH * Math.sin(radians(GAIT.swing)) * GAIT.stepLength;
  const phase = (2 * Math.PI * walked) / cycle;
  const amplitude = smoothstep(Math.min(1, speed * 1.6));

  return {
    leg: GAIT.swing * amplitude * Math.sin(phase),
    bob: GAIT.bob * amplitude * Math.abs(Math.sin(phase)),
    lean: -GAIT.walkLean * amplitude,
  };
}

/** Where every part of the drawing is, `time` seconds into the loop. */
export function pose(time: number): VisitorPose {
  const b = BEATS;
  const p: VisitorPose = {
    walk: 0,
    fade: 1,
    facing: 1,
    bob: 0,
    lean: 0,
    legFront: 0,
    legBack: 0,
    armFront: 0,
    armBack: 0,
    sheetX: 0,
    sheetY: 0,
    sheetFade: 1,
    pile: 0,
    glow: 0,
  };
  let step: Stride | null = null;
  let carried = false;
  let armAngle = CARRY_ANGLE;

  if (time < b.walk) {
    const u = time / b.walk;
    const walked = WALK_DISTANCE * travelled(u);
    p.walk = WALK_DISTANCE - walked;
    step = stride(walked, pace(u));
    carried = true;
    p.fade = smoothstep(walked / 5);
    p.sheetFade = p.fade;
  } else if (time < b.outStart) {
    if (time < b.lowerEnd) {
      const u = easeInOut((time - b.lowerStart) / (b.lowerEnd - b.lowerStart));
      armAngle = lerp(CARRY_ANGLE, REACH_ANGLE, u);
      p.lean = lerp(0, REACH_LEAN, u);
      carried = true;
    } else if (time < b.releaseStart) {
      armAngle = REACH_ANGLE;
      p.lean = REACH_LEAN;
    } else {
      const u = easeInOut((time - b.releaseStart) / (b.releaseEnd - b.releaseStart));
      armAngle = lerp(REACH_ANGLE, 0, u);
      p.lean = lerp(REACH_LEAN, 0, u);
    }
    p.armFront = armAngle;
    if (time >= b.turnStart) {
      p.facing = Math.cos(Math.PI * easeInOut((time - b.turnStart) / (b.turnEnd - b.turnStart)));
    }
  } else if (time < b.outEnd) {
    const u = (time - b.outStart) / b.walk;
    const walked = WALK_DISTANCE * (1 - travelled(1 - u));
    p.walk = walked;
    p.facing = -1;
    step = stride(walked, pace(1 - u));
    p.fade = smoothstep((WALK_DISTANCE - walked) / 5);
  } else {
    const u = easeInOut((time - b.outEnd) / (b.loop - b.outEnd));
    const last = stride(WALK_DISTANCE, 1);
    p.walk = WALK_DISTANCE;
    p.facing = -1;
    p.fade = 0;
    p.legFront = last.leg * (1 - u);
    p.legBack = -last.leg * (1 - u);
    p.bob = last.bob * (1 - u);
    p.lean = last.lean;
    p.armBack = last.leg * GAIT.arms * (1 - u);
    p.armFront = lerp(-last.leg * GAIT.arms, CARRY_ANGLE, u);
  }

  if (step !== null) {
    p.legFront = step.leg;
    p.legBack = -step.leg;
    p.bob = step.bob;
    p.lean = step.lean;
    p.armBack = step.leg * GAIT.arms;
    p.armFront = carried ? CARRY_ANGLE : -step.leg * GAIT.arms;
  }

  if (carried) {
    const walking = time < b.walk;
    const held = heldSheet(walking ? CARRY_ANGLE : armAngle, p.lean, walking ? p.walk : 0, p.bob);
    p.sheetX = held.x - LANDING.x;
    p.sheetY = held.y - LANDING.y;
  }

  p.pile = smoothstep((time - b.sinkStart) / (b.sinkEnd - b.sinkStart));
  if (time >= b.sinkStart) {
    p.sheetY = p.pile;
  }

  const glow = (time - b.lowerEnd) / 0.5;
  p.glow = glow > 0 && glow < 1 ? Math.sin(Math.PI * glow) * GAIT.pulse : 0;

  return p;
}

/** The moment reduced motion draws: the sheet touching down on the pile. */
export const STILL_TIME = BEATS.lowerEnd;

type Channel = {
  part: string;
  value: (p: VisitorPose) => number;
  declare: (v: number) => string;
  tolerance: number;
};

const px = (v: number) => `${round(v)}px`;

/** Each animated part, the one number it moves by, and how close is close enough. */
export const CHANNELS: readonly Channel[] = [
  {
    part: "walk",
    value: (p) => p.walk,
    declare: (v) => `transform: translateX(${px(v)});`,
    tolerance: 0.01,
  },
  {
    part: "fade",
    value: (p) => p.fade,
    declare: (v) => `opacity: ${round(v)};`,
    tolerance: 0.004,
  },
  {
    part: "turn",
    value: (p) => p.facing,
    declare: (v) =>
      `transform: translateX(${VISITOR_X}px) scaleX(${round(v)}) translateX(-${VISITOR_X}px);`,
    tolerance: 0.004,
  },
  {
    part: "bob",
    value: (p) => p.bob,
    declare: (v) => `transform: translateY(${px(v)});`,
    tolerance: 0.015,
  },
  {
    part: "lean",
    value: (p) => p.lean,
    declare: (v) =>
      `transform: translate(${VISITOR_X}px, ${GROUND_Y}px) rotate(${round(v, 2)}deg) translate(-${VISITOR_X}px, -${GROUND_Y}px);`,
    tolerance: 0.08,
  },
  {
    part: "leg-front",
    value: (p) => p.legFront,
    declare: (v) => `transform: rotate(${round(v, 2)}deg);`,
    tolerance: 0.6,
  },
  {
    part: "leg-back",
    value: (p) => p.legBack,
    declare: (v) => `transform: rotate(${round(v, 2)}deg);`,
    tolerance: 0.6,
  },
  {
    part: "arm-front",
    value: (p) => p.armFront,
    declare: (v) => `transform: rotate(${round(v, 2)}deg);`,
    tolerance: 0.6,
  },
  {
    part: "arm-back",
    value: (p) => p.armBack,
    declare: (v) => `transform: rotate(${round(v, 2)}deg);`,
    tolerance: 0.6,
  },
  {
    part: "sheet-x",
    value: (p) => p.sheetX,
    declare: (v) => `transform: translateX(${px(v)});`,
    tolerance: 0.01,
  },
  {
    part: "sheet-y",
    value: (p) => p.sheetY,
    declare: (v) => `transform: translateY(${px(v)});`,
    tolerance: 0.015,
  },
  {
    part: "sheet-fade",
    value: (p) => p.sheetFade,
    declare: (v) => `opacity: ${round(v)};`,
    tolerance: 0.004,
  },
  {
    part: "pile",
    value: (p) => p.pile,
    declare: (v) => `transform: translateY(${px(v)});`,
    tolerance: 0.006,
  },
  {
    part: "pool",
    value: (p) => p.glow,
    declare: (v) =>
      `opacity: ${round(0.8 + v * 0.8)};\n    transform: scaleX(${round(1 + v * 0.5)});`,
    tolerance: 0.003,
  },
  {
    part: "cone",
    value: (p) => p.glow,
    declare: (v) => `opacity: ${round(0.2 + v * 0.25)};`,
    tolerance: 0.003,
  },
];

const SAMPLES = 2400;
const EPSILON = 1e-4;

type Key = { index: number; timing: string | null };

/**
 * The fewest keyframes that hold a part within its tolerance. Each span is a
 * cubic that leaves one keyframe and arrives at the next at the part's true
 * speed, so a curve passes through a keyframe instead of pausing on it.
 * Every turning point is a keyframe, since a swing is what a cubic between
 * two of them draws best; a span that still strays is split where it strays
 * most.
 */
function fit(channel: Channel, times: readonly number[]): Key[] {
  const values = times.map((t) => channel.value(pose(t)));
  const outgoing = (i: number) => (channel.value(pose(times[i]! + EPSILON)) - values[i]!) / EPSILON;
  const incoming = (i: number) => (values[i]! - channel.value(pose(times[i]! - EPSILON))) / EPSILON;

  const segment = (from: number, to: number) => {
    const span = times[to]! - times[from]!;
    const rise = values[to]! - values[from]!;
    if (Math.abs(rise) < 1e-9) {
      return { at: () => values[from]!, timing: null };
    }
    const y1 = (outgoing(from) * span) / rise / 3;
    const y2 = 1 - (incoming(to) * span) / rise / 3;
    const c1 = round(Math.max(-4, Math.min(5, y1)));
    const c2 = round(Math.max(-4, Math.min(5, y2)));
    const at = (u: number) => {
      const v = 1 - u;
      return values[from]! + rise * (3 * v * v * u * c1 + 3 * v * u * u * c2 + u * u * u);
    };
    const linear = Math.abs(c1 - 1 / 3) < 0.002 && Math.abs(c2 - 2 / 3) < 0.002;

    return { at, timing: linear ? null : `cubic-bezier(0.333, ${c1}, 0.667, ${c2})` };
  };

  const keys: Key[] = [];
  const turns = [0];
  for (let i = 1; i < times.length - 1; i++) {
    const before = values[i]! - values[i - 1]!;
    const after = values[i + 1]! - values[i]!;
    if (before * after < 0 || (before === 0) !== (after === 0)) {
      turns.push(i);
    }
  }
  turns.push(times.length - 1);
  const refine = (from: number, to: number) => {
    const { at, timing } = segment(from, to);
    let worst = -1;
    let error = channel.tolerance;
    for (let i = from + 1; i < to; i++) {
      const u = (times[i]! - times[from]!) / (times[to]! - times[from]!);
      const miss = Math.abs(at(u) - values[i]!);
      if (miss > error) {
        error = miss;
        worst = i;
      }
    }
    if (worst === -1) {
      keys.push({ index: from, timing });
      return;
    }
    refine(from, worst);
    refine(worst, to);
  };
  for (let i = 1; i < turns.length; i++) {
    refine(turns[i - 1]!, turns[i]!);
  }
  keys.push({ index: times.length - 1, timing: null });

  return keys;
}

function sampleTimes(): number[] {
  const b = BEATS;
  const times: number[] = [];
  for (let i = 0; i <= SAMPLES; i++) {
    times.push((b.loop * i) / SAMPLES);
  }
  for (const beat of [
    b.walk,
    b.lowerStart,
    b.lowerEnd,
    b.releaseStart,
    b.releaseEnd,
    b.turnStart,
    b.turnEnd,
    b.outStart,
    b.outEnd,
    b.sinkStart,
    b.sinkEnd,
  ]) {
    times.push(beat);
  }

  return [...new Set(times.map((t) => round(t, 6)))].sort((a, c) => a - c);
}

/** The keyframes and animation classes for every part, as committed CSS. */
export function deskVisitorKeyframes(): string {
  const times = sampleTimes();
  const blocks = CHANNELS.map((channel) => {
    const name = `desk-visitor-${channel.part}`;
    const lines = [`@keyframes ${name} {`];
    let last = -1;
    for (const key of fit(channel, times)) {
      const at = times[key.index]!;
      const percent = round((at / BEATS.loop) * 100, 3);
      if (percent <= last) {
        continue;
      }
      last = percent;
      const body = [`    ${channel.declare(channel.value(pose(at)))}`];
      if (key.timing !== null) {
        body.push(`    animation-timing-function: ${key.timing};`);
      }
      lines.push(`  ${percent}% {`, ...body, "  }");
    }
    lines.push("}");

    return lines.join("\n");
  });
  const classes = CHANNELS.map(
    (channel) =>
      `.animate-desk-visitor-${channel.part} {\n  animation: desk-visitor-${channel.part} ${LOOP_SECONDS}s linear infinite;\n}`,
  );

  return [
    "/* Generated by scripts/desk-visitor/generate.mjs from desk-visitor-motion.ts. Do not edit. */",
    "",
    classes.join("\n\n"),
    "",
    blocks.join("\n\n"),
    "",
  ].join("\n");
}
