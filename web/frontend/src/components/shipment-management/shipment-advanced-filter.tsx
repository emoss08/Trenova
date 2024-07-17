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
