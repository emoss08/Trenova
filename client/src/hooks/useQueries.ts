/*
 * COPYRIGHT(c) 2024 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
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

import {
  getAccountingControl,
  getGLAccounts,
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
} from "@/services/DispatchRequestService";
import {
  getEquipmentManufacturers,
  getEquipmentTypes,
} from "@/services/EquipmentRequestService";
import {
  getLocationCategories,
  getLocations,
  getUSStates,
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
  getUserOrganizationDetails,
} from "@/services/OrganizationRequestService";
import {
  getUserDetails,
  getUserNotifications,
  getUsers,
} from "@/services/UserRequestService";
import { getWorkers } from "@/services/WorkerRequestService";
import { QueryKeys } from "@/types";
import {
  AccountingControl,
  GeneralLedgerAccount,
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
} from "@/types/dispatch";
import { EquipmentManufacturer, EquipmentType } from "@/types/equipment";
import { InvoiceControl } from "@/types/invoicing";
import { Location, LocationCategory, USStates } from "@/types/location";
import { ShipmentControl, ShipmentType } from "@/types/order";
import {
  Depot,
  EmailControl,
  EmailProfile,
  Organization,
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
    queryKey: ["tags"] as QueryKeys[],
    queryFn: async () => getTags(),
    enabled: show,
    initialData: () => {
      return queryClient.getQueryData(["tags"] as QueryKeys[]);
    },
    staleTime: Infinity,
    retry: false,
    refetchOnWindowFocus: false,
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
    queryKey: ["glAccounts"] as QueryKeys[],
    queryFn: async () => getGLAccounts(),
    enabled: show,
    initialData: () => queryClient.getQueryData(["glAccounts"] as QueryKeys[]),
    staleTime: Infinity,
    retry: false,
    refetchOnWindowFocus: false,
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
    queryKey: ["accessorialCharges"] as QueryKeys[],
    queryFn: async () => getAccessorialCharges(),
    enabled: show,
    initialData: () =>
      queryClient.getQueryData(["accessorialCharges"] as QueryKeys[]),
    staleTime: Infinity,
    retry: false,
    refetchOnWindowFocus: false,
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
    queryKey: ["accountingControl"] as QueryKeys[],
    queryFn: async () => getAccountingControl(),
    initialData: () =>
      queryClient.getQueryData(["accountingControl"] as QueryKeys[]),
    staleTime: Infinity,
    refetchOnWindowFocus: false,
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
    queryKey: ["billingControl"] as QueryKeys[],
    queryFn: () => getBillingControl(),
    initialData: () =>
      queryClient.getQueryData(["billingControl"] as QueryKeys[]),
    staleTime: Infinity,
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
    queryKey: ["invoiceControl"] as QueryKeys[],
    queryFn: () => getInvoiceControl(),
    initialData: () =>
      queryClient.getQueryData(["invoiceControl"] as QueryKeys[]),
    staleTime: Infinity,
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
    queryKey: ["dispatchControl"] as QueryKeys[],
    queryFn: () => getDispatchControl(),
    initialData: () =>
      queryClient.getQueryData(["dispatchControl"] as QueryKeys[]),
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
    queryKey: ["shipmentControl"] as QueryKeys[],
    queryFn: () => getShipmentControl(),
    initialData: () =>
      queryClient.getQueryData(["shipmentControl"] as QueryKeys[]),
    staleTime: Infinity,
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
    queryKey: ["routeControl"] as QueryKeys[],
    queryFn: () => getRouteControl(),
    initialData: () =>
      queryClient.getQueryData(["routeControl"] as QueryKeys[]),
    staleTime: Infinity,
  });

  // Store first element of dispatchControlData in variable
  const routeControlData = (data as RouteControl[])?.[0];

  return { routeControlData, isLoading, isError, isFetched, isFetching };
}

/**
 * Get Commodities for select options
 * @param show - show or hide the query
 */
