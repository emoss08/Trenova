/**
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
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
