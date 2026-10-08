import { cn } from "@trenova/shared/lib/utils";
import {
  useEffect,
  useRef,
  type CSSProperties,
  type KeyboardEvent,
  type PointerEvent,
  type ReactNode,
} from "react";

type MagicProps = {
  /** The two ends of the glow that follows the pointer. */
  from: string;
  to: string;
  className?: string;
  label?: string;
  onClick: () => void;
  children: ReactNode;
};

type Glow = { x: number; y: number; tx: number; ty: number; o: number; to: number; raf: number };

/**
 * A card that lights where the pointer is: a soft orb eases after it and the border
 * picks up the glow. It is a button; the motion stops when the pointer leaves, and
 * there is none under reduced motion, where the CSS keeps the orb hidden.
 */
export function Magic({ from, to, className, label, onClick, children }: MagicProps) {
  const root = useRef<HTMLDivElement>(null);
  const orb = useRef<HTMLSpanElement>(null);
  const glow = useRef<Glow>({ x: -200, y: -200, tx: -200, ty: -200, o: 0, to: 0, raf: 0 });

  useEffect(() => {
    const state = glow.current;
    return () => cancelAnimationFrame(state.raf);
  }, []);

  const frame = () => {
    const state = glow.current;
    state.x += (state.tx - state.x) * 0.22;
    state.y += (state.ty - state.y) * 0.22;
    state.o += (state.to - state.o) * 0.16;
    if (orb.current) {
      orb.current.style.transform = `translate(${state.x}px,${state.y}px) translate(-50%,-50%)`;
      orb.current.style.opacity = String(state.o);
    }
    root.current?.style.setProperty("--mx", `${state.tx}px`);
    root.current?.style.setProperty("--my", `${state.ty}px`);
    const moving =
      Math.abs(state.tx - state.x) > 0.5 ||
      Math.abs(state.ty - state.y) > 0.5 ||
      Math.abs(state.to - state.o) > 0.01;
    if (moving) {
      state.raf = requestAnimationFrame(frame);
      return;
    }
    state.raf = 0;
    state.o = state.to;
    if (orb.current) orb.current.style.opacity = String(state.o);
  };
  const kick = () => {
    if (!glow.current.raf) glow.current.raf = requestAnimationFrame(frame);
  };
  const follow = (event: PointerEvent<HTMLDivElement>) => {
    const box = root.current?.getBoundingClientRect();
    if (!box) return;
    const state = glow.current;
    state.tx = event.clientX - box.left;
    state.ty = event.clientY - box.top;
    if (state.o < 0.02 && state.to === 0) {
      state.x = state.tx;
      state.y = state.ty;
    }
    kick();
  };
  const onKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if ((event.key === "Enter" || event.key === " ") && event.target === event.currentTarget) {
      event.preventDefault();
      onClick();
    }
  };

  return (
    <div
      ref={root}
      role="button"
      tabIndex={0}
      aria-label={label}
      className={cn("mg", className)}
      style={{ "--gf": from, "--gt": to } as CSSProperties}
      onClick={onClick}
      onKeyDown={onKeyDown}
      onPointerMove={follow}
      onPointerEnter={(event) => {
        follow(event);
        glow.current.to = 0.9;
        kick();
      }}
      onPointerLeave={() => {
        glow.current.to = 0;
        kick();
      }}
    >
      <span className="mg-bg" />
      <span ref={orb} className="mg-orb" aria-hidden="true" />
      <span className="mg-c">{children}</span>
    </div>
  );
}
