import { isCaptureRecordKind, type CaptureRecordKind } from "@/lib/capture";
import { capture } from "@/lib/queries/capture";
import { useQuery } from "@tanstack/react-query";

/**
 * The kind of record capture can file onto, when this record is one and the
 * person may scan and print into it. Null otherwise, including while access
 * is still loading, so nothing capture-related shows until it is known.
 */
export function useCaptureKind(resourceType: string): CaptureRecordKind | null {
  const kind = isCaptureRecordKind(resourceType) ? resourceType : null;
  const accessQuery = useQuery({
    ...capture.access(),
    staleTime: 5 * 60 * 1000,
    enabled: kind !== null,
  });
  const access = accessQuery.data;
  if (kind === null || access === undefined || !access.enabled || !access.canCapture) {
    return null;
  }
  return kind;
}
