import { createGlobalStore } from "@/hooks/use-global-store";
import { Container } from "@/types/docker";

type ContainerLogStoreProps = {
  showAll: boolean;
  selectedContainer: Container | null;
  showLogs: string | null;
  searchTerm: string;
  tail: string;
  autoRefresh: boolean;
  follow: boolean;
  wrap: boolean;
};

export const useContainerLogStore = createGlobalStore<ContainerLogStoreProps>({
  showAll: false,
  selectedContainer: null,
  showLogs: null,
  searchTerm: "",
  tail: "100",
  autoRefresh: true,
  follow: true,
  wrap: true,
});
