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

import { InboxIcon } from "lucide-react";

export function ArchiveMenuContent() {
  return (
    <div className="flex h-80 w-full items-center justify-center p-4">
      <div className="flex flex-col items-center justify-center gap-y-3">
        <div className="bg-accent flex size-10 items-center justify-center rounded-full">
          <InboxIcon className="text-muted-foreground" />
        </div>
        <p className="text-muted-foreground select-none text-center text-sm">
          Nothing appears to be here
        </p>
      </div>
    </div>
  );
}
