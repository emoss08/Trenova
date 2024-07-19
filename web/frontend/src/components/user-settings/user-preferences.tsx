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

import { ScrollArea } from "../ui/scroll-area";
import { ColorBlindSwitcher } from "./appearance/color-mode-switcher";
import { ThemeSwitcher } from "./appearance/theme-switcher";

export default function UserPreferences() {
  return (
    <>
      <div className="space-y-3">
        <div className="sticky top-0 z-20 mb-6 flex items-center gap-x-2">
          <h2 className="shrink-0 text-sm" id="personal-information">
            Preferences
          </h2>
          <p className="text-xs text-muted-foreground">
            Adjust your interface settings to suit your individual needs.
          </p>
        </div>
      </div>
      <ScrollArea className="-mr-4 h-[550px]">
        <ThemeSwitcher />
        <ColorBlindSwitcher />
      </ScrollArea>
    </>
  );
}
