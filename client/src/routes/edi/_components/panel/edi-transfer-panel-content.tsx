"use no memo";

import type { EDITransfer } from "@/types/edi";
import type { ReactNode } from "react";

type EDITransferPanelContentProps = {
  transfer: EDITransfer;
  children: ReactNode;
};

export function EDITransferPanelContent({ transfer, children }: EDITransferPanelContentProps) {
  return (
    <div className="flex min-h-0 flex-col gap-4">
      {transfer.rejectionReason || transfer.failureReason ? (
        <div className="rounded-md border border-destructive/30 p-3 text-sm">
          {transfer.rejectionReason || transfer.failureReason}
        </div>
      ) : null}
      {children}
    </div>
  );
}
