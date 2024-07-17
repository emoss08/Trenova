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

import { truncateText } from "@/lib/utils";
import { CaretSortIcon } from "@radix-ui/react-icons";

// TODO: Implement this when workflows are available
export function WorkflowPlaceholder() {
  return (
    <div className="group col-span-full flex w-full select-none items-center gap-x-4 rounded-lg border border-dashed border-blue-200 bg-blue-200 p-1 px-4 hover:cursor-pointer dark:border-blue-500 dark:bg-blue-600/20 dark:text-blue-400">
      <div className="flex flex-1 flex-col">
        <p className="text-foreground text-sm dark:text-blue-100">Workflow</p>
        <h2 className="truncate text-lg font-semibold leading-7 text-blue-600 dark:text-blue-400">
          {truncateText("Operation Management", 20)}
        </h2>
      </div>
      <div className="ml-auto flex items-center justify-center">
        <CaretSortIcon className="size-6 text-blue-600 dark:text-blue-400" />
      </div>
    </div>
  );
}
