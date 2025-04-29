import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { EntityRedirectLink } from "@/components/ui/link";
import { cn } from "@/lib/utils";
import { type AdditionalCharge } from "@/types/shipment";
import {
  faPencil,
  faTrash,
  faTriangleExclamation,
} from "@fortawesome/pro-solid-svg-icons";
import { memo, type CSSProperties } from "react";

function AdditionalChargeRow({
  index,
  additionalCharge,
  style,
  isLast,
  isDuplicate,
  onEdit,
  onDelete,
}: {
  index: number;
  additionalCharge: AdditionalCharge;
  style: CSSProperties;
  isLast: boolean;
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

  // Create a memoization key based on the additional charge data
  const memoKey = `${additionalCharge.accessorialChargeId}-${additionalCharge.unit}-${additionalCharge.method}-${additionalCharge.amount}`;

  return (
    <div
      key={memoKey}
      className={cn(
        "grid grid-cols-10 gap-4 px-2 items-center text-sm ",
        !isLast && "border-b border-border",
        isLast && "rounded-b-md",
        isDuplicate && "bg-yellow-500/20 border-yellow-500/30 border",
      )}
      style={style}
    >
      <div className="flex col-span-4 gap-2">
        <EntityRedirectLink
          entityId={additionalCharge.accessorialCharge.id}
          baseUrl="/billing/configurations/accessorial-charges"
          modelOpen
        >
          {additionalCharge.accessorialCharge.code}
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
      <div className="col-span-2 text-left">{additionalCharge.unit}</div>
      <div className="col-span-2 text-left">{additionalCharge.amount}</div>
      <div className="col-span-2 flex gap-0.5 justify-end">
        <Button
          type="button"
          variant="ghost"
          size="xs"
          title="Edit Additional Charge"
          onClick={(e) => {
            e.preventDefault();
            onEdit(index);
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
            onDelete(index);
          }}
        >
          <Icon icon={faTrash} className="size-3" />
        </Button>
      </div>
    </div>
  );
}

AdditionalChargeRow.displayName = "AdditionalChargeRow";

export const MemoizedAdditionalChargeRow = memo(
  AdditionalChargeRow,
  (prevProps, nextProps) => {
    const prevAdditionalCharge = prevProps.additionalCharge;
    const nextAdditionalCharge = nextProps.additionalCharge;

    return (
      prevProps.isLast === nextProps.isLast &&
      prevProps.isDuplicate === nextProps.isDuplicate &&
      prevAdditionalCharge.accessorialChargeId ===
        nextAdditionalCharge.accessorialChargeId &&
      prevAdditionalCharge.unit === nextAdditionalCharge.unit &&
      prevAdditionalCharge.amount === nextAdditionalCharge.amount
    );
  },
);
