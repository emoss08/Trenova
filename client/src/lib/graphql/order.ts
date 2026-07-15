import {
  AddOrderChargeDocument,
  AttachOrderShipmentsDocument,
  CreateInvoiceFromOrderDocument,
  CreateOrderDocument,
  DetachOrderShipmentDocument,
  OrderDetailDocument,
  RemoveOrderChargeDocument,
  UpdateOrderDocument,
  type AddOrderChargeMutation,
  type AttachOrderShipmentsMutation,
  type CreateInvoiceFromOrderMutation,
  type DetachOrderShipmentMutation,
  type OrderDetailQuery,
  type OrderInput,
  type RemoveOrderChargeMutation,
} from "@/graphql/generated/graphql";
import { requestGraphQL } from "@/lib/graphql";
import type { OrderFormValues } from "@/types/order";

// The order's status is derived from its shipment legs, so it is never sent on write.
function toOrderInput(values: OrderFormValues): OrderInput {
  return {
    customerId: values.customerId,
    ownerId: values.ownerId || undefined,
    poNumber: values.poNumber || undefined,
    bol: values.bol || undefined,
    currencyCode: values.currencyCode,
    quotedAmount: values.quotedAmount != null ? String(values.quotedAmount) : undefined,
    baseAmount: values.baseAmount != null ? String(values.baseAmount) : undefined,
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
  };
}

export type OrderDetail = NonNullable<OrderDetailQuery["order"]>;
export type OrderLeg = OrderDetail["legs"][number];
export type OrderCharge = OrderDetail["charges"][number];

export async function fetchOrderDetail(id: string): Promise<OrderDetail> {
  const data = await requestGraphQL({
    document: OrderDetailDocument,
    operationName: "OrderDetail",
    variables: { id },
  });

  if (!data.order) {
    throw new Error("Order not found");
  }

  return data.order;
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

export async function createInvoiceFromOrder(
  orderId: string,
): Promise<CreateInvoiceFromOrderMutation["createInvoiceFromOrder"]> {
  const data = await requestGraphQL({
    document: CreateInvoiceFromOrderDocument,
    operationName: "CreateInvoiceFromOrder",
    variables: { orderId },
  });

  return data.createInvoiceFromOrder;
}

export async function addOrderCharge(
  orderId: string,
  description: string,
  amount: string,
): Promise<AddOrderChargeMutation["addOrderCharge"]> {
  const data = await requestGraphQL({
    document: AddOrderChargeDocument,
    operationName: "AddOrderCharge",
    variables: { orderId, description, amount },
  });

  return data.addOrderCharge;
}

export async function removeOrderCharge(
  orderId: string,
  chargeId: string,
): Promise<RemoveOrderChargeMutation["removeOrderCharge"]> {
  const data = await requestGraphQL({
    document: RemoveOrderChargeDocument,
    operationName: "RemoveOrderCharge",
    variables: { orderId, chargeId },
  });

  return data.removeOrderCharge;
}
