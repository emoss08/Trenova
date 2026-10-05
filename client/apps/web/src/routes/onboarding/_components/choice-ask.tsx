import { numberKeyIndex } from "@/lib/dom";
import { useEffect, useRef, useState, type ReactNode } from "react";
import { Kbd } from "./keyboard-hint";

const PICK_DELAY_MS = 420;

export type ChoiceItem<T> = {
  value: T;
  title: string;
  description: string;
  extra?: ReactNode;
};

/**
 * A turn answered by picking one card: click it or press its number. The pick stays
 * on screen for a beat, the other cards fading back, before the answer is sent.
 */
export function ChoiceAsk<T extends string | boolean>({
  items,
  value,
  label,
  error,
  onPick,
}: {
  items: readonly ChoiceItem<T>[];
  value: T | undefined;
  label: string;
  error?: string;
  onPick: (value: T) => Promise<boolean>;
}) {
  const [picked, setPicked] = useState<{ value: T } | null>(null);
  const timer = useRef<number | null>(null);

  useEffect(() => {
    return () => {
      if (timer.current !== null) {
        window.clearTimeout(timer.current);
      }
    };
  }, []);

  const pick = (next: T) => {
    if (picked !== null) {
      return;
    }
    setPicked({ value: next });
    timer.current = window.setTimeout(() => {
      timer.current = null;
      void onPick(next).then((advanced) => {
        if (!advanced) {
          setPicked(null);
        }
      });
    }, PICK_DELAY_MS);
  };

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      const index = numberKeyIndex(event, items.length);
      if (index !== -1) {
        event.preventDefault();
        pick(items[index].value);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });

  const selected = picked ? picked.value : value;

  return (
    <div className="nv-opts" role="group" aria-label={label} data-chosen={picked !== null}>
      {error ? (
        <p className="nv-err" role="alert">
          {error}
        </p>
      ) : null}
      {items.map((item, index) => (
        <button
          key={String(item.value)}
          type="button"
          className="nv-op"
          aria-pressed={selected === item.value}
          style={{ animationDelay: `${index * 60}ms` }}
          onClick={() => pick(item.value)}
        >
          <b>{item.title}</b>
          <Kbd>{index + 1}</Kbd>
          <span className="nv-d">{item.description}</span>
          {item.extra}
        </button>
      ))}
    </div>
  );
}
