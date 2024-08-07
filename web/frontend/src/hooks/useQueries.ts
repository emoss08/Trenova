/**
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import { toTitleCase } from "@/lib/utils";
import { getUserFavorites } from "@/services/AccountRequestService";
import {
  getAccountingControl,
  getGLAccounts,
  getRevenueCodes,
  getTags,
} from "@/services/AccountingRequestService";
import { getDailyShipmentCounts } from "@/services/AnalyticRequestService";
import { getDocumentClassifications } from "@/services/BillingRequestService";
import { getCustomers } from "@/services/CustomerRequestService";
import {
  getCommentTypes,
  getFeasibilityControl,
} from "@/services/DispatchRequestService";
import {
  getEquipmentTypes,
  getTrailers,
} from "@/services/EquipmentRequestService";
import {
  getLocations,
  getUSStates,
  searchLocation,
} from "@/services/LocationRequestService";
import {
  getBillingControl,
  getDispatchControl,
  getEmailControl,
  getEmailProfiles,
  getFeatureFlags,
  getGoogleApiInformation,
  getInvoiceControl,
  getOrganizationDetails,
  getRouteControl,
  getShipmentControl,
  getTopicNames,
} from "@/services/OrganizationRequestService";
import { getColumns } from "@/services/ReportRequestService";
import {
  getNextProNumber,
  getServiceTypes,
  getShipmentTypes,
} from "@/services/ShipmentRequestService";
import { getUserNotifications, getUsers } from "@/services/UserRequestService";
import { getWorkers } from "@/services/WorkerRequestService";
import type { QueryKeys, QueryKeyWithParams, StatusChoiceProps } from "@/types";
import type {
  GeneralLedgerAccount,
  RevenueCode,
  Tag,
} from "@/types/accounting";
import type { User } from "@/types/accounts";
import type { DocumentClassification } from "@/types/billing";
import type { Customer } from "@/types/customer";
import type { CommentType } from "@/types/dispatch";
import type { EquipmentType, Trailer } from "@/types/equipment";
import type { Location, USStates } from "@/types/location";
import type { EmailProfile, Topic } from "@/types/organization";
import type { ServiceType, ShipmentType } from "@/types/shipment";
import type { Worker } from "@/types/worker";
import { useQuery } from "@tanstack/react-query";

/**
 * Get Tags for select options
 * @param show - show or hide the query
 */
export function useTags(show: boolean) {
  const {
    data: tagsData,
    isLoading,
    isError,
    isFetched,
    isPending,
  } = useQuery({
    queryKey: ["tags"] as QueryKeys,
    queryFn: async () => getTags(),
    enabled: show,
  });

  const selectTags =
    (tagsData as Tag[])?.map((item: Tag) => ({
      value: item.id,
      label: item.name,
    })) || [];

  return { selectTags, isLoading, isError, isFetched, isPending };
}

/**
 * Get GL Accounts for select options
 * @param show - show or hide the query
 */
export function useGLAccounts(show?: boolean) {
  const {
    data: glAccountsData,
    isLoading,
    isError,
    isFetched,
  } = useQuery({
    queryKey: ["glAccounts"] as QueryKeys,
    queryFn: async () => getGLAccounts(),
    enabled: show,
  });

  const selectGLAccounts =
    (glAccountsData as GeneralLedgerAccount[])?.map(
      (item: GeneralLedgerAccount) => ({
        value: item.id,
        label: `${item.accountNumber} - ${item.accountType}`,
      }),
    ) || [];

  return { glAccountsData, selectGLAccounts, isLoading, isError, isFetched };
}

/**
 * Use Accounting Control Hook to get Accounting Control Details
 */
export function useAccountingControl() {
  const { data, isLoading, isError, isFetched, isFetching } = useQuery({
    queryKey: ["accountingControl"] as QueryKeys,
    queryFn: async () => getAccountingControl(),
  });

  return { data, isLoading, isError, isFetched, isFetching };
}

/**
 * Use BillingControl Hook to get Billing Control Details
 */
