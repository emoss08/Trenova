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

import { ShipmentSearchForm } from "@/types/shipment";
import {
    CaretSortIcon,
    DownloadIcon,
    MagnifyingGlassIcon,
    PlusIcon,
} from "@radix-ui/react-icons";
import { useFormContext } from "react-hook-form";
import { InputField } from "../common/fields/input";
import { Button } from "../ui/button";

export function ShipmentToolbar() {
  return (
    <div className="flex justify-between">
      <ShipmentSearch />
      <div className="space-x-2">
        <Button variant="outline" size="sm">
          <DownloadIcon className="mr-1 size-4" />
          Export
        </Button>
        <Button variant="outline" size="sm">
          <CaretSortIcon className="mr-1 size-4" />
          Filter
        </Button>
        <Button variant="outline" size="sm">
          <PlusIcon className="mr-1 size-4" />
          New Shipment
        </Button>
      </div>
    </div>
  );
}

function ShipmentSearch() {
  const { control } = useFormContext<ShipmentSearchForm>();

  return (
    <div className="relative">
      <InputField
        name="searchQuery"
        control={control}
        placeholder="Search Shipments..."
        icon={<MagnifyingGlassIcon className="text-muted-foreground size-4" />}
      />
    </div>
  );
}
