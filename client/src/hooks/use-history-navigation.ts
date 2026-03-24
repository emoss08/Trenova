import { useCallback, useEffect, useRef, useState } from "react";
import { useLocation, useNavigate, useNavigationType } from "react-router";

interface HistoryStackState {
  entries: string[];
  index: number;
}

function getLocationId(location: ReturnType<typeof useLocation>): string {
  return `${location.key}:${location.pathname}${location.search}${location.hash}`;
}

export function useHistoryNavigation() {
  const location = useLocation();
  const navigate = useNavigate();
  const navigationType = useNavigationType();

  const initialId = getLocationId(location);
  const [historyState, setHistoryState] = useState<HistoryStackState>({
    entries: [initialId],
    index: 0,
  });
  const lastProcessedTokenRef = useRef<string | null>(null);

  useEffect(() => {
    const locationId = getLocationId(location);
    const token = `${navigationType}:${locationId}`;

    // Avoid duplicate transitions in StrictMode re-renders/effects.
    if (lastProcessedTokenRef.current === token) {
      return;
    }
    lastProcessedTokenRef.current = token;

    setHistoryState((currentState) => {
      if (currentState.entries.length === 0) {
        return { entries: [locationId], index: 0 };
      }

      if (navigationType === "PUSH") {
        const nextEntries = currentState.entries.slice(0, currentState.index + 1);
        nextEntries.push(locationId);
        return { entries: nextEntries, index: nextEntries.length - 1 };
      }

      if (navigationType === "REPLACE") {
        const nextEntries = [...currentState.entries];
        nextEntries[currentState.index] = locationId;
        return { entries: nextEntries, index: currentState.index };
      }

      // POP: move within known entries when possible.
      if (currentState.entries[currentState.index - 1] === locationId) {
        return { entries: currentState.entries, index: currentState.index - 1 };
      }

      if (currentState.entries[currentState.index + 1] === locationId) {
        return { entries: currentState.entries, index: currentState.index + 1 };
      }

      const existingIndex = currentState.entries.lastIndexOf(locationId);
      if (existingIndex >= 0) {
        return { entries: currentState.entries, index: existingIndex };
      }

      // If location is outside tracked stack, append so controls recover gracefully.
      return {
        entries: [...currentState.entries, locationId],
        index: currentState.entries.length,
      };
    });
  }, [location, navigationType]);

  const canGoBack = historyState.index > 0;
  const canGoForward = historyState.index < historyState.entries.length - 1;

  const goBack = useCallback(() => {
    if (!canGoBack) {
      return;
    }
    void navigate(-1);
  }, [canGoBack, navigate]);

  const goForward = useCallback(() => {
    if (!canGoForward) {
      return;
    }
    void navigate(1);
  }, [canGoForward, navigate]);

  return {
    canGoBack,
    canGoForward,
    goBack,
    goForward,
  };
}
