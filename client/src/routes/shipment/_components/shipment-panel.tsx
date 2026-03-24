import { FormCreatePanel } from "@/components/form-create-panel";
import { TabbedFormEditPanel } from "@/components/tabbed-form-edit-panel";
import { Skeleton } from "@/components/ui/skeleton";
import { apiService } from "@/services/api";
import type { DataTablePanelProps } from "@/types/data-table";
import {
  shipmentCreateSchema,
  shipmentUpdateSchema,
  type Shipment,
  type ShipmentCreateInput,
  type ShipmentUpdateInput,
} from "@/types/shipment";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
import { FileTextIcon, HistoryIcon, MessageSquareIcon } from "lucide-react";
import { lazy, useMemo } from "react";
import { useForm } from "react-hook-form";
import { ShipmentForm } from "./shipment-form";

const AuditTab = lazy(() => import("@/components/audit-tab"));
const DocumentsTab = lazy(() => import("@/components/documents/documents-tab"));
const ShipmentCommentsTab = lazy(() => import("./shipment-comments"));

function OwnerDisplay({ ownerId }: { ownerId?: string | null }) {
  const { data: owner, isLoading } = useQuery({
    queryKey: ["user", ownerId],
    queryFn: () => apiService.userService.get(ownerId!),
    enabled: !!ownerId,
    staleTime: 5 * 60 * 1000,
  });

  if (isLoading) return <Skeleton className="h-2.5 w-34" />;

  if (ownerId) {
    return (
      <div className="flex items-center gap-1">
        <span className="text-2xs text-muted-foreground">Owner:</span>
        <span className="text-2xs text-blue-500">{owner?.name}</span>
      </div>
    );
  }

  return <span className="text-2xs text-foreground">No owner assigned</span>;
}

const defaultValues: ShipmentCreateInput = {
  status: "New",
  bol: "",
  serviceTypeId: "",
  shipmentTypeId: "",
  customerId: "",
  tractorTypeId: undefined,
  trailerTypeId: undefined,
  ownerId: undefined,
  enteredById: undefined,
  canceledById: undefined,
  formulaTemplateId: "",
  consolidationGroupId: undefined,
  otherChargeAmount: 0,
  freightChargeAmount: 0,
  totalChargeAmount: 0,
  pieces: undefined,
  weight: undefined,
  temperatureMin: undefined,
  temperatureMax: undefined,
  actualDeliveryDate: undefined,
  actualShipDate: undefined,
  canceledAt: undefined,
  ratingUnit: 1,
  additionalCharges: [],
  commodities: [],
  moves: [
    {
      status: "New",
      loaded: true,
      sequence: 0,
      distance: 0,
      stops: [
        {
          status: "New",
          type: "Pickup",
          scheduleType: "Open",
          locationId: "",
          sequence: 0,
          scheduledWindowStart: 0,
          scheduledWindowEnd: null,
        },
        {
          status: "New",
          type: "Delivery",
          scheduleType: "Open",
          locationId: "",
          sequence: 1,
          scheduledWindowStart: 0,
          scheduledWindowEnd: null,
        },
      ],
    },
  ],
};

export function ShipmentPanel({ open, onOpenChange, mode, row }: DataTablePanelProps<Shipment>) {
  const createForm = useForm({
    resolver: zodResolver(shipmentCreateSchema),
    defaultValues,
    mode: "onChange",
  });

  const editForm = useForm({
    resolver: zodResolver(shipmentUpdateSchema),
    defaultValues: row as ShipmentUpdateInput | undefined,
    mode: "onChange",
  });

  const { data: commentCountData } = useQuery({
    queryKey: ["shipment-comment-count", row?.id],
    queryFn: () => apiService.shipmentCommentService.getCount(row!.id!),
    enabled: !!row?.id && mode === "edit",
    staleTime: 30_000,
  });
  const commentCount = commentCountData?.count ?? 0;

  const extraTabs = useMemo(
    () => [
      {
        value: "documents",
        label: "Documents",
        icon: FileTextIcon,
        hideFooter: true,
        content: DocumentsTab,
        contentProps: {
          resourceType: "shipment",
          resourceId: row?.id,
        },
      },
      {
        value: "comments",
        label: commentCount > 0 ? `Comments (${commentCount})` : "Comments",
        icon: MessageSquareIcon,
        manageScroll: true,
        hideFooter: true,
        content: ShipmentCommentsTab,
        contentProps: {
          shipmentId: row?.id,
        },
      },
      {
        value: "history",
        label: "History",
        icon: HistoryIcon,
        hideFooter: true,
        content: AuditTab,
        contentProps: {
          resourceId: row?.id,
        },
      },
    ],
    [row?.id, commentCount],
  );

  if (mode === "edit") {
    return (
      <TabbedFormEditPanel
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        form={editForm}
        url="/shipments/"
        queryKey="shipment-list"
        title="Shipment"
        fieldKey="proNumber"
        formComponent={<ShipmentForm />}
        tabs={extraTabs}
        descriptionExtra={<OwnerDisplay ownerId={row?.ownerId} />}
      />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={createForm}
      url="/shipments/"
      queryKey="shipment-list"
      title="Shipment"
      formComponent={<ShipmentForm />}
    />
  );
}
