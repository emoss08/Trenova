/*
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

import { getUserFavorites } from "@/services/AccountRequestService";
import {
  getAccountingControl,
  getGLAccounts,
  getRevenueCodes,
  getTags,
} from "@/services/AccountingRequestService";
import {
  getAccessorialCharges,
  getDocumentClassifications,
} from "@/services/BillingRequestService";
import {
  getCommodities,
  getHazardousMaterials,
} from "@/services/CommodityRequestService";
import { getCustomers } from "@/services/CustomerRequestService";
import {
  getCommentTypes,
  getFeasibilityControl,
  getFleetCodes,
  getRates,
} from "@/services/DispatchRequestService";
import {
  getEquipmentManufacturers,
  getEquipmentTypes,
  getTrailers,
} from "@/services/EquipmentRequestService";
import {
  getLocationCategories,
  getLocations,
  getUSStates,
  searchLocation,
} from "@/services/LocationRequestService";
import { getShipmentTypes } from "@/services/OrderRequestService";
import {
  getBillingControl,
  getDepots,
  getDispatchControl,
  getEmailControl,
  getEmailProfiles,
  getFeatureFlags,
  getGoogleApiInformation,
  getInvoiceControl,
  getRouteControl,
  getShipmentControl,
  getTableNames,
  getTopicNames,
  getUserOrganizationDetails,
} from "@/services/OrganizationRequestService";
import {
  getFormulaTemplates,
  getNextProNumber,
  validateBOLNumber,
} from "@/services/ShipmentRequestService";
import {
  getUserDetails,
  getUserNotifications,
  getUsers,
} from "@/services/UserRequestService";
import { getWorkers } from "@/services/WorkerRequestService";
import { QueryKeys, QueryKeyWithParams } from "@/types";
import {
  AccountingControl,
  GeneralLedgerAccount,
  RevenueCode,
  Tag,
} from "@/types/accounting";
import { User } from "@/types/accounts";
import {
  AccessorialCharge,
  BillingControl,
  DocumentClassification,
} from "@/types/billing";
import { Commodity, HazardousMaterial } from "@/types/commodities";
import { Customer } from "@/types/customer";
import {
  CommentType,
  DispatchControl,
  FeasibilityToolControl,
  FleetCode,
  Rate,
} from "@/types/dispatch";
import {
  EquipmentClass,
  EquipmentManufacturer,
  EquipmentType,
} from "@/types/equipment";
import { InvoiceControl } from "@/types/invoicing";
import { Location, LocationCategory, USStates } from "@/types/location";
import {
  FormulaTemplate,
  ServiceType,
  ShipmentControl,
  ShipmentType,
} from "@/types/shipment";
import {
  Depot,
  EmailControl,
  EmailProfile,
  Organization,
  TableName,
  Topic,
} from "@/types/organization";
import { RouteControl } from "@/types/route";
import { Worker } from "@/types/worker";
import { useQuery, useQueryClient } from "@tanstack/react-query";

/**
 * Get Tags for select options
 * @param show - show or hide the query
 */
