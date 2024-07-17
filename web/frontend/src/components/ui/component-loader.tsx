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

import { cn } from "@/lib/utils";
import { faSpinner } from "@fortawesome/pro-duotone-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";

export function ComponentLoader({ className }: { className?: string }) {
  return (
    <div
      className={cn("flex flex-col items-center justify-center p-2", className)}
    >
      <FontAwesomeIcon
        icon={faSpinner}
        size="1x"
        className="text-primary motion-safe:animate-spin"
      />
      <p className="text-foreground mt-2 text-sm">Loading data...</p>
      <p className="text-muted-foreground mt-2 text-sm">
        If this takes too long, please refresh the page.
      </p>
    </div>
  );
}
