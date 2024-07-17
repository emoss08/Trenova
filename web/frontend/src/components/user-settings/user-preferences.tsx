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