export function useBillingControl() {
  const { data, isLoading, isError, isFetched, isFetching } = useQuery({
    queryKey: ["billingControl"] as QueryKeys,
    queryFn: async () => getBillingControl(),
  });

  return { data, isLoading, isError, isFetched, isFetching };
}

/**
 * Use InvoiceControl Hook to get Invoice Control Details
 */
export function useInvoiceControl() {
  const { data, isLoading, isError, isFetched, isFetching } = useQuery({
    queryKey: ["invoiceControl"] as QueryKeys,
    queryFn: async () => getInvoiceControl(),
  });

  return { data, isLoading, isError, isFetched, isFetching };
}

/**
 * Use DispatchControl Hook to get Dispatch Control Details
 */
export function useDispatchControl() {
  const { data, isLoading, isError, isFetched, isFetching } = useQuery({
    queryKey: ["dispatchControl"] as QueryKeys,
    queryFn: () => getDispatchControl(),
  });

  return { data, isLoading, isError, isFetched, isFetching };
}

/**
 * Use ShipmentControl hook to get Shipment Control Details
 */
export function useShipmentControl() {
  const { data, isLoading, isError, isFetched, isFetching } = useQuery({
    queryKey: ["shipmentControl"] as QueryKeys,
    queryFn: async () => getShipmentControl(),
  });

  return { data, isLoading, isError, isFetched, isFetching };
}

/**
 * Use RouteControl hook to get Route Control Details
 */
export function useRouteControl() {
  const { data, isLoading, isError, isFetched, isFetching } = useQuery({
    queryKey: ["routeControl"] as QueryKeys,
    queryFn: async () => getRouteControl(),
  });

  return { data, isLoading, isError, isFetched, isFetching };
}

/**
 * Get Customers for select options
 * @param show - show or hide the query
 */
export function useCustomers(show?: boolean) {
  const { data, isLoading, isError, isFetched } = useQuery({
    queryKey: ["customers"] as QueryKeys,
    queryFn: async () => getCustomers(),
    enabled: show,
  });

  const selectCustomersData =
    (data as Customer[])?.map((item: Customer) => ({
      value: item.id,
      label: `${item.code} - ${item.name}`,
    })) || [];

  return { selectCustomersData, isLoading, isError, isFetched };
}

/**
 * Get Document Classifications for select options
 * @param show - show or hide the query
 */
export function useDocumentClass(show?: boolean) {
  const { data, isLoading, isError, isFetched } = useQuery({
    queryKey: ["documentClassifications"] as QueryKeys,
    queryFn: async () => getDocumentClassifications(),
    enabled: show,
  });

  const selectDocumentClassData =
    (data as DocumentClassification[])?.map((item: DocumentClassification) => ({
      value: item.id,
      label: item.code,
    })) || [];

  return { selectDocumentClassData, isLoading, isError, isFetched };
}

/**
 * Get Equipment Types for select options
 * @param show - show or hide the query
 * @param limit - limit the number of results
 */
export function useEquipmentTypes(limit: number = 100, show?: boolean) {
  const { data, isLoading, isError, isFetched } = useQuery({
    queryKey: ["equipmentTypes", limit] as QueryKeyWithParams<
      "equipmentTypes",
      [number]
    >,
    queryFn: async () => getEquipmentTypes(limit),
    enabled: show,
  });

  const selectEquipmentType =
    (data as EquipmentType[])?.map((item: EquipmentType) => ({
      value: item.id,
      label: item.code,
    })) || [];

  return { selectEquipmentType, isLoading, isError, isFetched };
}

/**
 * Get Feasibility Control Details
 */
export function useFeasibilityControl() {
  const { data, isLoading, isError, isFetched, isFetching } = useQuery({
    queryKey: ["feasibilityControl"] as QueryKeys,
    queryFn: async () => getFeasibilityControl(),
  });

  return { data, isLoading, isError, isFetched, isFetching };
}

