import { repoRoot } from "@/test/go-source";
import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";
import {
  CHANNELS,
  LOOP_SECONDS,
  PILE,
  REACH_ANGLE,
  REACH_LEAN,
  STILL_TIME,
  deskVisitorKeyframes,
  pose,
  type VisitorPose,
} from "../voice/desk-visitor-motion";

const STEPS = 3000;

const committed = readFileSync(
  join(repoRoot(), "client/apps/web/src/components/assistant/voice/desk-visitor.css"),
  "utf8",
);

function samples(): VisitorPose[] {
  return Array.from({ length: STEPS }, (_, i) => pose((LOOP_SECONDS * i) / STEPS));
}

describe("desk visitor motion", () => {
  it("is committed as generated", () => {
    expect(committed).toBe(deskVisitorKeyframes());
  });

  it("animates every part through the whole loop", () => {
    for (const channel of CHANNELS) {
      const block = committed.match(
        new RegExp(`@keyframes desk-visitor-${channel.part} \\{([\\s\\S]*?)\\n\\}`),
      )?.[1];

      expect(block, channel.part).toBeDefined();
      expect(block, channel.part).toMatch(/^\s+0% \{/);
      expect(block, channel.part).toMatch(/\n\s+100% \{[^}]*\}\s*$/);
      expect(committed).toContain(
        `.animate-desk-visitor-${channel.part} {\n  animation: desk-visitor-${channel.part} ${LOOP_SECONDS}s linear infinite;`,
      );
    }
  });

  it("moves every part continuously, with no jump anywhere inside the loop", () => {
    const frames = samples();
    const limits: Partial<Record<keyof VisitorPose, number>> = {
      legFront: 2,
      legBack: 2,
      armFront: 2,
      armBack: 2,
      lean: 0.5,
    };

    for (const key of Object.keys(frames[0]!) as (keyof VisitorPose)[]) {
      const limit = limits[key] ?? 0.1;
      let worst = 0;
      for (let i = 1; i < frames.length; i++) {
        worst = Math.max(worst, Math.abs(frames[i]![key] - frames[i - 1]![key]));
      }
      expect(worst, key).toBeLessThan(limit);
    }
  });

  it("lays the sheet on the pile at the moment reduced motion draws", () => {
    const still = pose(STILL_TIME);

    expect(still.sheetX).toBeCloseTo(0, 6);
    expect(still.sheetY).toBeCloseTo(0, 6);
    expect(still.armFront).toBeCloseTo(REACH_ANGLE, 1);
    expect(still.lean).toBe(REACH_LEAN);
    expect(still.walk).toBe(0);
    expect(still.facing).toBe(1);
    expect(still.fade).toBe(1);
    expect(still.pile).toBe(0);
    expect([still.legFront, still.legBack, still.armBack, still.bob, still.glow]).toEqual([
      0, 0, 0, 0, 0,
    ]);
  });

  it("sets the sheet down one sheet above the pile, so the sunk pile is where it started", () => {
    expect(PILE.sheetY + 1).toBeCloseTo(PILE.rows[0]!, 6);
    PILE.rows.slice(1).forEach((y, i) => expect(y - PILE.rows[i]!).toBeCloseTo(1, 6));
  });

  it("closes the loop on the frame it opens with", () => {
    const start = pose(0);
    const end = pose(LOOP_SECONDS - 1e-6);

    expect(start.fade).toBe(0);
    expect(end.fade).toBe(0);
    expect(start.sheetFade).toBe(0);
    expect(start.pile).toBe(0);
    expect(end.pile).toBeCloseTo(1, 6);
    expect(end.sheetX).toBe(0);
    expect(end.sheetY).toBeCloseTo(1, 6);
    expect(end.walk).toBe(start.walk);
    for (const key of ["legFront", "legBack", "armBack", "bob", "lean"] as const) {
      expect(end[key], key).toBeCloseTo(start[key], 3);
    }
    expect(end.armFront).toBeCloseTo(start.armFront, 3);
  });

  it("carries the sheet in and walks off empty-handed", () => {
    const frames = samples();
    const arriving = frames.slice(0, Math.floor(STEPS * 0.25));
    const leaving = frames.slice(Math.floor(STEPS * 0.65), Math.floor(STEPS * 0.8));

    expect(arriving.some((p) => p.sheetX > 10 && p.sheetFade > 0.5)).toBe(true);
    expect(leaving.every((p) => p.sheetX === 0)).toBe(true);
    expect(leaving.some((p) => p.walk > 10 && p.facing === -1 && p.fade > 0.5)).toBe(true);
  });
});
