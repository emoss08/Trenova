import { buildId, deployedBuildId } from "@/lib/build-id";
import { reloadIsSafe } from "@/lib/reload-guards";
import { settleOnNewestRelease } from "@/lib/service-worker";

const RELOADED_FOR_BUILD = "trenova:reloaded-for-build";

type RecoveryTarget = Pick<Window, "addEventListener"> & { location: Pick<Location, "reload"> };
type RecoveryStorage = Pick<Storage, "getItem" | "setItem">;

type ChunkRecoveryOptions = {
  target?: RecoveryTarget;
  storage?: () => RecoveryStorage;
  /** The build this page runs. */
  currentBuild?: () => string;
  /** The build the server deploys now; null when it cannot be asked. */
  deployedBuild?: () => Promise<string | null>;
  /** Runs before reloading; by default, lets the new release's worker take over first. */
  beforeReload?: () => Promise<void>;
  /** Whether nothing holds work a reload would lose (lib/reload-guards). */
  canReload?: () => boolean;
};

/**
 * A page left open across a release still runs the old build, and the first time it opens a
 * screen it has not loaded yet it asks for a file the release deleted. Vite reports that as
 * `vite:preloadError`. The failure is left to show as it would anyway; then, if the server
 * confirms a different build is deployed, the page reloads into it.
 *
 * Nothing reloads when the failure is not a release — offline, a server error, the same
 * build still deployed — nor while the page holds work a reload would lose (an unsaved form,
 * an open popout); the next change of page then moves onto the release (lib/service-worker).
 * And only once per build: if the same build fails again after reloading, reloading would
 * loop. Without sessionStorage (some private modes) there is no way to tell, so nothing
 * reloads.
 */
export function installChunkRecovery({
  target = window,
  storage = () => window.sessionStorage,
  currentBuild = buildId,
  deployedBuild = deployedBuildId,
  beforeReload = () => settleOnNewestRelease(),
  canReload = reloadIsSafe,
}: ChunkRecoveryOptions = {}): void {
  let checking = false;

  const recover = async () => {
    const running = currentBuild();
    const deployed = await deployedBuild();
    if (deployed === null || deployed === running || !canReload()) return;
    try {
      const store = storage();
      if (store.getItem(RELOADED_FOR_BUILD) === running) return;
      store.setItem(RELOADED_FOR_BUILD, running);
    } catch {
      return;
    }
    // Waiting for the new release only makes the reload land on it; the reload is the
    // recovery and happens whatever the wait came to.
    await beforeReload().catch(() => undefined);
    target.location.reload();
  };

  target.addEventListener("vite:preloadError", () => {
    if (checking || !canReload()) return;
    checking = true;
    void recover().finally(() => {
      checking = false;
    });
  });
}