/**
 * Get Locations for select options
 * @param locationStatus
 * @param show - show or hide the query
 */
export function useLocations(locationStatus: string = "A", show?: boolean) {
  const {
    data: locations,
    isError,
    isLoading,
  } = useQuery({
    queryKey: ["locations", locationStatus] as QueryKeyWithParams<
      "locations",
      [string]
    >,
    queryFn: async () => getLocations(locationStatus),
    enabled: show,
  });

  const selectLocationData =
    (locations as Location[])?.map((location: Location) => ({
      value: location.id,
      label: location.name,
    })) || [];

  return { selectLocationData, isError, isLoading, locations };
}

/**
 * Get Shipment Types for select options
 * @param show - show or hide the query
 * @returns
 */
export function useShipmentTypes(show?: boolean) {
  const { data, isLoading, isError, isFetched } = useQuery({
    queryKey: ["shipmentTypes"] as QueryKeys,
    queryFn: async () => getShipmentTypes(),
    enabled: show,
  });

  const selectShipmentType =
    (data as ShipmentType[])?.map((item: ShipmentType) => ({
      value: item.id,
      label: item.code,
    })) || [];

  return { selectShipmentType, isLoading, isError, isFetched };
}

/**
 * Get Users for select options
 * @param show - show or hide the query
 */
export function useUsers(show?: boolean) {
  /** Get users for the select input */
  const { data, isError, isLoading } = useQuery({
    queryKey: ["users"] as QueryKeys,
    queryFn: async () => getUsers(),
    enabled: show,
  });

  const selectUsersData =
    (data as User[])?.map((user: User) => ({
      value: user.id,
      label: user.name || user.username, // if fullName is null, use username
    })) || [];

  return { selectUsersData, isError, isLoading };
}

/**
 * Get US States for select options
 * @param show - show or hide the query
 * @param limit
 */
export function useUSStates(show?: boolean, limit?: number) {
  const { data, isError, isLoading } = useQuery({
    queryKey: ["usStates", limit] as QueryKeyWithParams<"usStates", [number]>,
    queryFn: async () => getUSStates(limit),
    enabled: show,
  });

  // Create an array of objects with value and label for each state
  const selectUSStates =
    (data as USStates[])?.map((state) => ({
      value: state.id,
      label: state.name,
    })) || [];

  return { selectUSStates, isError, isLoading };
}

/**
 * Get Comment Types for select options
 * @param show - show or hide the query
 */
export function useCommentTypes(show?: boolean) {
  const { data, isError, isLoading } = useQuery({
    queryKey: ["commentTypes"] as QueryKeys,
    queryFn: async () => getCommentTypes(),
    enabled: show,
  });

  const selectCommentTypes =
    (data as CommentType[])?.map((commentType: CommentType) => ({
      value: commentType.id,
      label: commentType.name,
    })) || [];

  return { selectCommentTypes, isError, isLoading };
}

/**
 * Get Workers for select options
 * @param show - show or hide the query
 * @param searchQuery
 * @param fleetCodeId
 * @param limit - limit the number of results
 * @param status - status of the workers.
 * @fleetCodeId - id of the fleet code.
 */
export function useWorkers(
  show?: boolean,
  searchQuery?: string,
  fleetCodeId?: string,
  status: StatusChoiceProps = "Active",
  limit: number = 100,
) {
  const { data, isError, isLoading } = useQuery({
    queryKey: [
      "workers",
      limit,
      status,
      searchQuery,
      fleetCodeId,
    ] as QueryKeyWithParams<"workers", [number, string, string, string]>,
    queryFn: async () => getWorkers(searchQuery, fleetCodeId, limit, status),
    enabled: show,
  });

  const selectWorkers =
    (data as Worker[])?.map((worker: Worker) => ({
      value: worker.id,
      label: worker.code,
    })) || [];

  return { data, selectWorkers, isError, isLoading };
}

