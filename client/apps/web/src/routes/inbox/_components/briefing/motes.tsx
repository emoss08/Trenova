import { useReducedMotion } from "motion/react";
import { useEffect, useRef } from "react";
import { usePausedWhenAway } from "./use-briefing-motion";

const COUNT = 46;
const MAX_PIXEL_RATIO = 2;

type Mote = {
  x: number;
  y: number;
  radius: number;
  speed: number;
  phase: number;
  alpha: number;
};

function seedMotes(): Mote[] {
  return Array.from({ length: COUNT }, () => ({
    x: Math.random(),
    y: Math.random(),
    radius: 0.4 + Math.random() * 1.1,
    speed: 0.004 + Math.random() * 0.012,
    phase: Math.random() * Math.PI * 2,
    alpha: 0.25 + Math.random() * 0.55,
  }));
}

/**
 * The stars that drift up behind the top of the briefing: the ink colour of
 * the page (white in dark, black in light), fading out towards the edges.
 *
 * They are drawn on a canvas, so nothing about them reaches layout. Drawing
 * stops while the canvas is out of view or the tab is hidden, and under
 * reduced motion they are drawn once and left still.
 */
export function Motes() {
  const ref = useRef<HTMLCanvasElement>(null);
  const reduce = useReducedMotion() ?? false;
  const paused = usePausedWhenAway(ref);
  const motes = useRef<Mote[] | null>(null);
  const clock = useRef(0);

  useEffect(() => {
    const canvas = ref.current;
    const context = canvas?.getContext("2d") ?? null;
    if (canvas === null || context === null) {
      return;
    }
    motes.current ??= seedMotes();
    const field = motes.current;
    const ratio = Math.min(MAX_PIXEL_RATIO, window.devicePixelRatio || 1);
    let width = 0;
    let height = 0;
    let ink = getComputedStyle(canvas).color;

    const resize = () => {
      const rect = canvas.getBoundingClientRect();
      width = rect.width;
      height = rect.height;
      canvas.width = Math.round(width * ratio);
      canvas.height = Math.round(height * ratio);
      context.setTransform(ratio, 0, 0, ratio, 0, 0);
    };
    resize();
    const sizes = new ResizeObserver(() => {
      resize();
      draw(clock.current);
    });
    sizes.observe(canvas);
    const theme = new MutationObserver(() => {
      ink = getComputedStyle(canvas).color;
      draw(clock.current);
    });
    theme.observe(document.documentElement, { attributes: true, attributeFilter: ["class"] });

    function draw(seconds: number) {
      if (context === null) {
        return;
      }
      context.clearRect(0, 0, width, height);
      context.fillStyle = ink;
      context.shadowColor = ink;
      for (const mote of field) {
        const y = (((mote.y - seconds * mote.speed) % 1) + 1) % 1;
        const x = mote.x + Math.sin(seconds * 0.3 + mote.phase) * 0.018;
        const px = x * width;
        const py = y * height;
        const falloff = Math.max(
          0,
          1 - Math.hypot((px - width / 2) / (width * 0.5), (py - height * 0.35) / (height * 0.6)),
        );
        const twinkle = 0.45 + 0.55 * Math.abs(Math.sin(seconds * (0.8 + mote.speed * 60) + mote.phase));
        context.globalAlpha = Math.min(1, mote.alpha * twinkle * falloff);
        context.shadowBlur = mote.radius * 2;
        context.beginPath();
        context.arc(px, py, mote.radius, 0, Math.PI * 2);
        context.fill();
      }
      context.globalAlpha = 1;
    }

    if (reduce || paused) {
      draw(clock.current);
      return () => {
        sizes.disconnect();
        theme.disconnect();
      };
    }

    let frame = 0;
    const began = performance.now() - clock.current * 1000;
    const loop = (now: number) => {
      clock.current = (now - began) / 1000;
      draw(clock.current);
      frame = requestAnimationFrame(loop);
    };
    frame = requestAnimationFrame(loop);

    return () => {
      cancelAnimationFrame(frame);
      sizes.disconnect();
      theme.disconnect();
    };
  }, [reduce, paused]);

  return <canvas ref={ref} aria-hidden className="tb-motes" />;
}
