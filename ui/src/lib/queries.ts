import {
  getOrgById,
  getShipmentControl,
  listOrganizations,
} from "@/services/organization";
import { getUsStateOptions, getUsStates } from "@/services/us-state";
import { createQueryKeyStore } from "@lukemorales/query-key-factory";

export const queries = createQueryKeyStore({
  organization: {
    getOrgById: (
      orgId: string,
      includeState: boolean = false,
      includeBu: boolean = false,
    ) => ({
      queryKey: ["organization", orgId, includeState, includeBu],
      queryFn: async () => {
        const response = await getOrgById({
          orgId,
          includeState,
          includeBu,
        });
        return response.data;
      },
    }),
    getUserOrganizations: () => ({
      queryKey: ["organization/user"],
      queryFn: async () => {
        const response = await listOrganizations();
        return response.data;
      },
    }),
    getShipmentControl: () => ({
      queryKey: ["shipmentControl"],
      queryFn: async () => {
        const response = await getShipmentControl();
        return response.data;
      },
    }),
  },
  usState: {
    list: () => ({
      queryKey: ["us-states"],
      queryFn: async () => getUsStates(),
    }),
    options: () => ({
      queryKey: ["us-states/options"],
      queryFn: async () => {
        return await getUsStateOptions();
      },
    }),
  },
});
