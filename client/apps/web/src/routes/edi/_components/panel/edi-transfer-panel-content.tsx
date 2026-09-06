"use no memo";

import type { EDITransferRow } from "@/lib/graphql/edi-table";
import type { ReactNode } from "react";

type EDITransferPanelContentProps = {
  transfer: EDITransferRow;
  children: ReactNode;
};

export function EDITransferPanelContent({ transfer, children }: EDITransferPanelContentProps) {
  return (
    <div className="flex min-h-0 flex-col gap-4">
      {transfer.rejectionReason || transfer.failureReason ? (
        <div className="border-destructive/30 rounded-md border p-3 text-sm">
          {transfer.rejectionReason || transfer.failureReason}
        </div>
      ) : null}
      {children}
    </div>
  );
}
