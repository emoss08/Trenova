import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { Document } from "@/types/document";
import type { Shipment, ShipmentBillingRequirement } from "@/types/shipment";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";

type DocumentRow = {
  id: string;
  label: string;
  requirement: ShipmentBillingRequirement;
  matchedDocumentCount: number;
};

export type ShipmentDocumentUploadContext = {
  documentTypeId: string;
  documentTypeName: string;
};

function getMatchingDocuments(requirement: ShipmentBillingRequirement, documents: Document[]) {
  const requiredDocumentIds = new Set(requirement.documentIds);

  if (requiredDocumentIds.size > 0) {
    return documents.filter((document) => requiredDocumentIds.has(document.id));
  }

  return documents.filter((document) => document.documentTypeId === requirement.documentTypeId);
}

function getUploadedCountLabel(count: number) {
  return `${count} uploaded`;
}

export function DocumentsBlock({
  shipment,
  onUpload,
}: {
  shipment: Shipment;
  onUpload: (shipment: Shipment, context?: ShipmentDocumentUploadContext) => void;
}) {
  const shipmentId = shipment.id ?? "";
  const hasShipmentId = shipmentId.length > 0;
  const billingReadinessQuery = queries.shipment.billingReadiness(shipmentId);
  const {
    data: billingReadiness,
    isLoading: isBillingReadinessLoading,
    isError: isBillingReadinessError,
  } = useQuery({
    ...billingReadinessQuery,
    enabled: hasShipmentId,
  });

  const {
    data: documents = [],
    isLoading: isDocumentsLoading,
    isError: isDocumentsError,
  } = useQuery({
    queryKey: ["documents", "shipment", shipmentId, "includeDocumentType"] as const,
    queryFn: () =>
      apiService.documentService.getByResource("shipment", shipmentId, undefined, {
        includeDocumentType: "true",
      }),
    enabled: hasShipmentId,
  });

  const docRows = useMemo<DocumentRow[]>(
    () =>
      (billingReadiness?.requirements ?? []).map((requirement) => {
        const matchingDocuments = getMatchingDocuments(requirement, documents);

        return {
          id: requirement.documentTypeId,
          label: requirement.documentTypeName,
          requirement,
          matchedDocumentCount: matchingDocuments.length,
        };
      }),
    [billingReadiness?.requirements, documents],
  );

  const isLoading = isBillingReadinessLoading || isDocumentsLoading;
  const isError = isBillingReadinessError || isDocumentsError;

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-col gap-1 text-[11px]">
        {isLoading ? (
          Array.from({ length: 4 }).map((_, index) => (
            <div key={index} className="flex items-center justify-between gap-3">
              <Skeleton className="h-3 w-38" />
              <Skeleton className="h-3 w-24" />
            </div>
          ))
        ) : isError ? (
          <span className="text-muted-foreground">Documents unavailable</span>
        ) : docRows.length === 0 ? (
          <span className="text-muted-foreground">No required documents</span>
        ) : (
          docRows.map((row) => (
            <div key={row.id} className="flex items-center justify-between gap-3">
              <span className="truncate text-muted-foreground">{row.label}</span>
              {row.matchedDocumentCount > 0 ? (
                <span className="max-w-32 truncate text-right font-table text-[10.5px] text-success tabular-nums">
                  {getUploadedCountLabel(row.matchedDocumentCount)}
                </span>
              ) : (
                <Button
                  type="button"
                  variant="link"
                  size="xxs"
                  className="h-auto px-0 py-0 text-[10.5px] text-blue-500 hover:underline"
                  disabled={!hasShipmentId}
                  onClick={() =>
                    onUpload(shipment, {
                      documentTypeId: row.requirement.documentTypeId,
                      documentTypeName: row.requirement.documentTypeName,
                    })
                  }
                >
                  Upload
                </Button>
              )}
            </div>
          ))
        )}
      </div>
    </div>
  );
}
