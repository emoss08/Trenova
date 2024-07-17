/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
 */

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
