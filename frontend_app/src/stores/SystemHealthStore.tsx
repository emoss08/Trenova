import { create } from "zustand";
import createRequestFactory from "../factory/RequestFactory";

interface SystemHealthData {
  [key: string]: any;
}
interface SystemHealthStore {
  loading: boolean;
  serviceData: SystemHealthData | null;
  fetchData: () => void;
}

export const useSystemHealthStore = create<SystemHealthStore>((set) => ({
  loading: true,
  serviceData: null,
  fetchData: createRequestFactory(
    "http://127.0.0.1:8000/api/system_health/",
    (data) => set({ serviceData: data }),
    (loading) => set({ loading })
  ),
}));
