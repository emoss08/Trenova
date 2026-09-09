import { cn } from "@trenova/shared/lib/utils";
import { useReducedMotion } from "motion/react";
import {
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type ReactNode,
  type RefObject,
} from "react";

const COUNT_DURATION_MS = 420;

/**
 * Eases a displayed number towards `target`, so the permission tally on the role step
 * and the panel's live figures count rather than snap.
 */
function useAnimatedNumber(target: number, enabled: boolean): number {
  // Holds the interpolated value only while a tween is in flight, and returns to null
  // when it lands. Falling back to the prop rather than mirroring it into state means
  // the settled value is always exactly the target, and there is nothing to synchronise
  // when motion is off.
  const [tween, setTween] = useState<number | null>(null);
  const fromRef = useRef(target);

  useEffect(() => {
    const from = fromRef.current;
    fromRef.current = target;
    if (!enabled || from === target) {
      return;
    }

    let frame = 0;
    const start = performance.now();
    const tick = (now: number) => {
      const progress = Math.min(1, (now - start) / COUNT_DURATION_MS);
      if (progress >= 1) {
        setTween(null);
        return;
      }
      const eased = 1 - (1 - progress) ** 3;
      setTween(Math.round(from + (target - from) * eased));
      frame = requestAnimationFrame(tick);
    };

    frame = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(frame);
  }, [target, enabled]);

  return tween ?? target;
}

export function Tally({ value }: { value: number }) {
  const prefersReducedMotion = useReducedMotion();
  const display = useAnimatedNumber(value, !prefersReducedMotion);
  return <>{display.toLocaleString()}</>;
}

export function StepCrumbs({ left, right }: { left: ReactNode; right: ReactNode }) {
  return (
    <div className="mb-3.5 flex items-center justify-between">
      <span className="text-subtle-foreground font-table text-[11px]">{left}</span>
      <span className="text-subtle-foreground font-table text-[11px]">{right}</span>
    </div>
  );
}

export function StepHeading({ title, children }: { title: string; children?: ReactNode }) {
  return (
    <>
      <h1 className="m-0 text-[17px] font-semibold tracking-[-0.028em]">{title}</h1>
      {children ? (
        <p className="text-muted-foreground mt-1 mb-0 text-[12.5px]">{children}</p>
      ) : null}
    </>
  );
}

export function KeyHint({ children }: { children: ReactNode }) {
  return (
    <span className="border-border-2 text-subtle-foreground font-table rounded border px-[5px] py-px text-[10px] whitespace-nowrap">
      {children}
    </span>
  );
}

export function AuthTray({ onBack, hints }: { onBack: () => void; hints: ReactNode }) {
  return (
    <div className="text-subtle-foreground mt-3.5 flex items-center justify-between gap-3 text-[11.5px] whitespace-nowrap">
      <button
        type="button"
        onClick={onBack}
        className="text-muted-foreground hover:text-foreground inline-flex cursor-pointer items-center gap-1.5 text-[12px] transition-colors duration-150"
      >
        <BackArrow />
        Back
      </button>
      <span className="hidden items-center gap-1.5 sm:flex">{hints}</span>
    </div>
  );
}

function BackArrow() {
  return (
    <svg
      viewBox="0 0 24 24"
      width="13"
      height="13"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M19 12H5M11 6l-6 6 6 6" />
    </svg>
  );
}

/**
 * A selectable row: organization on step 2, role on step 3. Multi-select rows report
 * `aria-pressed`; single-select rows behave as a radio group and report `aria-checked`.
 */
