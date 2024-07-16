import { useEffect, useState } from "react";
import { getPopoutWindowManager, PopoutWindowOptions } from "./popout-window";

export function usePopoutWindow() {
  const [isPopout, setIsPopout] = useState(false);
  const [popoutId, setPopoutId] = useState<string | null>(null);

  useEffect(() => {
    const queryParams = new URLSearchParams(window.location.search);
    const id = queryParams.get("popoutId");
    if (id) {
      setIsPopout(true);
      setPopoutId(id);

      const handleBeforeUnload = () => {
        window.opener?.postMessage(
          { type: "popout-closed", popoutId: id },
          "*",
        );
      };

      window.addEventListener("beforeunload", handleBeforeUnload);
      return () => {
        window.removeEventListener("beforeunload", handleBeforeUnload);
      };
    }
  }, []);

  const openPopout = (
    path: string,
    queryParams?: Record<string, string | number | boolean>,
    options?: PopoutWindowOptions,
  ) => {
    const manager = getPopoutWindowManager();
    return manager.openWindow(path, queryParams, options);
  };

  const closePopout = () => {
    if (isPopout && popoutId) {
      window.close();
    }
  };

  return { isPopout, popoutId, openPopout, closePopout };
}
