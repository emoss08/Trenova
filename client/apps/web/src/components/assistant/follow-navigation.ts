import { isAppPath } from "@/lib/app-path";
import type { AssistantArtifactEvent } from "@/types/assistant";
import { useEffect } from "react";
import { useNavigate } from "react-router";

/*
 * "Take me there" moves the app once, when the assistant's navigation
 * arrives live, and never again.
 *
 * A turn's events can reach this tab more than once: a reply rejoined after
 * a reload replays from its start, and a conversation carried from the Desk
 * to the floating assistant is followed by a second reader. Each would move
 * the person back to where the assistant sent them minutes ago, over
 * wherever they had gone since. So a navigation is claimed by its id for the
 * life of the tab, and only the first reader to see it follows it. History
 * never navigates at all: it is read from the saved artifacts, not from these
 * events.
 */

const STORAGE_KEY = "trenova-assistant-followed-navigation";
const MAX_REMEMBERED = 50;

/** Kept as well as the session copy, so a tab whose storage is blocked still follows once. */
const followedThisPage = new Set<string>();

function readFollowed(): string[] {
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY);
    const parsed: unknown = raw ? JSON.parse(raw) : [];
    return Array.isArray(parsed)
      ? parsed.filter((entry): entry is string => typeof entry === "string")
      : [];
  } catch {
    return [];
  }
}

/** Marks a navigation as followed; false when this tab already followed it. */
export function claimNavigation(id: string): boolean {
  if (followedThisPage.has(id)) {
    return false;
  }
  followedThisPage.add(id);

  const followed = readFollowed();
  if (followed.includes(id)) {
    return false;
  }
  try {
    sessionStorage.setItem(STORAGE_KEY, JSON.stringify([...followed, id].slice(-MAX_REMEMBERED)));
  } catch {
    // Blocked storage leaves the in-page record, which is enough for this tab.
  }

  return true;
}

/**
 * The path to follow from what a turn has announced: the newest navigation
 * not yet followed. Every unfollowed one is claimed, so a turn that moved
 * twice lands on the last place and does not replay the first later.
 */
export function nextNavigation(
  artifacts: readonly AssistantArtifactEvent[],
  claim: (id: string) => boolean,
): string | null {
  let target: string | null = null;
  for (const artifact of artifacts) {
    if (artifact.kind !== "navigation" || !isAppPath(artifact.path)) {
      continue;
    }
    if (claim(artifact.id)) {
      target = artifact.path;
    }
  }

  return target;
}

/**
 * Follows the assistant's navigation as it arrives. `onFollow` runs first,
 * so a surface that will not survive the move (the Desk) can hand the
 * conversation to one that will before the page changes under it.
 */
export function useFollowNavigation(
  artifacts: readonly AssistantArtifactEvent[],
  onFollow?: () => void,
) {
  const navigate = useNavigate();

  useEffect(() => {
    const target = nextNavigation(artifacts, claimNavigation);
    if (target === null) {
      return;
    }
    onFollow?.();
    void navigate(target);
  }, [artifacts, navigate, onFollow]);
}