export function useEmailProfiles(show?: boolean) {
  const { data, isLoading, isError } = useQuery({
    queryKey: ["emailProfiles"] as QueryKeys,
    queryFn: async () => getEmailProfiles(),
    enabled: show,
  });

  const selectEmailProfile =
    (data as EmailProfile[])?.map((emailProfile: EmailProfile) => ({
      value: emailProfile.id,
      label: `${emailProfile.name} (${emailProfile.email})`,
    })) || [];

  return { selectEmailProfile, data, isError, isLoading };
}

export function useEmailControl() {
  const { data, isLoading, isFetched, isError, isFetching } = useQuery({
    queryKey: ["emailControl"] as QueryKeys,
    queryFn: async () => getEmailControl(),
  });

  return { data, isLoading, isFetched, isError, isFetching };
}

/**
 * Get UserNotifications for notification menu
 * @param userId - user id
 * @param markAsRead
 */
export function useNotifications(userId: string, markAsRead: boolean = false) {
  const { data: notificationsData, isLoading: notificationsLoading } = useQuery(
    {
      queryKey: ["userNotifications", userId, markAsRead],
      queryFn: async () => getUserNotifications(markAsRead),
    },
  );

  return { notificationsData, notificationsLoading };
}

/**
 * Get Feature Flags for Admin Dashbaord.
 */
export function useFeatureFlags() {
  const { data: featureFlagsData, isLoading: featureFlagsLoading } = useQuery({
    queryKey: ["featureFlags"],
    queryFn: async () => getFeatureFlags(),
  });

  return { featureFlagsData, featureFlagsLoading };
}

/**
 * Get the Logged-in Users Organization
 */
export function useOrganization() {
  const { data, isLoading, isError } = useQuery({
    queryKey: ["organization"] as QueryKeys,
    queryFn: async () => getOrganizationDetails(),
  });

  return {
    data,
    isLoading,
    isError,
  };
}

/**
 * Get the Google API information for the organization
 */
export function useGoogleAPI(open?: boolean) {
  const {
    data: googleAPIData,
    isLoading,
    isError,
  } = useQuery({
    queryKey: ["googleAPI"] as QueryKeys,
    queryFn: async () => getGoogleApiInformation(),
    enabled: open,
  });

  return { googleAPIData, isLoading, isError };
}

/**
 * Get Topic Names for select options
 * @param show - show or hide the query
 */
export function useTopics(show?: boolean) {
  const { data, isError, isLoading } = useQuery({
    queryKey: ["topicNames"] as QueryKeys,
    queryFn: async () => getTopicNames(),
    enabled: show,
  });

  const selectTopics =
    (data as Topic[])?.map((table: Topic) => ({
      value: table.value,
      label: table.label,
    })) || [];

  return { selectTopics, isError, isLoading };
}

/**
 * Get the User's Favorites
 * @param show - show or hide the query
 */
export function useUserFavorites(show?: boolean) {
  const { data, isError, isLoading } = useQuery({
    queryKey: ["userFavorites"] as QueryKeys,
    queryFn: async () => getUserFavorites(),
    enabled: show,
  });

  return { data, isError, isLoading };
}

/**
 * Get the Accounting Control for select options
 * @param show - show or hide the query
 */
export function useRevenueCodes(show?: boolean) {
  const {
    data: revenueCodesData,
    isError: isRevenueCodeError,
    isLoading: isRevenueCodeLoading,
  } = useQuery({
    queryKey: ["revenueCodes"] as QueryKeys,
    queryFn: async () => getRevenueCodes(),
    enabled: show,
  });

  const selectRevenueCodes =
    (revenueCodesData as RevenueCode[])?.map((revenueCode: RevenueCode) => ({
      value: revenueCode.id,
      label: revenueCode.code,
    })) || [];

  return { selectRevenueCodes, isRevenueCodeError, isRevenueCodeLoading };
}

/**
 * Get the Trailers for select options
 * @param show - show or hide the query
 * @returns selectTrailers, isTrailerError, isTrailerLoading, trailerData
 */
