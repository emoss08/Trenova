/*
 * COPYRIGHT(c) 2023 MONTA
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
import { getEquipmentTypes } from "@/services/EquipmentRequestService";
import {
  getLocationCategories,
  getLocations,
  getUSStates,
} from "@/services/LocationRequestService";
import { getShipmentTypes } from "@/services/OrderRequestService";
import { getDepots } from "@/services/OrganizationRequestService";
import { getUserDetails, getUsers } from "@/services/UserRequestService";
import { QueryKeys } from "@/types";
import {
  AccountingControl,
  GeneralLedgerAccount,
  Tag,
} from "@/types/accounting";
import { User } from "@/types/accounts";
import { AccessorialCharge, DocumentClassification } from "@/types/billing";
import { Commodity, HazardousMaterial } from "@/types/commodities";
import { Customer } from "@/types/customer";
import { CommentType, FleetCode } from "@/types/dispatch";
import { EquipmentType } from "@/types/equipment";
import { Location, LocationCategory, USStates } from "@/types/location";
import { ShipmentType } from "@/types/order";
import { Depot } from "@/types/organization";
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
 * Get Accounting Control for select options
 */
export function useAccountingControl() {
  const queryClient = useQueryClient();

  const { data, isLoading, isError, isFetched } = useQuery({
    queryKey: ["accountingControl"] as QueryKeys[],
    queryFn: async () => getAccountingControl(),
    initialData: () =>
      queryClient.getQueryData(["accountingControl"] as QueryKeys[]),
    staleTime: Infinity,
    refetchOnWindowFocus: false,
  });

  const dataArray = (data as AccountingControl[])?.[0];

  return { dataArray, isLoading, isError, isFetched };
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

  return useQuery({
    queryKey: ["feasibilityControl"] as QueryKeys[],
    queryFn: async () => getFeasibilityControl(),
    initialData: () =>
      queryClient.getQueryData(["feasibilityControl"] as QueryKeys[]),
    staleTime: Infinity,
    retry: false,
    refetchOnWindowFocus: false,
  });
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
    queryKey: ["user", userId] as QueryKeys[],
    queryFn: () => (userId ? getUserDetails(userId) : Promise.resolve(null)),
    initialData: () =>
      queryClient.getQueryData(["user", userId] as QueryKeys[]),
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
