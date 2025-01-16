import { createGlobalStore } from "@/hooks/use-global-store";
import { Organization } from "@/types/organization";

type OrganizationStoreProps = {
  organization: Organization | null;
};

export const useOrganizationStore = createGlobalStore<OrganizationStoreProps>({
  organization: null,
});
