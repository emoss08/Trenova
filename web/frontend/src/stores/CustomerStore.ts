import { createGlobalStore } from "@/lib/useGlobalStore";

type customerStoreProps = {
  editModalOpen: boolean;
  activeTab: string | null;
  createRuleProfileModalOpen: boolean;
};

export const customerStore = createGlobalStore<customerStoreProps>({
  editModalOpen: false,
  createRuleProfileModalOpen: false,
  activeTab: "overview",
});

type CustomerFormStore = {
  activeTab: string;
};

export const useCustomerFormStore = createGlobalStore<CustomerFormStore>({
  activeTab: "info",
});
