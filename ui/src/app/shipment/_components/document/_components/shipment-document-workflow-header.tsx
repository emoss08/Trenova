import { type DocumentCategory } from "@/types/document";
import type React from "react";

export function ShipmentDocumentWorkflowHeader({
  activeCategoryData,
}: {
  activeCategoryData?: DocumentCategory | null;
}) {
  return (
    <ShipmentDocumentWorkflowHeaderOuter>
      <ShipmentDocumentWorkflowHeaderInner>
        <ShipmentDocumentWorkflowHeaderDetails
          activeCategoryData={activeCategoryData}
        />
      </ShipmentDocumentWorkflowHeaderInner>
    </ShipmentDocumentWorkflowHeaderOuter>
  );
}

function ShipmentDocumentWorkflowHeaderOuter({
  children,
}: {
  children: React.ReactNode;
}) {
  return <div className="p-4 border-b border-border">{children}</div>;
}

function ShipmentDocumentWorkflowHeaderInner({
  children,
}: {
  children: React.ReactNode;
}) {
  return <div className="flex flex-col">{children}</div>;
}

function ShipmentDocumentWorkflowHeaderDetails({
  activeCategoryData,
}: {
  activeCategoryData?: DocumentCategory | null;
}) {
  return (
    <>
      <h2 className="text-lg font-semibold">
        {activeCategoryData?.name || "Document Management"}
      </h2>
      <p className="text-2xs text-muted-foreground">
        {activeCategoryData?.description ||
          "Upload and manage shipment documents"}
      </p>
    </>
  );
}
