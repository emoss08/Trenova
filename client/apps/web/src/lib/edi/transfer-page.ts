import { apiService } from "@/services/api";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { transferDirection } from "./transfer-sides";

/**
 * Where a tender's record link should have landed when the tender is one the
 * reader's organization sent. A tender is one record whichever way it went,
 * and its record link opens the inbound page; a tender we sent is listed on
 * the outbound page, so the link is carried there with everything it asked
 * for. Null when the tender belongs where the link put it, when the link names
 * none, or when it cannot be read, in which case the inbound page says what it
 * can.
 */
export async function sentTransferPath(url: URL, signal?: AbortSignal): Promise<string | null> {
  const id = url.searchParams.get("panelEntityId");
  if (!id) return null;
  try {
    const transfer = await apiService.ediService.getTransfer(id, { signal });
    const organizationId = useAuthStore.getState().user?.currentOrganizationId;
    return transferDirection(transfer, organizationId) === "outbound"
      ? `/edi/transfers/outbound${url.search}`
      : null;
  } catch {
    return null;
  }
}
