import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Form } from "@/components/ui/form";
import { ScrollArea } from "@/components/ui/scroll-area";
import { ShipmentForm } from "@/routes/shipment/_components/shipment-form";
import { apiService } from "@/services/api";
import type {
  Document,
  DocumentIntelligenceField,
  DocumentIntelligenceStop,
  DocumentShipmentDraft,
} from "@/types/document";
import { shipmentCreateSchema, type ShipmentCreateInput } from "@/types/shipment";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { AlertCircleIcon, LoaderCircleIcon, SparklesIcon } from "lucide-react";
import { useEffect } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { Link } from "react-router";
import { toast } from "sonner";

interface DocumentShipmentDraftReviewDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  document: Document | null;
  draft: DocumentShipmentDraft | null;
  sourceResourceType: string;
  sourceResourceId: string;
  embedded?: boolean;
  onShipmentCreated?: (result: {
    shipmentId: string;
    attachError: Error | null;
  }) => void;
}

function parseInteger(value: unknown): number | undefined {
  if (typeof value === "number" && Number.isFinite(value)) {
    return Math.trunc(value);
  }
  if (typeof value !== "string") return undefined;

  const digits = value.replace(/[^\d-]/g, "");
  if (!digits) return undefined;

  const parsed = Number.parseInt(digits, 10);
  return Number.isFinite(parsed) ? parsed : undefined;
}

function parseDecimal(value: unknown): number {
  if (typeof value === "number" && Number.isFinite(value)) return value;
  if (typeof value !== "string") return 0;

  const normalized = value.replace(/[^0-9.-]/g, "");
  const parsed = Number.parseFloat(normalized);
  return Number.isFinite(parsed) ? parsed : 0;
}

function parseTimestamp(value?: string): number {
  if (!value?.trim()) return 0;
  const parsed = Date.parse(value);
  if (Number.isNaN(parsed)) return 0;
  return Math.floor(parsed / 1000);
}

function formatUnixTimestamp(value?: number | null) {
  if (!value) return "Not recorded";
  return new Date(value * 1000).toLocaleString();
}

function getFieldValue(draft: DocumentShipmentDraft | null, key: string): unknown {
  return draft?.draftData?.fields?.[key]?.value;
}

function stopTypeFromRole(role?: string) {
  return role === "delivery" ? ("Delivery" as const) : ("Pickup" as const);
}

function addressLine(stop: DocumentIntelligenceStop) {
  return [stop.addressLine1, stop.addressLine2, stop.city, stop.state, stop.postalCode]
    .filter((part) => !!part && part.trim() !== "")
    .join(", ");
}

function createDefaultShipmentValues(draft: DocumentShipmentDraft | null): ShipmentCreateInput {
  const extractedStops = draft?.draftData?.stops ?? [];
  const rate = parseDecimal(getFieldValue(draft, "rate"));
  const weight = parseInteger(getFieldValue(draft, "weight"));
  const pieces = parseInteger(getFieldValue(draft, "pieces"));
  const bol =
    getFieldValue(draft, "bol") ??
    getFieldValue(draft, "reference") ??
    getFieldValue(draft, "loadNumber");

  const moves =
    extractedStops.length > 0
      ? [
          {
            status: "New" as const,
            loaded: true,
            sequence: 0,
            distance: 0,
            stops: extractedStops.map((stop, index) => ({
              status: "New" as const,
              type: stopTypeFromRole(stop.role),
              scheduleType: stop.appointmentRequired ? ("Appointment" as const) : ("Open" as const),
              locationId: "",
              sequence: index,
              scheduledWindowStart: parseTimestamp(stop.date),
              scheduledWindowEnd: null,
              addressLine: addressLine(stop),
              weight: weight ?? null,
              pieces: pieces ?? null,
            })),
          },
        ]
      : [
          {
            status: "New" as const,
            loaded: true,
            sequence: 0,
            distance: 0,
            stops: [
              {
                status: "New" as const,
                type: "Pickup" as const,
                scheduleType: "Open" as const,
                locationId: "",
                sequence: 0,
                scheduledWindowStart: 0,
                scheduledWindowEnd: null,
                pieces: pieces ?? null,
                weight: weight ?? null,
              },
              {
                status: "New" as const,
                type: "Delivery" as const,
                scheduleType: "Open" as const,
                locationId: "",
                sequence: 1,
                scheduledWindowStart: 0,
                scheduledWindowEnd: null,
                pieces: pieces ?? null,
                weight: weight ?? null,
              },
            ],
          },
        ];

  return {
    status: "New",
    bol: typeof bol === "string" ? bol : "",
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
    freightChargeAmount: rate,
    baseRate: rate,
    totalChargeAmount: rate,
    pieces: pieces ?? undefined,
    weight: weight ?? undefined,
    temperatureMin: undefined,
    temperatureMax: undefined,
    actualDeliveryDate: undefined,
    actualShipDate: undefined,
    canceledAt: undefined,
    ratingUnit: 1,
    additionalCharges: [],
    commodities: [],
    moves,
  };
}

