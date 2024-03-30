/*
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

import { Badge, badgeVariants } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { cn, upperFirst } from "@/lib/utils";
import { EquipmentStatus } from "@/types/equipment";
import { IconProp } from "@fortawesome/fontawesome-svg-core";
import {
  faPlus,
  faTriangleExclamation,
} from "@fortawesome/pro-solid-svg-icons";

import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { VariantProps } from "class-variance-authority";

/**
 * Component that displays a message when no data is found.
 * @param message The message to display
 * @param name The name of the data
 * @param Icon The icon to display
 * @param onButtonClick The callback to call when the button is clicked
 * @returns A component that displays a message when no data is found
 */
export function DataNotFound({
  message,
  name,
  icon,
  onButtonClick,
  className,
}: {
  message: string;
  name: string;
  icon: IconProp;
  onButtonClick?: () => void;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "my-10 text-center flex grow flex-col items-center justify-center",
        className,
      )}
    >
      <FontAwesomeIcon icon={icon} className="text-foreground size-10" />
      <h3 className="mt-4 text-lg font-semibold">
        No {upperFirst(name)} added
      </h3>
      <p className="text-muted-foreground mt-2 text-sm">{message}</p>
      <Button
        className="mt-3"
        type="button"
        size="sm"
        variant="expandIcon"
        iconPlacement="left"
        icon={faPlus}
        onClick={onButtonClick}
      >
        Add {upperFirst(name)}
      </Button>
    </div>
  );
}

/**
 * Status badge that can be used to display the status of a record.
 * @param status The status of the record
 * @returns A badge with the status of the record
 */
export function StatusBadge({ status }: { status: string }) {
  return (
    <Badge variant={status === "A" ? "active" : "inactive"}>
      {status === "A" ? "Active" : "Inactive"}
    </Badge>
  );
}

/**
 * Status badge that can be used to display the status of a boolean value.
 * @param status The status of the boolean value
 * @returns A badge with the status of the boolean value
 */
export function BoolStatusBadge({ status }: { status: boolean }) {
  return (
    <Badge variant={status ? "active" : "inactive"}>
      {status ? "Yes" : "No"}
    </Badge>
  );
}

type StatusAttrProps = {
  variant: VariantProps<typeof badgeVariants>["variant"];
  text: string;
};
/**
 * Status badge that can be used to display the status of equipment. (e.g. Trailer & Tractor statuses)
 * @param status The status of the equipment
 * @returns A badge with the status of the equipment
 */
export function EquipmentStatusBadge({ status }: { status: EquipmentStatus }) {
  const statusAttributes: Record<EquipmentStatus, StatusAttrProps> = {
    Available: {
      variant: "active",
      text: "Available",
    },
    OutOfService: {
      variant: "inactive",
      text: "Out of Service",
    },
    AtMaintenance: {
      variant: "purple",
      text: "At Maintenance",
    },
    Sold: {
      variant: "info",
      text: "Sold",
    },
    Lost: {
      variant: "warning",
      text: "Lost",
    },
  };

  return (
    <Badge variant={statusAttributes[status].variant}>
      {statusAttributes[status].text}
    </Badge>
  );
}

export function ErrorLoadingData({ message }: { message?: string }) {
  return (
    <div className="text-center">
      <FontAwesomeIcon
        icon={faTriangleExclamation}
        className="text-accent-foreground mx-auto size-10"
      />
      <p className="text-accent-foreground mt-2 font-semibold">
        Well, this is embarrassing...
      </p>
      <p className="text-muted-foreground mt-2">
        {message || "There was an error loading the data."}
      </p>
    </div>
  );
}
