import {
  fetchPageAccess,
  fetchPageAccessPreview,
  fetchPageAccessScreens,
  fetchRoleResourceGuides,
  type PageAccessSubject,
} from "@/lib/graphql/page-access";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const pageAccess = createQueryKeys("pageAccess", {
  forCaller: (location: string) => ({
    queryKey: [location],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchPageAccess(location, { signal }),
  }),
  preview: (location: string, subject: PageAccessSubject) => ({
    queryKey: [location, subject],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      fetchPageAccessPreview(location, subject, { signal }),
  }),
  screens: () => ({
    queryKey: ["screens"],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchPageAccessScreens({ signal }),
  }),
  // The server answers by resource whatever order they arrive in, so the
  // same set shares one entry.
  guides: (resources: readonly string[], maxSensitivity: string) => ({
    queryKey: [[...resources].sort().join(","), maxSensitivity],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      fetchRoleResourceGuides(resources, maxSensitivity, { signal }),
  }),
});
