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

import { truncateText } from "@/lib/utils";
import { CaretSortIcon } from "@radix-ui/react-icons";

// TODO: Implement this when workflows are available
export function WorkflowPlaceholder() {
  return (
    <div className="group col-span-full flex w-full select-none items-center gap-x-4 rounded-lg border border-dashed border-blue-200 bg-blue-200 p-1 px-4 hover:cursor-pointer dark:border-blue-500 dark:bg-blue-600/20 dark:text-blue-400">
      <div className="flex flex-1 flex-col">
        <p className="text-sm text-foreground dark:text-blue-100">Workflow</p>
        <h2 className="text-lg truncate font-semibold leading-7 text-blue-600 dark:text-blue-400">
          {truncateText("Operation Management", 20)}
        </h2>
      </div>
      <div className="ml-auto flex items-center justify-center">
        <CaretSortIcon className="size-6 text-blue-600 dark:text-blue-400" />
      </div>
    </div>
  );
}
