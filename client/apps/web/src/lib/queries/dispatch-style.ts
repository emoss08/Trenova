import { fetchDispatchStyle } from "@/lib/graphql/dispatch-style";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const dispatchStyle = createQueryKeys("dispatchStyle", {
  get: () => ({
    queryKey: ["get"],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchDispatchStyle({ signal }),
  }),
});
