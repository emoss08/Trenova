import { FormCreatePanel } from "@/components/form-create-panel";
import { TabbedFormEditPanel } from "@/components/tabbed-form-edit-panel";
import { useIdempotencyKey } from "@/hooks/use-idempotency-key";
import { apiService } from "@/services/api";
import { rulesResolver } from "@/lib/rules-resolver";
import { refineShipmentInputs, shipmentTemplateIds } from "@/lib/shipment-tariff-inputs";
import { tariffInputsResolver } from "@/lib/tariff-inputs-resolver";
import { TARIFF_INPUTS_REFINEMENT } from "@trenova/shared/lib/tariff-inputs";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { getTodayDate } from "@trenova/shared/lib/date";
import { withShipmentAllocations } from "@trenova/shared/rules/allocation-form";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import {
  shipmentCreateSchema,
  shipmentUpdateSchema,
  type Shipment,
  type ShipmentCreateInput,
  type ShipmentUpdateInput,
} from "@trenova/shared/types/shipment";
import {
  AlertTriangleIcon,
  ClockRewindIcon,
  File06Icon,
  MessageSquare01Icon,
  PhoneCall01Icon,
  Share07Icon,
  ShippingContainerIcon,
} from "@trenova/shared/components/icons";
import { parseAsBoolean, useQueryState } from "nuqs";
import { NEW_SHIPMENT_CUSTOMER_PARAM } from "@/lib/shipment-links";
import { shipment as shipmentQueries } from "@/lib/queries/shipment";
import { lazy } from "react";
import { type Resolver, useForm } from "react-hook-form";
import { CheckCallsSkeleton } from "./check-calls/check-calls-skeleton";
import { ShipmentBillingActionsMenu } from "./shipment-billing-actions-menu";
import { ShipmentForm } from "./shipment-form";

const AuditTab = lazy(() => import("@/components/audit-tab"));
const DocumentsTab = lazy(() => import("@/components/documents/documents-tab"));
const ShipmentCommentsTab = lazy(() => import("./comments"));
const ShipmentServiceFailuresTab = lazy(() => import("./shipment-service-failures"));
const ShipmentCustomerTrackingTab = lazy(() => import("./shipment-customer-tracking"));
const ShipmentCheckCallsTab = lazy(() => import("./check-calls/shipment-check-calls"));

function OwnerDisplay({ ownerId }: { ownerId?: string | null }) {
  const t = useT();

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
        <span className="text-2xs text-muted-foreground">{t("Owner:")}</span>
        <span className="text-2xs text-info-foreground">{owner?.name}</span>
      </div>
    );
  }

  return <span className="text-2xs text-foreground">{t("No owner assigned")}</span>;
}

const getDefaultValues = (customerId: string | null): ShipmentCreateInput => {
  const today = getTodayDate();

  return {
    status: "New",
    bol: "",
    externalReference: "",
    serviceTypeId: "",
    shipmentTypeId: "",
    customerId: customerId ?? "",
    tractorTypeId: undefined,
    trailerTypeId: undefined,
    ownerId: undefined,
    enteredById: undefined,
    canceledById: undefined,
    formulaTemplateId: "",
    consolidationGroupId: undefined,
    otherChargeAmount: 0,
    freightChargeAmount: 0,
    tariffInputs: {},
    totalChargeAmount: 0,
    pieces: undefined,
    weight: undefined,
    temperatureMin: undefined,
    temperatureMax: undefined,
    actualDeliveryDate: undefined,
    actualShipDate: undefined,
    canceledAt: undefined,
    fuelSurchargeLocked: false,
    freightTerms: "Prepaid",
    freightAllocations: [],
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
            scheduledWindowStart: today,
            scheduledWindowEnd: null,
          },
          {
            status: "New",
            type: "Delivery",
            scheduleType: "Open",
            locationId: "",
            sequence: 1,
            scheduledWindowStart: today,
            scheduledWindowEnd: null,
          },
        ],
      },
    ],
  };
};

