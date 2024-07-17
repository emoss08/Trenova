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

import { Badge, badgeVariants } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { cn, upperFirst } from "@/lib/utils";
import { StatusChoiceProps } from "@/types";
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
export function StatusBadge({ status }: { status: StatusChoiceProps }) {
  return (
    <Badge variant={status === "Active" ? "active" : "inactive"}>
      {status}
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
  const handleRetry = () => {
    // Handle retry logic here
    console.info("Retry not implemented");
  };

  const handleContactSupport = () => {
    console.info("Contact support not implemented");
  };

  return (
    <Card className="col-span-4 lg:col-span-2">
      <CardContent className="relative p-0">
        <div className="bg-muted/50 border-border m-5 flex h-[40vh] flex-col items-center justify-center rounded-md border">
          <FontAwesomeIcon
            icon={faTriangleExclamation}
            className="mb-2"
            size="3x"
          />
          <h3 className="text-foreground text-xl font-semibold">
            Well, this is embarrassing...
          </h3>
          <p className="text-muted-foreground text-sm">
            {message || "There was an error loading the data."}
          </p>
          <div className="mt-5 flex space-x-4">
            <Button variant="default" onClick={handleRetry}>
              Retry
            </Button>
            <Button variant="outline" onClick={handleContactSupport}>
              Contact Support
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