export function AuthOption({
  selected,
  onSelect,
  leading,
  name,
  meta,
  chip,
  shortcut,
  disabled,
  role,
}: {
  selected: boolean;
  onSelect: () => void;
  leading: ReactNode;
  name: string;
  meta?: string;
  chip?: string;
  shortcut?: string;
  disabled?: boolean;
  role: "radio" | "checkbox";
}) {
  return (
    <button
      type="button"
      role={role}
      aria-checked={selected}
      aria-pressed={role === "checkbox" ? selected : undefined}
      data-selected={selected}
      data-auth-option=""
      disabled={disabled}
      onClick={onSelect}
      className="auth-option group border-border-2 bg-field relative flex w-full cursor-pointer items-center gap-[11px] rounded-[10px] border px-3 py-[11px] text-left transition-[border-color,background-color,transform] duration-[180ms] active:scale-[0.994] disabled:cursor-not-allowed disabled:opacity-60"
    >
      <span
        className={cn(
          "border-border-2 bg-popover grid size-8 shrink-0 place-items-center rounded-lg border transition-colors duration-[180ms]",
          "font-table text-[11px] font-medium",
          selected ? "border-border text-foreground" : "text-muted-foreground",
        )}
      >
        {leading}
      </span>
      <span className="min-w-0 flex-1">
        <span className="block truncate text-[13px] font-[550] tracking-[-0.005em]">{name}</span>
        {meta ? (
          <span className="text-subtle-foreground block truncate text-[11.5px]">{meta}</span>
        ) : null}
      </span>
      {chip ? (
        <span className="border-border-2 text-subtle-foreground rounded-full border px-2 py-0.5 text-[10.5px] font-medium whitespace-nowrap">
          {chip}
        </span>
      ) : null}
      {shortcut ? (
        <span
          aria-hidden="true"
          className="border-border-2 text-subtle-foreground font-table hidden rounded border px-[5px] py-px text-[10px] opacity-0 transition-opacity duration-150 group-hover:opacity-100 group-focus-visible:opacity-100 sm:inline"
        >
          {shortcut}
        </span>
      ) : null}
      <span
        aria-hidden="true"
        className={cn(
          "border-input grid size-[18px] shrink-0 place-items-center rounded-full border transition-[background-color,border-color] duration-[180ms]",
          selected && "bg-foreground border-foreground",
        )}
      >
        <svg
          viewBox="0 0 24 24"
          width="11"
          height="11"
          fill="none"
          stroke="currentColor"
          strokeWidth="2.6"
          strokeLinecap="round"
          strokeLinejoin="round"
          className={cn(
            "text-background transition-[opacity,transform] duration-200",
            selected ? "scale-100 opacity-100" : "scale-50 opacity-0",
          )}
        >
          <path d="M5 12.5l4.5 4.5L19 7" />
        </svg>
      </span>
    </button>
  );
}

/**
 * ↑/↓ walk the option rows (wrapping), the platform modifier plus 1…9 picks one, and
 * the modifier plus Enter advances the step. Scoped to the org and role steps by the
 * caller mounting the hook only while one of them is on screen.
 */
export function useOptionListKeyboard({
  listRef,
  onSelectIndex,
  onAdvance,
  enabled,
}: {
  listRef: RefObject<HTMLDivElement | null>;
  onSelectIndex: (index: number) => void;
  onAdvance: () => void;
  enabled: boolean;
}) {
  // Latest-callback refs, updated in an effect that runs after every render. The window
  // listener is registered once and must not close over the callbacks a stale render
  // captured, but writing the refs during render would be a render-phase side effect.
  const selectRef = useRef(onSelectIndex);
  const advanceRef = useRef(onAdvance);

  useEffect(() => {
    selectRef.current = onSelectIndex;
    advanceRef.current = onAdvance;
  });

  useEffect(() => {
    if (!enabled) {
      return;
    }

    const onKeyDown = (event: KeyboardEvent) => {
      const modifier = event.metaKey || event.ctrlKey;

      if (modifier && event.key === "Enter") {
        event.preventDefault();
        advanceRef.current();
        return;
      }

      if (modifier && /^[1-9]$/.test(event.key)) {
        event.preventDefault();
        selectRef.current(Number(event.key) - 1);
        return;
      }

      if (event.key !== "ArrowDown" && event.key !== "ArrowUp") {
        return;
      }

      const options = Array.from(
        listRef.current?.querySelectorAll<HTMLButtonElement>("[data-auth-option]") ?? [],
      );
      if (options.length === 0) {
        return;
      }

      event.preventDefault();
      const current = options.indexOf(document.activeElement as HTMLButtonElement);
      const next =
        event.key === "ArrowDown"
          ? (current + 1) % options.length
          : (current - 1 + options.length) % options.length;
      options[next].focus();
    };

    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [enabled, listRef]);
}

/**
 * Measures the active step and animates the card to its height, so swapping steps
 * grows or shrinks the card instead of snapping. The first measurement is applied
 * without a transition — animating up from an unset height would slide the card open
 * on every page load.
 */
export function useMorphHeight() {
  const outerRef = useRef<HTMLDivElement>(null);
  const innerRef = useRef<HTMLDivElement>(null);
  const [animated, setAnimated] = useState(false);

  useLayoutEffect(() => {
    const outer = outerRef.current;
    const inner = innerRef.current;
    if (!outer || !inner) {
      return;
    }

    const apply = () => {
      outer.style.height = `${inner.offsetHeight}px`;
    };

    apply();
    const observer = new ResizeObserver(apply);
    observer.observe(inner);
    return () => observer.disconnect();
  }, []);

  useEffect(() => {
    const frame = requestAnimationFrame(() => setAnimated(true));
    return () => cancelAnimationFrame(frame);
  }, []);

  return { outerRef, innerRef, animated };
}