export function useTags(show: boolean) {
  const queryClient = useQueryClient();

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
    initialData: () => {
      return queryClient.getQueryData(["tags"] as QueryKeys);
    },
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
  const queryClient = useQueryClient();

  const {
    data: glAccountsData,
    isLoading,
    isError,
    isFetched,
  } = useQuery({
    queryKey: ["glAccounts"] as QueryKeys,
    queryFn: async () => getGLAccounts(),
    enabled: show,
    initialData: () => queryClient.getQueryData(["glAccounts"] as QueryKeys),
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
 * Get Accessorial Charges for select options
 * @param show - show or hide the query
 */
export function useAccessorialCharges(show?: boolean) {
  const queryClient = useQueryClient();

  const { data, isLoading, isError, isFetched } = useQuery({
    queryKey: ["accessorialCharges"] as QueryKeys,
    queryFn: async () => getAccessorialCharges(),
    enabled: show,
    initialData: () =>
      queryClient.getQueryData(["accessorialCharges"] as QueryKeys),
  });

  const selectAccessorialChargeData =
    (data as AccessorialCharge[])?.map((item: AccessorialCharge) => ({
      value: item.id,
      label: item.code,
    })) || [];

  return { selectAccessorialChargeData, isLoading, isError, isFetched };
}

/**
 * Use Accounting Control Hook to get Accounting Control Details
 */
export function useAccountingControl() {
  const queryClient = useQueryClient();

  const { data, isLoading, isError, isFetched, isFetching } = useQuery({
    queryKey: ["accountingControl"] as QueryKeys,
    queryFn: async () => getAccountingControl(),
    initialData: () =>
      queryClient.getQueryData(["accountingControl"] as QueryKeys),
  });

  const accountingControlData = (data as AccountingControl[])?.[0];

  return { accountingControlData, isLoading, isError, isFetched, isFetching };
}

/**
 * Use BillingControl Hook to get Billing Control Details
 */
export function useBillingControl() {
  const queryClient = useQueryClient();
  const { data, isLoading, isError, isFetched, isFetching } = useQuery({
    queryKey: ["billingControl"] as QueryKeys,
    queryFn: () => getBillingControl(),
    initialData: () =>
      queryClient.getQueryData(["billingControl"] as QueryKeys),
  });

  // Store first element of BillingControlData in variable
  const billingControlData = (data as BillingControl[])?.[0];

  return { billingControlData, isLoading, isError, isFetched, isFetching };
}

/**
 * Use InvoiceControl Hook to get Invoice Control Details
 */
export function useInvoiceControl() {
  const queryClient = useQueryClient();

  const { data, isLoading, isError, isFetched, isFetching } = useQuery({
    queryKey: ["invoiceControl"] as QueryKeys,
    queryFn: () => getInvoiceControl(),
    initialData: () =>
      queryClient.getQueryData(["invoiceControl"] as QueryKeys),
  });

  // Store first element of invoiceControlData in variable
  const invoiceControlData = (data as InvoiceControl[])?.[0];

  return { invoiceControlData, isLoading, isError, isFetched, isFetching };
}

/**
 * Use DispatchControl Hook to get Dispatch Control Details
 */
export function useDispatchControl() {
  const queryClient = useQueryClient();

  const { data, isLoading, isError, isFetched, isFetching } = useQuery({
    queryKey: ["dispatchControl"] as QueryKeys,
    queryFn: () => getDispatchControl(),
    initialData: () =>
      queryClient.getQueryData(["dispatchControl"] as QueryKeys),
    staleTime: Infinity,
  });

  // Store first element of dispatchControlData in variable
  const dispatchControlData = (data as DispatchControl[])?.[0];

  return { dispatchControlData, isLoading, isError, isFetched, isFetching };
}

/**
 * Use ShipmentControl hook to get Shipment Control Details
 */
export function useShipmentControl() {
  const queryClient = useQueryClient();

  const { data, isLoading, isError, isFetched, isFetching } = useQuery({
    queryKey: ["shipmentControl"] as QueryKeys,
    queryFn: () => getShipmentControl(),
    initialData: () =>
      queryClient.getQueryData(["shipmentControl"] as QueryKeys),
  });

  // Store first element of shipmentControlData in variable
  const shipmentControlData = (data as ShipmentControl[])?.[0];

  return { shipmentControlData, isLoading, isError, isFetched, isFetching };
}

/**
 * Use RouteControl hook to get Route Control Details
 */
export function useRouteControl() {
  const queryClient = useQueryClient();

  const { data, isLoading, isError, isFetched, isFetching } = useQuery({
    queryKey: ["routeControl"] as QueryKeys,
    queryFn: () => getRouteControl(),
    initialData: () => queryClient.getQueryData(["routeControl"] as QueryKeys),
  });

  // Store first element of dispatchControlData in variable
  const routeControlData = (data as RouteControl[])?.[0];

  return { routeControlData, isLoading, isError, isFetched, isFetching };
}

/**
 * Get Commodities for select options
 * @param show - show or hide the query
 */
export function useCommodities(show?: boolean) {
  const queryClient = useQueryClient();

  const { data, isLoading, isError, isFetched } = useQuery({
    queryKey: ["commodities"] as QueryKeys,
    queryFn: async () => getCommodities(),
    enabled: show,
    initialData: () => queryClient.getQueryData(["commodities"] as QueryKeys),
  });

  const selectCommodityData =
    (data as Commodity[])?.map((item: Commodity) => ({
      value: item.id,
      label: item.name,
    })) || [];

  return { selectCommodityData, isLoading, isError, isFetched, data };
}

/**
 * Get Customers for select options
 * @param show - show or hide the query
 */
export function useCustomers(show?: boolean) {
  const queryClient = useQueryClient();

  const { data, isLoading, isError, isFetched } = useQuery({
    queryKey: ["customers"] as QueryKeys,
    queryFn: async () => getCustomers(),
    enabled: show,
    initialData: () => queryClient.getQueryData(["customers"] as QueryKeys),
  });

  const selectCustomersData =
    (data as Customer[])?.map((item: Customer) => ({
      value: item.id,
      label: item.name,
    })) || [];

  return { selectCustomersData, isLoading, isError, isFetched };
}

/**
 * Get Document Classifications for select options
 * @param show - show or hide the query
 */
export function useDocumentClass(show?: boolean) {
  const queryClient = useQueryClient();

  const { data, isLoading, isError, isFetched } = useQuery({
    queryKey: ["documentClassifications"] as QueryKeys,
    queryFn: async () => getDocumentClassifications(),
    enabled: show,
    initialData: () =>
      queryClient.getQueryData(["documentClassifications"] as QueryKeys),
  });

  const selectDocumentClassData =
    (data as DocumentClassification[])?.map((item: DocumentClassification) => ({
      value: item.id,
      label: item.name,
    })) || [];

  return { selectDocumentClassData, isLoading, isError, isFetched };
}

/**
 * Get Equipment Types for select options
 * @param equipmentClass - Equipment Class
 * @param show - show or hide the query
 * @param limit - limit the number of results
 */
export function useEquipmentTypes(
  equipmentClass: EquipmentClass,
  limit: number = 100,
  show?: boolean,
) {
  const queryClient = useQueryClient();

  const { data, isLoading, isError, isFetched } = useQuery({
    queryKey: ["equipmentTypes", equipmentClass, limit] as QueryKeyWithParams<
      "equipmentTypes",
      [string, number]
    >,
    queryFn: async () => getEquipmentTypes(equipmentClass, limit),
    enabled: show,
    initialData: () =>
      queryClient.getQueryData(["equipmentTypes"] as QueryKeys),
  });

  const selectEquipmentType =
    (data as EquipmentType[])?.map((item: EquipmentType) => ({
      value: item.id,
      label: item.name,
    })) || [];

  return { selectEquipmentType, isLoading, isError, isFetched };
}

/**
 * Get Feasibility Control Details
 */
export function useFeasibilityControl() {
  const queryClient = useQueryClient();
  const { data, isLoading, isError, isFetched, isFetching } = useQuery({
    queryKey: ["feasibilityControl"] as QueryKeys,
    queryFn: async () => getFeasibilityControl(),
    initialData: () =>
      queryClient.getQueryData(["feasibilityControl"] as QueryKeys),
  });

  const feasibilityControlData = (data as FeasibilityToolControl[])?.[0];

  return { feasibilityControlData, isLoading, isError, isFetched, isFetching };
}

/**
 * Get Hazardous Materials for select options
 * @param show - show or hide the query
 */
export function useHazardousMaterial(show?: boolean) {
  const queryClient = useQueryClient();

  const { data, isLoading, isError, isFetched } = useQuery({
    queryKey: ["hazardousMaterials"] as QueryKeys,
    queryFn: async () => getHazardousMaterials(),
    enabled: show,
    initialData: () =>
      queryClient.getQueryData(["hazardousMaterials"] as QueryKeys),
  });

  const selectHazardousMaterials =
    (data as HazardousMaterial[])?.map((item: HazardousMaterial) => ({
      value: item.id,
      label: item.name,
    })) || [];

  return { selectHazardousMaterials, isLoading, isError, isFetched, data };
}

/**
 * Get Locations for select options
 * @param locationStatus
 * @param show - show or hide the query
 */
export function useLocations(locationStatus: string = "A", show?: boolean) {
  const queryClient = useQueryClient();

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
    initialData: () =>
      queryClient.getQueryData([
        "locations",
        locationStatus,
      ] as QueryKeyWithParams<"locations", [string]>),
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
  const queryClient = useQueryClient();

  const { data, isLoading, isError, isFetched } = useQuery({
    queryKey: ["shipmentTypes"] as QueryKeys,
    queryFn: async () => getShipmentTypes(),
    enabled: show,
    initialData: () => queryClient.getQueryData(["shipmentTypes"] as QueryKeys),
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
  const queryClient = useQueryClient();

  /** Get users for the select input */
  const { data, isError, isLoading } = useQuery({
    queryKey: ["users"] as QueryKeys,
    queryFn: async () => getUsers(),
    enabled: show,
    initialData: () => queryClient.getQueryData(["users"] as QueryKeys),
  });

  const selectUsersData =
    (data as User[])?.map((user: User) => ({
      value: user.id,
      label: user.fullName || user.username, // if fullName is null, use username
    })) || [];

  return { selectUsersData, isError, isLoading };
}

/**
 * Get User Details
 * @param userId - user id
 */
export function useUser(userId: string) {
  const queryClient = useQueryClient();

  return useQuery({
    queryKey: ["users", userId] as QueryKeyWithParams<"users", [string]>,
    queryFn: () => (userId ? getUserDetails(userId) : Promise.resolve(null)),
    initialData: (): User | undefined =>
      queryClient.getQueryData(["users", userId] as QueryKeyWithParams<
        "users",
        [string]
      >),
  });
}

/**
 * Get Location Categories for select options
 * @param show - show or hide the query
 */
export function useLocationCategories(show?: boolean) {
  const queryClient = useQueryClient();

  const { data, isError, isLoading } = useQuery({
    queryKey: ["locationCategories"] as QueryKeys,
    queryFn: async () => getLocationCategories(),
    enabled: show,
    initialData: () =>
      queryClient.getQueryData(["locationCategories"] as QueryKeys),
  });

  const selectLocationCategories =
    (data as LocationCategory[])?.map((location: LocationCategory) => ({
      value: location.id,
      label: location.name,
    })) || [];

  return { selectLocationCategories, isError, isLoading };
}

/**
 * Get US States for select options
 * @param show - show or hide the query
 * @param limit
 */
export function useUSStates(show?: boolean, limit?: number) {
  const queryClient = useQueryClient();

  const { data, isError, isLoading } = useQuery({
    queryKey: ["usStates", limit] as QueryKeyWithParams<"usStates", [number]>,
    queryFn: async () => getUSStates(limit),
    enabled: show,
    initialData: () => queryClient.getQueryData(["usStates"] as QueryKeys),
  });

  // Create an array of objects with value and label for each state
  const selectUSStates =
    (data as USStates[])?.map((state) => ({
      value: state.abbreviation,
      label: state.name,
    })) || [];

  return { selectUSStates, isError, isLoading };
}

/**
 * Get Comment Types for select options
 * @param show - show or hide the query
 */
export function useCommentTypes(show?: boolean) {
  const queryClient = useQueryClient();

  const { data, isError, isLoading } = useQuery({
    queryKey: ["commentTypes"] as QueryKeys,
    queryFn: async () => getCommentTypes(),
    enabled: show,
    initialData: () => queryClient.getQueryData(["commentTypes"] as QueryKeys),
  });

  const selectCommentTypes =
    (data as CommentType[])?.map((commentType: CommentType) => ({
      value: commentType.id,
      label: commentType.name,
    })) || [];

  return { selectCommentTypes, isError, isLoading };
}

/**
 * Get Depots for select options
 * @param show - show or hide the query
 */
export function useDepots(show?: boolean) {
  const queryClient = useQueryClient();

  const { data, isError, isLoading } = useQuery({
    queryKey: ["depots"] as QueryKeys,
    queryFn: async () => getDepots(),
    enabled: show,
    initialData: () => queryClient.getQueryData(["depots"] as QueryKeys),
  });

  const selectDepots =
    (data as Depot[])?.map((depot: Depot) => ({
      value: depot.id,
      label: depot.name,
    })) || [];

  return { selectDepots, isError, isLoading };
}

/**
 * Get Fleet Codes for select options
 * @param show - show or hide the query
 * @param limit - limit the number of results
 */
export function useFleetCodes(show?: boolean, limit: number = 100) {
  const queryClient = useQueryClient();

  const { data, isError, isLoading } = useQuery({
    queryKey: ["fleetCodes", limit] as QueryKeyWithParams<
      "fleetCodes",
      [number]
    >,
    queryFn: async () => getFleetCodes(limit),
    enabled: show,
    initialData: () => queryClient.getQueryData(["fleetCodes"] as QueryKeys),
  });

  const selectFleetCodes =
    (data as FleetCode[])?.map((fleetCode: FleetCode) => ({
      value: fleetCode.id,
      label: fleetCode.code,
    })) || [];

  return { selectFleetCodes, isError, isLoading };
}

/**
 * Get Equipment Manufacturers for select options
 * @param show - show or hide the query
 * @param limit - limit the number of results
 */
export function useEquipManufacturers(show?: boolean, limit: number = 100) {
  const queryClient = useQueryClient();

  const { data, isError, isLoading } = useQuery({
    queryKey: ["equipmentManufacturers", limit] as QueryKeyWithParams<
      "equipmentManufacturers",
      [number]
    >,
    queryFn: async () => getEquipmentManufacturers(limit),
    enabled: show,
    initialData: () =>
      queryClient.getQueryData(["equipmentManufacturers"] as QueryKeys),
  });

  const selectEquipManufacturers =
    (data as EquipmentManufacturer[])?.map(
      (equipManufacturer: EquipmentManufacturer) => ({
        value: equipManufacturer.id,
        label: equipManufacturer.name,
      }),
    ) || [];

  return { selectEquipManufacturers, isError, isLoading };
}

/**
 * Get Workers for select options
 * @param show - show or hide the query
 * @param limit - limit the number of results
 */
export function useWorkers(show?: boolean, limit: number = 100) {
  const queryClient = useQueryClient();

  const { data, isError, isLoading } = useQuery({
    queryKey: ["workers", limit] as QueryKeyWithParams<"workers", [number]>,
    queryFn: async () => getWorkers(limit),
    enabled: show,
    initialData: () => queryClient.getQueryData(["workers"] as QueryKeys),
  });

  const selectWorkers =
    (data as Worker[])?.map((worker: Worker) => ({
      value: worker.id,
      label: worker.code,
    })) || [];

  return { data, selectWorkers, isError, isLoading };
}

export function useEmailProfiles(show?: boolean) {
  const queryClient = useQueryClient();

  const { data, isLoading, isError } = useQuery({
    queryKey: ["emailProfiles"] as QueryKeys,
    queryFn: () => getEmailProfiles(),
    enabled: show,
    initialData: () => queryClient.getQueryData(["emailProfiles"] as QueryKeys),
  });

  const selectEmailProfile =
    (data as EmailProfile[])?.map((emailProfile: EmailProfile) => ({
      value: emailProfile.id,
      label: `${emailProfile.name} (${emailProfile.email})`,
    })) || [];

  return { selectEmailProfile, data, isError, isLoading };
}

export function useEmailControl() {
  const queryClient = useQueryClient();

  const { data, isLoading, isFetched, isError, isFetching } = useQuery({
    queryKey: ["emailControl"] as QueryKeys,
    queryFn: () => getEmailControl(),
    initialData: () => queryClient.getQueryData(["emailControl"] as QueryKeys),
  });

  const emailControlData = (data as EmailControl[])?.[0];

  return { emailControlData, isLoading, isFetched, isError, isFetching };
}

/**
 * Get UserNotifications for notification menu
 * @param userId - user id
 */
export function useNotifications(userId: string) {
  const queryClient = useQueryClient();

  const { data: notificationsData, isLoading: notificationsLoading } = useQuery(
    {
      queryKey: ["userNotifications", userId],
      queryFn: async () => getUserNotifications(),
      initialData: () => {
        return queryClient.getQueryData(["userNotifications", userId]);
      },
    },
  );

  return { notificationsData, notificationsLoading };
}

/**
 * Get Feature Flags for Admin Dashbaord.
 */
export function useFeatureFlags() {
  const queryClient = useQueryClient();

  const { data: featureFlagsData, isLoading: featureFlagsLoading } = useQuery({
    queryKey: ["featureFlags"],
    queryFn: async () => getFeatureFlags(),
    initialData: () => {
      return queryClient.getQueryData(["featureFlags"]);
    },
  });

  return { featureFlagsData, featureFlagsLoading };
}

/**
 * Get the Logged-in Users Organization
 */
export function useUserOrganization() {
  const queryClient = useQueryClient();
  const {
    data: userOrganizationData,
    isLoading: userOrganizationLoading,
    isError: userOrganizationError,
  } = useQuery({
    queryKey: ["userOrganization"] as QueryKeys,
    queryFn: async () => getUserOrganizationDetails(),
    initialData: (): Organization | undefined =>
      queryClient.getQueryData(["userOrganization"]),
  });

  return {
    userOrganizationData,
    userOrganizationLoading,
    userOrganizationError,
  };
}

/**
 * Get the Google API information for the organization
 */
export function useGoogleAPI() {
  const queryClient = useQueryClient();

  const {
    data: googleAPIData,
    isLoading,
    isError,
  } = useQuery({
    queryKey: ["googleAPI"] as QueryKeys,
    queryFn: async () => getGoogleApiInformation(),
    initialData: () => {
      return queryClient.getQueryData(["googleAPI"] as QueryKeys);
    },
  });

  return { googleAPIData, isLoading, isError };
}

/**
 * Get Table Names for select options
 * @param show - show or hide the query
 */
export function useTableNames(show?: boolean) {
  const queryClient = useQueryClient();

  const { data, isError, isLoading } = useQuery({
    queryKey: ["tableNames"] as QueryKeys,
    queryFn: async () => getTableNames(),
    enabled: show,
    initialData: () => queryClient.getQueryData(["tableNames"] as QueryKeys),
  });

  const selectTableNames =
    (data as TableName[])?.map((table: TableName) => ({
      value: table.value,
      label: table.label,
    })) || [];

  return { selectTableNames, isError, isLoading };
}

/**
 * Get Topic Names for select options
 * @param show - show or hide the query
 */
export function useTopics(show?: boolean) {
  const queryClient = useQueryClient();

  const { data, isError, isLoading } = useQuery({
    queryKey: ["topicNames"] as QueryKeys,
    queryFn: async () => getTopicNames(),
    enabled: show,
    initialData: () => queryClient.getQueryData(["topicNames"] as QueryKeys),
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
  const queryClient = useQueryClient();

  const { data, isError, isLoading } = useQuery({
    queryKey: ["userFavorites"] as QueryKeys,
    queryFn: async () => getUserFavorites(),
    enabled: show,
    initialData: () => queryClient.getQueryData(["userFavorites"] as QueryKeys),
  });

  return { data, isError, isLoading };
}

/**
 * Get the Accounting Control for select options
 * @param show - show or hide the query
 */
export function useRevenueCodes(show?: boolean) {
  const queryClient = useQueryClient();

  const {
    data: revenueCodesData,
    isError: isRevenueCodeError,
    isLoading: isRevenueCodeLoading,
  } = useQuery({
    queryKey: ["revenueCodes"] as QueryKeys,
    queryFn: async () => getRevenueCodes(),
    enabled: show,
    initialData: () => queryClient.getQueryData(["revenueCodes"] as QueryKeys),
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
  const queryClient = useQueryClient();

  const {
    data: trailerData,
    isError: isTrailerError,
    isLoading: isTrailerLoading,
  } = useQuery({
    queryKey: ["trailers"] as QueryKeys,
    queryFn: async () => getTrailers(),
    enabled: show,
    initialData: () => queryClient.getQueryData(["trailers"] as QueryKeys),
  });

  const selectTrailers =
    (trailerData as RevenueCode[])?.map((trailer: RevenueCode) => ({
      value: trailer.id,
      label: trailer.code,
    })) || [];

  return { selectTrailers, isTrailerError, isTrailerLoading, trailerData };
}

/**
 * Get the next shipment pro number for the organization
 */
export function useNextProNumber() {
  const queryClient = useQueryClient();

  const {
    data: proNumber,
    isError: isProNumberError,
    isLoading: isProNumberLoading,
  } = useQuery({
    queryKey: ["proNumber"],
    queryFn: async () => getNextProNumber(),
    initialData: () => queryClient.getQueryData(["proNumber"]),
  });

  return { proNumber, isProNumberError, isProNumberLoading };
}

export function useLocationAutoComplete(searchQuery: string) {
  const queryClient = useQueryClient();
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
    initialData: () =>
      queryClient.getQueryData([
        "locationAutoComplete",
        searchQuery,
      ] as QueryKeyWithParams<"locationAutoComplete", [string]>),
  });

  return { searchResults, searchResultError, isSearchLoading };
}

/**
 * Get the Rates for select options
 * @param limit - limit the number of results
 * @param show - show or hide the query
 * @returns selectRates, isRateError, isRatesLoading, ratesData
 */
export function useRates(limit?: number, show?: boolean) {
  const queryClient = useQueryClient();

  const {
    data: ratesData,
    isError: isRateError,
    isLoading: isRatesLoading,
  } = useQuery({
    queryKey: ["rates", limit] as QueryKeyWithParams<"rates", [number]>,
    queryFn: async () => getRates(limit),
    enabled: show,
    initialData: () => queryClient.getQueryData(["rates"] as QueryKeys),
  });

  const selectRates =
    (ratesData as Rate[])?.map((rate: Rate) => ({
      value: rate.id,
      label: rate.rateNumber,
    })) || [];

  return { selectRates, isRateError, isRatesLoading, ratesData };
}

/**
 * Get the Formula Templates for select options
 * @returns selectFormulaTemplates, isFormulaError, isFormulaLoading
 */
export function useFormulaTemplates() {
  const queryClient = useQueryClient();

  const {
    data: formulaTemplates,
    isError: isFormulaError,
    isLoading: isFormulaLoading,
  } = useQuery({
    queryKey: ["formulaTemplates"],
    queryFn: async () => getFormulaTemplates(),
    initialData: () => queryClient.getQueryData(["formulaTemplates"]),
  });

  const selectFormulaTemplates =
    (formulaTemplates as FormulaTemplate[])?.map(
      (template: FormulaTemplate) => ({
        value: template.id,
        label: template.name,
      }),
    ) || [];

  return { selectFormulaTemplates, isFormulaError, isFormulaLoading };
}

/**
 * Get the Service Types for select options
 * @returns selectServiceTypes, isServiceTypeError, isServiceTypeLoading
 */
export function useServiceTypes() {
  const queryCLient = useQueryClient();

  const {
    data: serviceTypes,
    isError: isServiceTypeError,
    isLoading: isServiceTypeLoading,
  } = useQuery({
    queryKey: ["serviceTypes"],
    queryFn: async () => getShipmentTypes(),
    initialData: () => queryCLient.getQueryData(["serviceTypes"] as QueryKeys),
  });

  const selectServiceTypes =
    (serviceTypes as ServiceType[])?.map((serviceType: ServiceType) => ({
      value: serviceType.id,
      label: serviceType.code,
    })) || [];

  return { selectServiceTypes, isServiceTypeError, isServiceTypeLoading };
}

export function useValidateBOLNumber(bol_number: string) {
  const queryClient = useQueryClient();

  const { data, isError, isLoading } = useQuery({
    queryKey: ["validateBOLNumber", bol_number] as QueryKeyWithParams<
      "validateBOLNumber",
      [string]
    >,
    queryFn: async () => validateBOLNumber(bol_number),
    initialData: () =>
      queryClient.getQueryData([
        "validateBOLNumber",
        bol_number,
      ] as QueryKeyWithParams<"validateBOLNumber", [string]>),
  });

  return { data, isError, isLoading };
}
