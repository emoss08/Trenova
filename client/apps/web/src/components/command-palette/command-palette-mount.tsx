import { whenIdle } from "@/lib/when-idle";
import { useCommandPaletteStore } from "@/stores/command-palette-store";
import { useHotkey } from "@tanstack/react-hotkeys";
import { lazy, Suspense, useEffect, useState } from "react";
import { CommandPaletteSkeleton } from "./command-palette-skeleton";

const CommandPalette = lazy(() =>
  import("./command-palette").then((module) => ({ default: module.CommandPalette })),
);

let preloadPromise: Promise<unknown> | null = null;

/**
 * Pulls the palette chunk in ahead of the first open. Idempotent — repeat calls
 * reuse the in-flight import.
 */
export function preloadCommandPalette(): Promise<unknown> {
  preloadPromise ??= import("./command-palette");
  return preloadPromise;
}

/**
 * Always-mounted shell for the command palette. It owns the Mod+K binding and
 * nothing else, so cmdk, the global-search client and the Google Maps preview
 * stay out of the entry chunk; the palette itself is fetched when the app goes
 * idle, or on the first intent to open it — whichever comes first.
 */
export function CommandPaletteMount() {
  const open = useCommandPaletteStore((state) => state.open);
  const toggleOpen = useCommandPaletteStore((state) => state.toggleOpen);
  const [mounted, setMounted] = useState(false);

  useHotkey(
    "Mod+K",
    () => {
      void preloadCommandPalette();
      toggleOpen();
    },
    {
      ignoreInputs: true,
      preventDefault: true,
    },
  );

  // Adjusting state during render (rather than in an effect) so the palette
  // mounts in the same commit the store opens it — an effect would paint one
  // frame of nothing over the page first.
  if (open && !mounted) {
    setMounted(true);
  }

  useEffect(() => {
    if (mounted) return;
    return whenIdle(() => void preloadCommandPalette());
  }, [mounted]);

  if (!mounted) return null;

  return (
    <Suspense fallback={open ? <CommandPaletteSkeleton /> : null}>
      <CommandPalette />
    </Suspense>
  );
}
