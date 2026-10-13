import { lazyComponent } from "@/lib/lazy-component";
import { whenIdle } from "@/lib/when-idle";
import { useAppDialogsStore } from "@/stores/app-dialogs-store";
import { useCommandPaletteStore } from "@/stores/command-palette-store";
import { useHotkey } from "@tanstack/react-hotkeys";
import { Suspense, useEffect, useState } from "react";
import { CommandPaletteSkeleton } from "./command-palette-skeleton";

const Palette = lazyComponent(
  () => import("./command-palette"),
  (module) => module.CommandPalette,
);

/**
 * Pulls the palette chunk in ahead of the first open. Idempotent — repeat calls
 * reuse the in-flight import. A read that fails (offline, or a release removed the
 * file this tab was built with) is forgotten, so opening the palette asks again.
 */
export function preloadCommandPalette(): void {
  void Palette.preload().catch(() => undefined);
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
      // Settings has its own ⌘K, which finds a setting; the palette stays out of its way.
      if (useAppDialogsStore.getState().active === "settings") return;
      preloadCommandPalette();
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
    return whenIdle(preloadCommandPalette);
  }, [mounted]);

  if (!mounted) return null;

  return (
    <Suspense fallback={open ? <CommandPaletteSkeleton /> : null}>
      <Palette.Component />
    </Suspense>
  );
}