export function useCommodities(show: boolean) {
  const queryClient = useQueryClient();

  const { data, isLoading, isError, isFetched } = useQuery({
    queryKey: ["commodities"] as QueryKeys[],
    queryFn: async () => getCommodities(),
    enabled: show,
    initialData: () => queryClient.getQueryData(["commodities"] as QueryKeys[]),
    staleTime: Infinity,
    retry: false,
    refetchOnWindowFocus: false,
  });

  const selectCommodityData =
    (data as Commodity[])?.map((item: Commodity) => ({
      value: item.id,
      label: item.name,
    })) || [];

  return { selectCommodityData, isLoading, isError, isFetched };
}

/**
 * Get Customers for select options
 * @param show - show or hide the query
 */
export function useCustomers(show: boolean) {
  const queryClient = useQueryClient();

  const { data, isLoading, isError, isFetched } = useQuery({
    queryKey: ["customers"] as QueryKeys[],
    queryFn: async () => getCustomers(),
    enabled: show,
    initialData: () => queryClient.getQueryData(["customers"] as QueryKeys[]),
    staleTime: Infinity,
    retry: false,
    refetchOnWindowFocus: false,
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
    queryKey: ["documentClassifications"] as QueryKeys[],
    queryFn: async () => getDocumentClassifications(),
    enabled: show,
    initialData: () =>
      queryClient.getQueryData(["documentClassifications"] as QueryKeys[]),
    staleTime: Infinity,
    retry: false,
    refetchOnWindowFocus: false,
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
 * @param show - show or hide the query
 * @param limit - limit the number of results
 */
export function useEquipmentTypes(show?: boolean, limit: number = 100) {
  const queryClient = useQueryClient();

  const { data, isLoading, isError, isFetched } = useQuery({
    queryKey: ["equipmentTypes", limit] as QueryKeys[],
    queryFn: async () => getEquipmentTypes(limit),
    enabled: show,
    initialData: () =>
      queryClient.getQueryData(["equipmentTypes"] as QueryKeys[]),
    staleTime: Infinity,
    retry: false,
    refetchOnWindowFocus: false,
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
    queryKey: ["feasibilityControl"] as QueryKeys[],
    queryFn: async () => getFeasibilityControl(),
    initialData: () =>
      queryClient.getQueryData(["feasibilityControl"] as QueryKeys[]),
    staleTime: Infinity,
    retry: false,
    refetchOnWindowFocus: false,
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
    queryKey: ["hazardousMaterials"] as QueryKeys[],
    queryFn: async () => getHazardousMaterials(),
    enabled: show,
    initialData: () =>
      queryClient.getQueryData(["hazardousMaterials"] as QueryKeys[]),
    staleTime: Infinity,
    retry: false,
    refetchOnWindowFocus: false,
  });

  const selectHazardousMaterials =
    (data as HazardousMaterial[])?.map((item: HazardousMaterial) => ({
      value: item.id,
      label: item.name,
    })) || [];

  return { selectHazardousMaterials, isLoading, isError, isFetched };
}

/**
 * Get Locations for select options
 * @param show - show or hide the query
 */
export function useLocations(show?: boolean) {
  const queryClient = useQueryClient();

  const { data, isError, isLoading } = useQuery({
    queryKey: ["locations"] as QueryKeys[],
    queryFn: async () => getLocations(),
    enabled: show,
    initialData: () => queryClient.getQueryData(["locations"] as QueryKeys[]),
    staleTime: Infinity,
  });

  const selectLocationData =
    (data as Location[])?.map((location: Location) => ({
      value: location.id,
      label: location.code,
    })) || [];

  return { selectLocationData, isError, isLoading };
}

/**
 * Get Shipment Types for select options
 * @param show - show or hide the query
 * @returns
 */
export function useShipmentTypes(show: boolean) {
  const queryClient = useQueryClient();

  const { data, isLoading, isError, isFetched } = useQuery({
    queryKey: ["shipmentTypes"] as QueryKeys[],
    queryFn: async () => getShipmentTypes(),
    enabled: show,
    initialData: () =>
      queryClient.getQueryData(["shipmentTypes"] as QueryKeys[]),
    staleTime: Infinity,
    retry: false,
    refetchOnWindowFocus: false,
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
    queryKey: ["users"] as QueryKeys[],
    queryFn: async () => getUsers(),
    enabled: show,
    initialData: () => queryClient.getQueryData(["users"] as QueryKeys[]),
    staleTime: Infinity,
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
    queryKey: ["users", userId] as QueryKeys[],
    queryFn: () => (userId ? getUserDetails(userId) : Promise.resolve(null)),
    initialData: (): User | undefined =>
      queryClient.getQueryData(["users", userId] as QueryKeys[]),
    staleTime: Infinity,
  });
}

/**
 * Get Location Categories for select options
 * @param show - show or hide the query
 */
export function useLocationCategories(show?: boolean) {
  const queryClient = useQueryClient();

  const { data, isError, isLoading } = useQuery({
    queryKey: ["locationCategories"] as QueryKeys[],
    queryFn: async () => getLocationCategories(),
    enabled: show,
    initialData: () =>
      queryClient.getQueryData(["locationCategories"] as QueryKeys[]),
    refetchOnWindowFocus: true,
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
    queryKey: ["usStates", limit] as QueryKeys[],
    queryFn: async () => getUSStates(limit),
    enabled: show,
    initialData: () => queryClient.getQueryData(["usStates"] as QueryKeys[]),
    staleTime: Infinity,
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
    queryKey: ["commentTypes"] as QueryKeys[],
    queryFn: async () => getCommentTypes(),
    enabled: show,
    initialData: () =>
      queryClient.getQueryData(["commentTypes"] as QueryKeys[]),
    staleTime: Infinity,
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
    queryKey: ["depots"] as QueryKeys[],
    queryFn: async () => getDepots(),
    enabled: show,
    initialData: () => queryClient.getQueryData(["depots"] as QueryKeys[]),
    staleTime: Infinity,
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
    queryKey: ["fleetCodes", limit] as QueryKeys[],
    queryFn: async () => getFleetCodes(limit),
    enabled: show,
    initialData: () => queryClient.getQueryData(["fleetCodes"] as QueryKeys[]),
    staleTime: Infinity,
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
    queryKey: ["equipmentManufacturers", limit] as QueryKeys[],
    queryFn: async () => getEquipmentManufacturers(limit),
    enabled: show,
    initialData: () =>
      queryClient.getQueryData(["equipmentManufacturers"] as QueryKeys[]),
    staleTime: Infinity,
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
    queryKey: ["workers", limit] as QueryKeys[],
    queryFn: async () => getWorkers(limit),
    enabled: show,
    initialData: () => queryClient.getQueryData(["workers"] as QueryKeys[]),
    staleTime: Infinity,
  });

  const selectWorkers =
    (data as Worker[])?.map((worker: Worker) => ({
      value: worker.id,
      label: worker.code,
    })) || [];

  return { data, selectWorkers, isError, isLoading };
}

export function useEmailProfiles() {
  const queryClient = useQueryClient();

  const { data, isLoading, isError } = useQuery({
    queryKey: ["emailProfiles"] as QueryKeys[],
    queryFn: () => getEmailProfiles(),
    initialData: () =>
      queryClient.getQueryData(["emailProfiles"] as QueryKeys[]),
    staleTime: Infinity,
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
    queryKey: ["emailControl"] as QueryKeys[],
    queryFn: () => getEmailControl(),
    initialData: () =>
      queryClient.getQueryData(["emailControl"] as QueryKeys[]),
    staleTime: Infinity,
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
      staleTime: Infinity,
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
    staleTime: Infinity,
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
    queryKey: ["userOrganization"] as QueryKeys[],
    queryFn: async () => getUserOrganizationDetails(),
    initialData: (): Organization | undefined =>
      queryClient.getQueryData(["userOrganization"]),
    staleTime: Infinity,
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
    queryKey: ["googleAPI"] as QueryKeys[],
    queryFn: async () => getGoogleApiInformation(),
    initialData: () => {
      return queryClient.getQueryData(["googleAPI"] as QueryKeys[]);
    },
    staleTime: Infinity,
  });

  return { googleAPIData, isLoading, isError };
}