export function ShipmentPanel({ open, onOpenChange, mode, row }: DataTablePanelProps<Shipment>) {
  const t = useT();

  const [customerId] = useQueryState(NEW_SHIPMENT_CUSTOMER_PARAM);
  const defaultValues = getDefaultValues(mode === "create" ? customerId : null);
  const withIdempotencyKey = useIdempotencyKey();
  const queryClient = useQueryClient();
  // The shipment's input values are checked against its tariff's declarations as well as the
  // schema, so a mistyped or missing input is flagged at its field before the save.
  const createForm = useForm({
    resolver: rulesResolver((rules) =>
      tariffInputsResolver<ShipmentCreateInput>(
        queryClient,
        shipmentTemplateIds,
        (inputs) =>
          zodResolver(
            withShipmentAllocations(shipmentCreateSchema, rules).superRefine(
              refineShipmentInputs(inputs),
              TARIFF_INPUTS_REFINEMENT,
            ),
          ) as Resolver<ShipmentCreateInput>,
      ),
    ),
    defaultValues,
    mode: "onChange",
  });

  const editForm = useForm({
    resolver: rulesResolver((rules) =>
      tariffInputsResolver<ShipmentUpdateInput>(
        queryClient,
        shipmentTemplateIds,
        (inputs) =>
          zodResolver(
            withShipmentAllocations(shipmentUpdateSchema, rules).superRefine(
              refineShipmentInputs(inputs),
              TARIFF_INPUTS_REFINEMENT,
            ),
          ) as Resolver<ShipmentUpdateInput>,
      ),
    ),
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
  const { data: serviceFailures } = useQuery({
    queryKey: ["shipment-service-failure-count", row?.id],
    queryFn: () => apiService.serviceFailureService.listByShipment(row!.id!, "limit=100"),
    enabled: !!row?.id && mode === "edit",
    staleTime: 30_000,
  });
  const serviceFailureCount = serviceFailures?.count ?? 0;
  const { data: checkCalls } = useQuery({
    ...shipmentQueries.checkCalls(row?.id ?? ""),
    enabled: !!row?.id && mode === "edit",
  });
  const checkCallCount = checkCalls?.length ?? 0;

  const extraTabs = [
    {
      value: "documents",
      label: t("Documents"),
      icon: File06Icon,
      hideFooter: true,
      content: DocumentsTab,
      contentProps: {
        resourceType: "shipment",
        resourceId: row?.id,
      },
    },
    {
      value: "service-failures",
      label:
        serviceFailureCount > 0
          ? t("Service Failures ({0})", serviceFailureCount)
          : t("Service Failures"),
      icon: AlertTriangleIcon,
      hideFooter: true,
      content: ShipmentServiceFailuresTab,
      contentProps: {
        shipment: row,
      },
    },
    {
      value: "customer-tracking",
      label: t("Customer Tracking"),
      icon: Share07Icon,
      hideFooter: true,
      content: ShipmentCustomerTrackingTab,
      contentProps: {
        shipment: row,
      },
    },
    {
      value: "check-calls",
      label: checkCallCount > 0 ? t("Check Calls ({0})", checkCallCount) : t("Check Calls"),
      icon: PhoneCall01Icon,
      hideFooter: true,
      content: ShipmentCheckCallsTab,
      fallback: <CheckCallsSkeleton />,
      contentProps: {
        shipment: row,
      },
    },
    {
      value: "comments",
      label: commentCount > 0 ? t("Comments ({0})", commentCount) : t("Comments"),
      icon: MessageSquare01Icon,
      manageScroll: true,
      hideFooter: true,
      content: ShipmentCommentsTab,
      contentProps: {
        shipmentId: row?.id,
      },
    },
    {
      value: "history",
      label: t("History"),
      icon: ClockRewindIcon,
      hideFooter: true,
      content: AuditTab,
      contentProps: {
        resourceId: row?.id,
      },
    },
  ];

  const [, setLoadPlannerOpen] = useQueryState("loadPlanner", parseAsBoolean.withDefault(false));

  if (mode === "edit") {
    return (
      <TabbedFormEditPanel
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        form={editForm}
        url="/shipments/"
        queryKey="shipment-list"
        title={t("Shipment")}
        fieldKey="proNumber"
        formComponent={<ShipmentForm />}
        tabs={extraTabs}
        mutationFn={(values, currentRow) =>
          apiService.shipmentService.update(currentRow.id!, values as ShipmentUpdateInput)
        }
        descriptionExtra={<OwnerDisplay ownerId={row?.ownerId} />}
        headerActions={
          <>
            {row && <ShipmentBillingActionsMenu shipment={row} />}
            <Tooltip>
              <TooltipTrigger
                render={
                  <Button
                    type="button"
                    variant="outline"
                    size="icon-sm"
                    onClick={() => setLoadPlannerOpen(true)}
                  />
                }
              >
                <ShippingContainerIcon className="size-4" />
              </TooltipTrigger>
              <TooltipContent side="bottom">{t("Load planner")}</TooltipContent>
            </Tooltip>
          </>
        }
        useDock
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
      title={t("Shipment")}
      formComponent={<ShipmentForm />}
      mutationFn={(values) =>
        withIdempotencyKey(values, (idempotencyKey) =>
          apiService.shipmentService.create(values as ShipmentCreateInput, { idempotencyKey }),
        )
      }
      useDock
    />
  );
}