export function useTrailers(show?: boolean) {
  const {
    data: trailerData,
    isError: isTrailerError,
    isLoading: isTrailerLoading,
  } = useQuery({
    queryKey: ["trailers"] as QueryKeys,
    queryFn: async () => getTrailers(),
    enabled: show,
  });

  const selectTrailers =
    (trailerData as Trailer[])?.map((trailer: Trailer) => ({
      value: trailer.id,
      label: trailer.code,
    })) || [];

  return { selectTrailers, isTrailerError, isTrailerLoading, trailerData };
}

/**
 * Get the next shipment pro number for the organization
 */
export function useNextProNumber() {
  const {
    data: proNumber,
    isError: isProNumberError,
    isLoading: isProNumberLoading,
  } = useQuery({
    queryKey: ["proNumber"],
    queryFn: async () => getNextProNumber(),
  });

  return { proNumber, isProNumberError, isProNumberLoading };
}

export function useLocationAutoComplete(searchQuery: string) {
  const isQueryEnabled = searchQuery.trim().length > 0;

  const {
    data: searchResults = [],
    isError: searchResultError,
    isLoading: isSearchLoading,
  } = useQuery({
    queryKey: ["locationAutoComplete", searchQuery] as QueryKeyWithParams<
      "locationAutoComplete",
      [string]
    >,
    queryFn: async () => searchLocation(searchQuery),
    enabled: isQueryEnabled,
  });

  return { searchResults, searchResultError, isSearchLoading };
}

/**
 * Get the Service Types for select options
 * @returns selectServiceTypes, isServiceTypeError, isServiceTypeLoading
 */
export function useServiceTypes() {
  const {
    data: serviceTypes,
    isError: isServiceTypeError,
    isLoading: isServiceTypeLoading,
  } = useQuery({
    queryKey: ["serviceTypes"] as QueryKeys,
    queryFn: async () => getServiceTypes(),
  });

  const selectServiceTypes =
    (serviceTypes as ServiceType[])?.map((serviceType: ServiceType) => ({
      value: serviceType.id,
      label: serviceType.code,
    })) || [];

  return { selectServiceTypes, isServiceTypeError, isServiceTypeLoading };
}

export function useDailyShipmentCounts(startDate: string, endDate: string) {
  const { data, isError, isLoading, isSuccess, isFetched } = useQuery({
    queryKey: ["dailyShipmentCounts", startDate, endDate] as QueryKeyWithParams<
      "dailyShipmentCounts",
      [string, string]
    >,
    queryFn: async () => getDailyShipmentCounts(startDate, endDate),
  });

  const formattedData = [
    {
      id: "total-shipments",
      data:
        data?.results?.map((item) => ({
          x: item.day,
          y: item.value,
        })) ?? [],
    },
  ];

  return { formattedData, data, isError, isLoading, isSuccess, isFetched };
}

export function useReportColumns(modelName: string, show?: boolean) {
  const { data, isError, isLoading, isFetched, isPending } = useQuery({
    queryKey: ["reportColumns", modelName] as QueryKeyWithParams<
      "reportColumns",
      [string]
    >,
    enabled: show,
    queryFn: async () => getColumns(modelName as string),
    staleTime: 1000 * 60 * 60,
  });

  const selectColumnData = data?.columns?.map((column) => ({
    label: column.label,
    value: column.value,
    description: column.description,
  }));

  const selectRelationshipGroupedOptions =
    data?.relationships?.map((relationship) => {
      const label = `${toTitleCase(relationship.foreignKey)} (${toTitleCase(
        relationship.referencedTable,
      )})`;
      return {
        label: label,
        options: relationship.columns.map((column) => ({
          label: column.label,
          value: `${relationship.foreignKey}.${relationship.referencedTable}.${column.value}`,
          description: column.description,
        })),
      };
    }) ?? [];

  const groupedOptions = [
    { options: selectColumnData, label: "Columns" },
    ...selectRelationshipGroupedOptions,
  ];

  return { groupedOptions, isError, isLoading, isFetched, isPending };
}