function renderField(field?: DocumentIntelligenceField) {
  if (field?.value == null) return "Not extracted";
  if (typeof field.value === "string" && field.value.trim() === "") {
    return "Not extracted";
  }
  if (typeof field.value === "string") return field.value;
  if (typeof field.value === "number") return String(field.value);
  return JSON.stringify(field.value);
}

export function DocumentShipmentDraftReviewDialog({
  open,
  onOpenChange,
  document,
  draft,
  sourceResourceType,
  sourceResourceId,
  embedded = false,
  onShipmentCreated,
}: DocumentShipmentDraftReviewDialogProps) {
  const queryClient = useQueryClient();
  const form = useForm({
    resolver: zodResolver(shipmentCreateSchema),
    defaultValues: createDefaultShipmentValues(draft),
    mode: "onChange",
  });

  useEffect(() => {
    if (open) {
      form.reset(createDefaultShipmentValues(draft));
    }
  }, [draft, form, open]);

  const createShipment = useMutation({
    mutationFn: async (values: ShipmentCreateInput) => {
      const shipment = await apiService.shipmentService.create(values);
      const shipmentId = shipment.id;
      if (!shipmentId) {
        throw new Error("Shipment was created without an ID");
      }
      let attachError: Error | null = null;

      if (document) {
        try {
          await apiService.documentService.attachToShipment(document.id, shipmentId);
        } catch (error) {
          attachError = error instanceof Error ? error : new Error("Failed to attach document");
        }
      }

      return { shipment, shipmentId, attachError };
    },
    onSuccess: ({ shipment, shipmentId, attachError }) => {
      void queryClient.invalidateQueries({ queryKey: ["shipment-list"] });
      void queryClient.invalidateQueries({
        queryKey: ["documents", sourceResourceType, sourceResourceId],
      });
      if (document) {
        void queryClient.invalidateQueries({
          queryKey: ["document-content", document.id],
        });
        void queryClient.invalidateQueries({
          queryKey: ["document-shipment-draft", document.id],
        });
      }
      void queryClient.invalidateQueries({
        queryKey: ["documents", "shipment", shipment.id],
      });

      if (attachError) {
        toast.warning("Shipment created, but the source document could not be attached", {
          description: attachError.message,
        });
      } else {
        toast.success("Shipment created from document draft");
      }

      onShipmentCreated?.({
        shipmentId,
        attachError,
      });
      if (!embedded) {
        onOpenChange(false);
      }
    },
    onError: (error) => {
      toast.error(`Failed to create shipment: ${error.message}`);
    },
  });

  const stops = draft?.draftData?.stops ?? [];
  const missingFields = draft?.draftData?.missingFields ?? [];
  const signals = draft?.draftData?.signals ?? [];
  const isAttached = !!draft?.attachedShipmentId;

  const content = (
    <>
      <div className="grid min-h-0 md:grid-cols-[320px_minmax(0,1fr)]">
        <ScrollArea className="flex h-[400px] border-r bg-muted/10">
          <div className="grid gap-4 p-4">
            <div className="rounded-lg border p-3">
              <div className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
                Source Document
              </div>
              <div className="mt-1 text-sm font-medium">
                {document?.originalName ?? "Document"}
              </div>
              {document?.detectedKind ? (
                <div className="mt-2">
                  <Badge variant="info">{document.detectedKind}</Badge>
                </div>
              ) : null}
            </div>
            <div className="rounded-lg border p-3">
              <div className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
                Draft Summary
              </div>
              <div className="mt-3 grid gap-2 text-sm">
                <div>
                  <span className="text-muted-foreground">Shipper:</span>{" "}
                  {renderField(draft?.draftData?.fields?.shipper)}
                </div>
                <div>
                  <span className="text-muted-foreground">Consignee:</span>{" "}
                  {renderField(draft?.draftData?.fields?.consignee)}
                </div>
                <div>
                  <span className="text-muted-foreground">Reference:</span>{" "}
                  {renderField(
                    draft?.draftData?.fields?.reference ?? draft?.draftData?.fields?.loadNumber,
                  )}
                </div>
                <div>
                  <span className="text-muted-foreground">Rate:</span>{" "}
                  {renderField(draft?.draftData?.fields?.rate)}
                </div>
              </div>
            </div>
            {isAttached ? (
              <div className="rounded-lg border border-emerald-200 bg-emerald-50/70 p-3 text-sm text-emerald-950">
                <div className="font-medium">This source document is already attached.</div>
                <div className="mt-1 text-emerald-900/80">
                  Shipment {draft?.attachedShipmentId} attached{" "}
                  {formatUnixTimestamp(draft?.attachedAt)}.
                </div>
                <div className="mt-3">
                  <Button
                    variant="outline"
                    size="sm"
                    render={<Link to="/shipment-management/shipments" />}
                  >
                    Open Shipments
                  </Button>
                </div>
              </div>
            ) : null}
            {signals.length > 0 ? (
              <div className="rounded-lg border p-3">
                <div className="mb-2 flex items-center gap-2 text-xs font-medium tracking-wide text-muted-foreground uppercase">
                  <SparklesIcon className="size-3.5" />
                  Draft Signals
                </div>
                <div className="flex flex-wrap gap-2">
                  {signals.map((signal) => (
                    <Badge key={signal} variant="secondary">
                      {signal}
                    </Badge>
                  ))}
                </div>
              </div>
            ) : null}
            {missingFields.length > 0 ? (
              <div className="rounded-lg border border-dashed p-3">
                <div className="mb-2 flex items-center gap-2 text-sm font-medium text-foreground">
                  <AlertCircleIcon className="size-4" />
                  Review needed
                </div>
                <div className="flex flex-wrap gap-2">
                  {missingFields.map((field) => (
                    <Badge key={field} variant="outline">
                      {field}
                    </Badge>
                  ))}
                </div>
              </div>
            ) : null}
            <div className="rounded-lg border p-3">
              <div className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
                Extracted Stops
              </div>
              {stops.length === 0 ? (
                <div className="mt-2 text-sm text-muted-foreground">
                  No stops were extracted. You can still create the shipment manually.
                </div>
              ) : (
                <div className="mt-3 grid gap-3">
                  {stops.map((stop) => (
                    <div
                      key={`${stop.role}-${stop.sequence}-${stop.pageNumber ?? 0}`}
                      className="rounded-md border p-2"
                    >
                      <div className="flex items-center justify-between gap-2">
                        <div className="text-sm font-medium">
                          {stop.role === "delivery" ? "Delivery" : "Pickup"} #{stop.sequence}
                        </div>
                        {stop.pageNumber ? (
                          <Badge variant="outline">Page {stop.pageNumber}</Badge>
                        ) : null}
                      </div>
                      <div className="mt-1 text-xs text-muted-foreground">
                        {addressLine(stop) || "Address not extracted"}
                      </div>
                      {stop.date || stop.timeWindow ? (
                        <div className="mt-1 text-xs text-muted-foreground">
                          {[stop.date, stop.timeWindow].filter(Boolean).join(" · ")}
                        </div>
                      ) : null}
                    </div>
                  ))}
                </div>
              )}
            </div>
            <div className="rounded-lg border border-dashed p-3 text-xs text-muted-foreground">
              Location, customer, service type, shipment type, and formula template still need to
              be confirmed before the shipment can be created.
            </div>
          </div>
        </ScrollArea>
        <div className="min-w-0">
          <FormProvider {...form}>
            <Form
              id="document-shipment-draft-form"
              className="flex h-full min-h-0 flex-col"
              onSubmit={form.handleSubmit((values) =>
                createShipment.mutate(shipmentCreateSchema.parse(values)),
              )}
            >
              <ScrollArea className="max-h-[calc(90vh-170px)]">
                <div className="p-6">
                  {isAttached ? (
                    <div className="mb-4 rounded-lg border border-dashed p-3 text-sm text-muted-foreground">
                      Shipment creation from this draft is disabled because the source document
                      has already been linked to shipment {draft?.attachedShipmentId}.
                    </div>
                  ) : null}
                  <ShipmentForm />
                </div>
              </ScrollArea>
            </Form>
          </FormProvider>
        </div>
      </div>
      <div className="flex flex-col-reverse gap-2 border-t bg-muted/50 p-4 sm:flex-row sm:justify-end">
        <Button
          type="submit"
          form="document-shipment-draft-form"
          disabled={createShipment.isPending || isAttached}
        >
          {createShipment.isPending ? <LoaderCircleIcon className="size-4 animate-spin" /> : null}
          {isAttached ? "Shipment Attached" : "Create Shipment"}
        </Button>
      </div>
    </>
  );

  if (embedded) {
    return content;
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="gap-0 overflow-hidden p-0 sm:max-w-6xl" showCloseButton>
        <DialogHeader className="border-b px-6 pt-6 pb-4">
          <div className="flex flex-wrap items-center gap-2">
            <DialogTitle>Create Shipment from Document</DialogTitle>
            {draft?.status ? <Badge variant="secondary">{draft.status}</Badge> : null}
          </div>
          <DialogDescription>
            Review the extracted shipment draft, fill in the required shipment details, then create
            a shipment and attach this document lineage to it.
          </DialogDescription>
        </DialogHeader>
        {content}
      </DialogContent>
    </Dialog>
  );
}
