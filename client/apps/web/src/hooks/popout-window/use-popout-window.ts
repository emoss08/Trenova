import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  getPopoutWindowManager,
  type PopoutWindowOptions,
} from "./popout-window";

type UsePopoutWindowOptions = {
  onReady?: (windowId: string) => void;
  onClose?: (windowId: string) => void;
  onError?: (error: Error, windowId?: string) => void;
  onFocus?: (windowId: string) => void;
  onBlur?: (windowId: string) => void;
  onStateChange?: (windowId: string, state: any) => void;
};

export function usePopoutWindow(options?: UsePopoutWindowOptions) {
  const [activeWindows, setActiveWindows] = useState<string[]>([]);
  const managerRef = useRef(getPopoutWindowManager());
  const popoutId = useMemo(() => {
    const queryParams = new URLSearchParams(window.location.search);
    return queryParams.get("popoutId");
  }, []);
  const isPopout = Boolean(popoutId);

  useEffect(() => {
    const manager = managerRef.current;

    if (options?.onReady) manager.on("onReady", options.onReady);
    if (options?.onClose) manager.on("onClose", options.onClose);
    if (options?.onError) manager.on("onError", options.onError);
    if (options?.onFocus) manager.on("onFocus", options.onFocus);
    if (options?.onBlur) manager.on("onBlur", options.onBlur);
    if (options?.onStateChange)
      manager.on("onStateChange", options.onStateChange);

    const handleWindowsChange = (windowIds: string[]) => {
      setActiveWindows(windowIds);
    };

    manager.on("onWindowsChange", handleWindowsChange);

    setActiveWindows(manager.getActiveWindows());

    return () => {};
  }, [options]);

  useEffect(() => {
    if (popoutId) {
      window.opener?.postMessage(
        { type: "popout-ready", popoutId },
        window.location.origin,
      );

      const handleBeforeUnload = () => {
        window.opener?.postMessage(
          { type: "popout-closed", popoutId },
          window.location.origin,
        );
      };

      const handleFocus = () => {
        window.opener?.postMessage(
          { type: "popout-focus", popoutId },
          window.location.origin,
        );
      };

      const handleBlur = () => {
        window.opener?.postMessage(
          { type: "popout-blur", popoutId },
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
  }, [popoutId]);

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
    isPopout,
    popoutId,
    activeWindows,
    hasOpenWindows: activeWindows.length > 0,
    openPopout,
    closePopout,
    closeAllPopouts,
    focusPopout,
    sendMessage,
    broadcastMessage,
    getWindowState,
  };
}
