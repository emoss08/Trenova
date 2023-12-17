/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
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
import React from "react";
import { Badge } from "@/components/ui/badge";
import { EquipmentStatus } from "@/types/equipment";
import { IconProps } from "@radix-ui/react-icons/dist/types";
import { PlusIcon } from "@radix-ui/react-icons";
import { upperFirst } from "@/lib/utils";
import { Button } from "@/components/ui/button";

type DataNotFoundProps = {
  message: string;
  name: string;
  Icon: React.ForwardRefExoticComponent<
    IconProps & React.RefAttributes<SVGSVGElement>
  >;
  onButtonClick?: () => void;
};

export function DataNotFound({
  message,
  name,
  Icon,
  onButtonClick,
}: DataNotFoundProps) {
  return (
    <div className="text-center my-10">
      <Icon className="mx-auto h-10 w-10 text-foreground" />
      <h3 className="mt-2 text-sm font-semibold text-gray-900">
        No {upperFirst(name)}
      </h3>
      <p className="mt-1 text-sm text-gray-500">{message}</p>
      <div className="mt-3">
        <Button
          className="mt-3"
          type="button"
          size="sm"
          onClick={onButtonClick}
        >
          <PlusIcon className="-ml-0.5 mr-1.5 h-5 w-5" aria-hidden="true" />
          Add {upperFirst(name)}
        </Button>
      </div>
    </div>
  );
}

export function StatusBadge({ status }: { status: string }) {
  return (
    <Badge variant={status === "A" ? "default" : "destructive"}>
      {status === "A" ? "Active" : "Inactive"}
    </Badge>
  );
}

export function BoolStatusBadge({ status }: { status: boolean }) {
  return (
    <Badge variant={status ? "default" : "destructive"}>
      {status ? "Yes" : "No"}
    </Badge>
  );
}

/**
 * Status badge that can be used to display the status of equipment. (e.g. Trailer & Tractor statuses)
 * @param status The status of the equipment
 * @returns A badge with the status of the equipment
 */
export function EquipmentStatusBadge({ status }: { status: EquipmentStatus }) {
  const mapToStatus = {
    A: "Available",
    OOS: "Out of Service",
    AM: "At Maintenance",
    S: "Sold",
    L: "Lost",
  };

  return (
    <Badge variant={status === "A" ? "default" : "destructive"}>
      {mapToStatus[status]}
    </Badge>
  );
}
