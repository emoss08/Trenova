import {
  AddOrderChargeDocument,
  AttachOrderShipmentsDocument,
  CancelOrderDocument,
  CloseOrderDocument,
  CreateOrderDocument,
  DetachOrderShipmentDocument,
  OrderDetailDocument,
  RemoveOrderChargeDocument,
  SetOrderChargeAllocationsDocument,
  UpdateOrderChargeDocument,
  UpdateOrderDocument,
  type AddOrderChargeMutation,
  type AttachOrderShipmentsMutation,
  type CancelOrderMutation,
  type CloseOrderMutation,
  type DetachOrderShipmentMutation,
  type OrderDetailQuery,
  type OrderInput,
  type RemoveOrderChargeMutation,
  type SetOrderChargeAllocationsMutation,
  type UpdateOrderChargeMutation,
  type ChargeAllocationInput,
} from "@trenova/graphql/generated/graphql";
import {
  createInvoicesFromOrder,
  createInvoicesFromShipments,
  type CreatedInvoice,
} from "@/lib/graphql/invoice";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { UnmaskFragments } from "@trenova/shared/types/graphql-connection";
import type { OrderFormValues } from "@trenova/shared/types/order";

// The order's status is derived from its shipment legs, so it is never sent on write.
// Version rides along for optimistic locking on updates.
function toOrderInput(values: OrderFormValues): OrderInput {
  return {
    customerId: values.customerId,
    ownerId: values.ownerId || undefined,
    poNumber: values.poNumber || undefined,
    bol: values.bol || undefined,
    currencyCode: values.currencyCode,
    quotedAmount: values.quotedAmount != null ? String(values.quotedAmount) : undefined,
    baseAmount: values.baseAmount != null ? String(values.baseAmount) : undefined,
    version: values.version ?? undefined,
  };
}

export async function createOrder(values: OrderFormValues): Promise<OrderFormValues> {
  const data = await requestGraphQL({
    document: CreateOrderDocument,
    operationName: "CreateOrder",
    variables: { input: toOrderInput(values) },
  });

  return {
    ...values,
    id: data.createOrder.id,
    orderNumber: data.createOrder.orderNumber,
    status: data.createOrder.status,
    version: data.createOrder.version,
  };
}

export async function updateOrder(id: string, values: OrderFormValues): Promise<OrderFormValues> {
  const data = await requestGraphQL({
    document: UpdateOrderDocument,
    operationName: "UpdateOrder",
    variables: { id, input: toOrderInput(values) },
  });

  return {
    ...values,
    id: data.updateOrder.id,
    orderNumber: data.updateOrder.orderNumber,
    status: data.updateOrder.status,
    version: data.updateOrder.version,
  };
}

/*
 * Unmasked all the way down. The detail query nests fragments for charges,
 * their allocations and the created invoice; read off the masked type and
 * every one of those reads as a type with no properties, which is what the
 * charge dialog and the legs section were failing on.
 */
export type OrderDetail = UnmaskFragments<NonNullable<OrderDetailQuery["order"]>>;
export type OrderLeg = OrderDetail["legs"][number];
export type OrderCharge = OrderDetail["charges"][number];

export async function fetchOrderDetail(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<OrderDetail> {
  const data = await requestGraphQL({
    document: OrderDetailDocument,
    operationName: "OrderDetail",
    variables: { id },
    signal: options?.signal,
  });

  if (!data.order) {
    throw new Error("Order not found");
  }

  // The masks are resolved at runtime by the transport; this is the type
  // catching up with what the response already is.
  return data.order as unknown as OrderDetail;
}

export async function attachOrderShipments(
  orderId: string,
  shipmentIds: string[],
): Promise<AttachOrderShipmentsMutation["attachOrderShipments"]> {
  const data = await requestGraphQL({
    document: AttachOrderShipmentsDocument,
    operationName: "AttachOrderShipments",
    variables: { orderId, shipmentIds },
  });

  return data.attachOrderShipments;
}

export async function detachOrderShipment(
  orderId: string,
  shipmentId: string,
): Promise<DetachOrderShipmentMutation["detachOrderShipment"]> {
  const data = await requestGraphQL({
    document: DetachOrderShipmentDocument,
    operationName: "DetachOrderShipment",
    variables: { orderId, shipmentId },
  });

  return data.detachOrderShipment;
}

/**
 * Bills the order and returns the primary payer's invoice. A split-billed order
 * produces one invoice per payer; use createInvoicesFromOrder for all of them.
 * `offCycleReason` is required only when the customer is on a statement.
 */
export async function createInvoiceFromOrder(
  orderId: string,
  offCycleReason?: string,
): Promise<CreatedInvoice> {
  const result = await createInvoicesFromOrder(orderId, offCycleReason);
  return result.primary;
}

/** See createInvoiceFromOrder for what `offCycleReason` means. */
export async function createInvoiceFromShipments(
  shipmentIds: string[],
  offCycleReason?: string,
): Promise<CreatedInvoice> {
  const result = await createInvoicesFromShipments(shipmentIds, offCycleReason);
  return result.primary;
}

export async function addOrderCharge(
  orderId: string,
  description: string,
  amount: string,
  allocations?: ChargeAllocationInput[] | null,
): Promise<AddOrderChargeMutation["addOrderCharge"]> {
  const data = await requestGraphQL({
    document: AddOrderChargeDocument,
    operationName: "AddOrderCharge",
    variables: { orderId, description, amount, allocations: allocations ?? undefined },
  });

  return data.addOrderCharge;
}

export async function updateOrderCharge(input: {
  orderId: string;
  chargeId: string;
  description: string;
  amount: string;
  version: number;
  allocations?: ChargeAllocationInput[] | null;
}): Promise<UpdateOrderChargeMutation["updateOrderCharge"]> {
  const data = await requestGraphQL({
    document: UpdateOrderChargeDocument,
    operationName: "UpdateOrderCharge",
    variables: { input: { ...input, allocations: input.allocations ?? undefined } },
  });

  return data.updateOrderCharge;
}

export async function setOrderChargeAllocations(input: {
  orderId: string;
  chargeId: string;
  allocations: ChargeAllocationInput[];
}): Promise<SetOrderChargeAllocationsMutation["setOrderChargeAllocations"]> {
  const data = await requestGraphQL({
    document: SetOrderChargeAllocationsDocument,
    operationName: "SetOrderChargeAllocations",
    variables: { input },
  });

  return data.setOrderChargeAllocations;
}

export async function removeOrderCharge(
  orderId: string,
  chargeId: string,
): Promise<RemoveOrderChargeMutation["removeOrderCharge"]> {
  const data = await requestGraphQL({
    document: RemoveOrderChargeDocument,
    operationName: "RemoveOrderCharge",
    variables: { input: { orderId, chargeId } },
  });

  return data.removeOrderCharge;
}

export async function closeOrder(id: string): Promise<CloseOrderMutation["closeOrder"]> {
  const data = await requestGraphQL({
    document: CloseOrderDocument,
    operationName: "CloseOrder",
    variables: { id },
  });

  return data.closeOrder;
}

export async function cancelOrder(
  id: string,
  cancelReason: string,
): Promise<CancelOrderMutation["cancelOrder"]> {
  const data = await requestGraphQL({
    document: CancelOrderDocument,
    operationName: "CancelOrder",
    variables: { id, cancelReason },
  });

  return data.cancelOrder;
}
