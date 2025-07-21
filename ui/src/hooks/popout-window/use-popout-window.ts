import { useCallback, useEffect, useRef, useState } from "react";
import { getPopoutWindowManager, PopoutWindowOptions } from "./popout-window";

type UsePopoutWindowOptions = {
  onReady?: (windowId: string) => void;
  onClose?: (windowId: string) => void;
  onError?: (error: Error, windowId?: string) => void;
  onFocus?: (windowId: string) => void;
  onBlur?: (windowId: string) => void;
  onStateChange?: (windowId: string, state: any) => void;
};

export function usePopoutWindow(options?: UsePopoutWindowOptions) {
  const [isPopout, setIsPopout] = useState(false);
  const [popoutId, setPopoutId] = useState<string | null>(null);
  const [activeWindows, setActiveWindows] = useState<string[]>([]);
  const managerRef = useRef(getPopoutWindowManager());

  // Setup event handlers
  useEffect(() => {
    const manager = managerRef.current;

    if (options?.onReady) manager.on("onReady", options.onReady);
    if (options?.onClose) manager.on("onClose", options.onClose);
    if (options?.onError) manager.on("onError", options.onError);
    if (options?.onFocus) manager.on("onFocus", options.onFocus);
    if (options?.onBlur) manager.on("onBlur", options.onBlur);
    if (options?.onStateChange)
      manager.on("onStateChange", options.onStateChange);

    // Subscribe to window changes
    const handleWindowsChange = (windowIds: string[]) => {
      setActiveWindows(windowIds);
    };

    manager.on("onWindowsChange", handleWindowsChange);

    // Get initial windows
    setActiveWindows(manager.getActiveWindows());

    return () => {
      // No cleanup needed - events are managed by the singleton
    };
  }, [options]);

  // Handle popout window mode
  useEffect(() => {
    const queryParams = new URLSearchParams(window.location.search);
    const id = queryParams.get("popoutId");
    if (id) {
      setIsPopout(true);
      setPopoutId(id);

      // Send ready message to parent
      window.opener?.postMessage(
        { type: "popout-ready", popoutId: id },
        window.location.origin,
      );

      // Handle window events
      const handleBeforeUnload = () => {
        window.opener?.postMessage(
          { type: "popout-closed", popoutId: id },
          window.location.origin,
        );
      };

      const handleFocus = () => {
        window.opener?.postMessage(
          { type: "popout-focus", popoutId: id },
          window.location.origin,
        );
      };

      const handleBlur = () => {
        window.opener?.postMessage(
          { type: "popout-blur", popoutId: id },
          window.location.origin,
        );
      };

      window.addEventListener("beforeunload", handleBeforeUnload);
      window.addEventListener("focus", handleFocus);
      window.addEventListener("blur", handleBlur);

      return () => {
        window.removeEventListener("beforeunload", handleBeforeUnload);
        window.removeEventListener("focus", handleFocus);
        window.removeEventListener("blur", handleBlur);
      };
    }
  }, []);

  const openPopout = useCallback(
    (
      path: string,
      queryParams?: Record<string, string | number | boolean>,
      options?: PopoutWindowOptions,
    ) => {
      const manager = managerRef.current;
      return manager.openWindow(path, queryParams, options);
    },
    [],
  );

  const closePopout = useCallback(
    (windowId?: string) => {
      const manager = managerRef.current;
      if (windowId) {
        manager.closeWindow(windowId);
      } else if (isPopout && popoutId) {
        window.close();
      }
    },
    [isPopout, popoutId],
  );

  const closeAllPopouts = useCallback(() => {
    const manager = managerRef.current;
    manager.closeAllWindows();
  }, []);

  const focusPopout = useCallback((windowId: string) => {
    const manager = managerRef.current;
    manager.focusWindow(windowId);
  }, []);

  const sendMessage = useCallback(
    (windowId: string, type: string, data?: unknown) => {
      const manager = managerRef.current;
      manager.sendMessage(windowId, type as any, data);
    },
    [],
  );

  const broadcastMessage = useCallback((type: string, data?: unknown) => {
    const manager = managerRef.current;
    manager.broadcastMessage(type as any, data);
  }, []);

  const getWindowState = useCallback((windowId: string) => {
    const manager = managerRef.current;
    return manager.getWindowState(windowId);
  }, []);

  return {
    // State
    isPopout,
    popoutId,
    activeWindows,
    hasOpenWindows: activeWindows.length > 0,

    // Actions
    openPopout,
    closePopout,
    closeAllPopouts,
    focusPopout,
    sendMessage,
    broadcastMessage,
    getWindowState,
  };
}
