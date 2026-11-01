import {
  CustomerStatusNotificationsDocument,
  CustomerTrackingLinksDocument,
  CustomerTrackingProfileDocument,
  IssueCustomerTrackingLinkDocument,
  RevokeCustomerTrackingLinkDocument,
  type CustomerStatusNotificationFieldsFragment,
  type CustomerStatusNotificationsQuery,
  type CustomerStatusNotificationsQueryVariables,
  type CustomerTrackingField,
  type CustomerTrackingLinkFieldsFragment,
  type CustomerTrackingLinksQuery,
  type CustomerTrackingLinksQueryVariables,
  type CustomerTrackingProfileFieldsFragment,
  type CustomerTrackingProfileQuery,
  type CustomerTrackingProfileQueryVariables,
  type IssueCustomerTrackingLinkMutation,
  type IssueCustomerTrackingLinkMutationVariables,
  type RevokeCustomerTrackingLinkMutation,
  type RevokeCustomerTrackingLinkMutationVariables,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type CustomerTrackingLinkRow = CustomerTrackingLinkFieldsFragment;
export type CustomerStatusNotificationRow = CustomerStatusNotificationFieldsFragment;
export type CustomerTrackingProfile = CustomerTrackingProfileFieldsFragment;
export type { CustomerTrackingField };

export type IssuedCustomerTrackingLink = {
  link: CustomerTrackingLinkRow;
  url: string;
  emailed: boolean;
  emailError: string | null;
};

type RequestOptions = { signal?: AbortSignal };

export async function getCustomerTrackingProfile(
  customerId: string,
  options?: RequestOptions,
): Promise<CustomerTrackingProfile | null> {
  const data = await requestGraphQL<
    CustomerTrackingProfileQuery,
    CustomerTrackingProfileQueryVariables
  >({
    document: CustomerTrackingProfileDocument,
    operationName: "CustomerTrackingProfile",
    variables: { customerId },
    signal: options?.signal,
  });
  return (data.customer?.trackingProfile ?? null) as CustomerTrackingProfile | null;
}

export async function listCustomerTrackingLinks(
  shipmentId: string,
  options?: RequestOptions,
): Promise<CustomerTrackingLinkRow[]> {
  const data = await requestGraphQL<
    CustomerTrackingLinksQuery,
    CustomerTrackingLinksQueryVariables
  >({
    document: CustomerTrackingLinksDocument,
    operationName: "CustomerTrackingLinks",
    variables: { shipmentId },
    signal: options?.signal,
  });
  return data.customerTrackingLinks as CustomerTrackingLinkRow[];
}

export async function listCustomerStatusNotifications(
  shipmentId: string,
  options?: RequestOptions,
): Promise<CustomerStatusNotificationRow[]> {
  const data = await requestGraphQL<
    CustomerStatusNotificationsQuery,
    CustomerStatusNotificationsQueryVariables
  >({
    document: CustomerStatusNotificationsDocument,
    operationName: "CustomerStatusNotifications",
    variables: { shipmentId },
    signal: options?.signal,
  });
  return data.customerStatusNotifications as CustomerStatusNotificationRow[];
}

export async function issueCustomerTrackingLink(input: {
  shipmentId: string;
  recipientEmail?: string;
}): Promise<IssuedCustomerTrackingLink> {
  const data = await requestGraphQL<
    IssueCustomerTrackingLinkMutation,
    IssueCustomerTrackingLinkMutationVariables
  >({
    document: IssueCustomerTrackingLinkDocument,
    operationName: "IssueCustomerTrackingLink",
    variables: {
      input: { shipmentId: input.shipmentId, recipientEmail: blankToNull(input.recipientEmail) },
    },
  });
  const payload = data.issueCustomerTrackingLink;
  return {
    link: payload.link as CustomerTrackingLinkRow,
    url: payload.url,
    emailed: payload.emailed,
    emailError: payload.emailError ?? null,
  };
}

/** Revokes one link, or every live link on the shipment when no link is named. */
export async function revokeCustomerTrackingLink(
  shipmentId: string,
  linkId?: string,
): Promise<number> {
  const data = await requestGraphQL<
    RevokeCustomerTrackingLinkMutation,
    RevokeCustomerTrackingLinkMutationVariables
  >({
    document: RevokeCustomerTrackingLinkDocument,
    operationName: "RevokeCustomerTrackingLink",
    variables: { shipmentId, linkId: linkId ?? null },
  });
  return data.revokeCustomerTrackingLink;
}

function blankToNull(value: string | null | undefined): string | null {
  const trimmed = value?.trim() ?? "";
  return trimmed === "" ? null : trimmed;
}
