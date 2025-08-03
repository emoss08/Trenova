/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { EntityRedirectLink } from "@/components/ui/link";
import type { AccessorialChargeSchema } from "@/lib/schemas/accessorial-charge-schema";
import type { AdditionalChargeSchema } from "@/lib/schemas/additional-charge-schema";
import { cn } from "@/lib/utils";
import {
  faPencil,
  faTrash,
  faTriangleExclamation,
} from "@fortawesome/pro-solid-svg-icons";
import React, { memo } from "react";

export function AdditionalChargeRow({
  index,
  additionalCharge,
  isDuplicate,
  onEdit,
  onDelete,
}: {
  index: number;
  additionalCharge: AdditionalChargeSchema;
  isDuplicate?: boolean;
  onEdit: (index: number) => void;
  onDelete: (index: number) => void;
}) {
  if (!additionalCharge.accessorialCharge) {
    return (
      <div className="col-span-12 text-center text-sm text-muted-foreground">
        Unable to load accessorial charge
      </div>
    );
  }

  return (
    <AdditionalChargeInner isDuplicate={isDuplicate}>
      <AdditionalChargeContent>
        <AdditionalChargeAccessorialCharge
          accessorialCharge={additionalCharge.accessorialCharge}
          isDuplicate={isDuplicate}
        />
      </AdditionalChargeContent>
      <AdditionalChargeRowInformation
        unit={additionalCharge.unit}
        amount={additionalCharge.amount}
      />
      <AdditionalChargeAction
        onEdit={() => onEdit(index)}
        onDelete={() => onDelete(index)}
      />
    </AdditionalChargeInner>
  );
}

function AdditionalChargeContent({ children }: { children: React.ReactNode }) {
  return <div className="flex gap-2 col-span-4">{children}</div>;
}

const AdditionalChargeRowInformation = memo(
  function AdditionalChargeRowInformation({
    unit,
    amount,
  }: {
    unit: number;
    amount: number;
  }) {
    return (
      <>
        <div className="col-span-2 text-left">{unit}</div>
        <div className="col-span-2 text-left">{amount}</div>
      </>
    );
  },
);

function AdditionalChargeInner({
  isDuplicate,
  children,
}: {
  children: React.ReactNode;
  isDuplicate?: boolean;
}) {
  return (
    <div
      className={cn(
        "grid grid-cols-10 gap-4 p-2 text-sm",
        isDuplicate && "bg-yellow-500/20 border-yellow-500/30 border",
      )}
    >
      {children}
    </div>
  );
}

function AdditionalChargeAccessorialCharge({
  accessorialCharge,
  isDuplicate,
}: {
  accessorialCharge: AccessorialChargeSchema;
  isDuplicate?: boolean;
}) {
  return (
    <div className="flex col-span-4 gap-2">
      <EntityRedirectLink
        entityId={accessorialCharge.id}
        baseUrl="/billing/configurations/accessorial-charges"
        modelOpen
      >
        {accessorialCharge.code}
      </EntityRedirectLink>
      {isDuplicate && (
        <span
          title="Possible duplicate charge detected"
          className="text-yellow-600"
        >
          <Icon icon={faTriangleExclamation} className="size-3 mb-0.5" />
        </span>
      )}
    </div>
  );
}

function AdditionalChargeAction({
  onEdit,
  onDelete,
}: {
  onEdit: () => void;
  onDelete: () => void;
}) {
  return (
    <div className="col-span-2 flex gap-0.5 justify-end">
      <Button
        type="button"
        variant="ghost"
        size="xs"
        title="Edit Additional Charge"
        onClick={(e) => {
          e.preventDefault();
          onEdit();
        }}
      >
        <Icon icon={faPencil} className="size-3" />
      </Button>

      <Button
        type="button"
        variant="ghost"
        className="hover:bg-red-500/30 text-red-600 hover:text-red-600"
        size="xs"
        title="Delete Additional Charge"
        onClick={(e) => {
          e.preventDefault();
          onDelete();
        }}
      >
        <Icon icon={faTrash} className="size-3" />
      </Button>
    </div>
  );
}
