import { TableConfigurationDetailDocument } from "@trenova/graphql/generated/graphql";
import { toast } from "@trenova/shared/components/ui/toast";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { TableConfigurationService } from "../table-configuration";

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: vi.fn(),
}));

vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: { error: vi.fn() },
}));

const requestGraphQLMock = vi.mocked(requestGraphQL);
const toastErrorMock = vi.mocked(toast.error);

type Call = {
  document: unknown;
  operationName: string;
  signal?: AbortSignal;
  variables: Record<string, unknown>;
};

function lastCall(): Call {
  return requestGraphQLMock.mock.calls.at(-1)?.[0] as unknown as Call;
}

beforeEach(() => {
  requestGraphQLMock.mockReset();
  toastErrorMock.mockReset();
});

describe("TableConfigurationService.get", () => {
  it("reads the view by id and fills the stored config's missing settings", async () => {
    requestGraphQLMock.mockResolvedValue({
      tableConfiguration: {
        id: "tc_1",
        organizationId: "org_1",
        businessUnitId: "bu_1",
        userId: "usr_1",
        name: "Late loads",
        description: "",
        resource: "shipment",
        tableConfig: { pageSize: 25, columnPinning: null, density: "compact" },
        visibility: "Shared",
        isDefault: false,
        isOrgDefault: null,
        version: 3,
        createdAt: 1_700_000_000,
        updatedAt: 1_700_000_100,
        user: null,
      },
    } as never);
    const controller = new AbortController();

    const view = await new TableConfigurationService().get("tc_1", { signal: controller.signal });

    expect(view).toEqual({
      id: "tc_1",
      organizationId: "org_1",
      businessUnitId: "bu_1",
      userId: "usr_1",
      name: "Late loads",
      description: "",
      resource: "shipment",
      tableConfig: {
        fieldFilters: [],
        filterGroups: [],
        joinOperator: "and",
        sort: [],
        pageSize: 25,
        columnVisibility: {},
        columnOrder: [],
        columnSizing: {},
        columnPinning: { left: [], right: [] },
        density: "compact",
        formatRules: [],
      },
      visibility: "Shared",
      isDefault: false,
      isOrgDefault: false,
      version: 3,
      createdAt: 1_700_000_000,
      updatedAt: 1_700_000_100,
      user: null,
    });
    const call = lastCall();
    expect(call.document).toBe(TableConfigurationDetailDocument);
    expect(call.operationName).toBe("TableConfigurationDetail");
    expect(call.variables).toEqual({ id: "tc_1" });
    expect(call.signal).toBe(controller.signal);
  });

  it("is null, without a parse-failure toast, for a view that was deleted or is no longer readable", async () => {
    requestGraphQLMock.mockResolvedValue({ tableConfiguration: null } as never);

    await expect(new TableConfigurationService().get("tc_gone")).resolves.toBeNull();
    expect(toastErrorMock).not.toHaveBeenCalled();
  });
});
