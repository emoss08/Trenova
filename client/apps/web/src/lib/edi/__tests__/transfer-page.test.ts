import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const getTransfer = vi.hoisted(() => vi.fn());

vi.mock("@/services/api", () => ({
  apiService: { ediService: { getTransfer } },
}));

const { sentTransferPath } = await import("../transfer-page");
const { transfer } =
  await import("@/routes/desk/_components/artifacts/cards/edi-transfer/__tests__/fixtures");

const link = (id: string) =>
  new URL(`http://app/edi/transfers/inbound?panelType=edit&panelEntityId=${id}`);

beforeEach(() => {
  getTransfer.mockReset();
  useAuthStore.setState({
    user: { currentOrganizationId: "org_1" } as ReturnType<typeof useAuthStore.getState>["user"],
  });
});

afterEach(() => {
  useAuthStore.setState({ user: null });
});

describe("sentTransferPath", () => {
  it("leaves a tender we received on the inbound page", async () => {
    getTransfer.mockResolvedValue(transfer("mappingRequired"));

    await expect(sentTransferPath(link("edilt_acme_88241"))).resolves.toBeNull();
    expect(getTransfer).toHaveBeenCalledWith("edilt_acme_88241", { signal: undefined });
  });

  it("moves a tender we sent to the outbound page, keeping what the link asked for", async () => {
    getTransfer.mockResolvedValue(transfer("outbound"));

    await expect(sentTransferPath(link("edilt_s2610_1366"))).resolves.toBe(
      "/edi/transfers/outbound?panelType=edit&panelEntityId=edilt_s2610_1366",
    );
  });

  it("leaves a tender that cannot be read where the link put it", async () => {
    getTransfer.mockRejectedValue(new Error("gone"));

    await expect(sentTransferPath(link("edilt_gone"))).resolves.toBeNull();
  });

  it("asks for nothing when the link names no tender", async () => {
    await expect(sentTransferPath(new URL("http://app/edi/transfers/inbound"))).resolves.toBeNull();
    expect(getTransfer).not.toHaveBeenCalled();
  });
});
