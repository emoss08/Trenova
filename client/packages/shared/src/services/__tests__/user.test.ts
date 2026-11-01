import { beforeEach, describe, expect, it, vi } from "vitest";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { userService } from "../user";

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: vi.fn(),
}));

const requestGraphQLMock = vi.mocked(requestGraphQL);

const graphQLUser = {
  id: "usr_1",
  businessUnitId: "bu_1",
  currentOrganizationId: "org_1",
  status: "Active",
  name: "Dana Whitfield",
  username: "dwhitfield",
  emailAddress: "dana@example.com",
  profilePicUrl: "",
  thumbnailUrl: "",
  timezone: "America/Chicago",
  timeFormat: "TwentyFourHour",
  locale: "en",
  isLocked: false,
  mustChangePassword: false,
  jobTitle: "",
  phoneNumber: "",
  avatarStyle: "auto",
  availabilityStatus: "available",
  focusUntil: null,
  awayBackOn: null,
  awayDelegateUserId: null,
  awayNote: "",
  assistantHidden: false,
  version: 3,
  createdAt: 1_700_000_000,
  updatedAt: 1_700_000_100,
  lastLoginAt: null,
  passwordChangedAt: null,
};

beforeEach(() => {
  requestGraphQLMock.mockReset();
});

describe("userService self settings", () => {
  it("sends the schema's time format and reads the user back in the client's spelling", async () => {
    requestGraphQLMock.mockResolvedValue({ updateMySettings: graphQLUser } as never);

    const user = await userService.updateMySettings({
      timezone: "America/Chicago",
      timeFormat: "24-hour",
      locale: "en",
    });

    expect(requestGraphQLMock).toHaveBeenCalledWith(
      expect.objectContaining({
        operationName: "UpdateMySettings",
        variables: {
          input: { timezone: "America/Chicago", timeFormat: "TwentyFourHour", locale: "en" },
        },
      }),
    );
    expect(user.timeFormat).toBe("24-hour");
    expect(user.name).toBe("Dana Whitfield");
  });

  it("hands availability off to a trimmed person id and clears a blank one", async () => {
    requestGraphQLMock.mockResolvedValue({ updateMyAvailability: graphQLUser } as never);

    await userService.updateMyAvailability({ status: "away", delegateUserId: " usr_2 " });
    await userService.updateMyAvailability({ status: "available", delegateUserId: "  " });

    expect(requestGraphQLMock.mock.calls[0]?.[0]).toMatchObject({
      operationName: "UpdateMyAvailability",
      variables: { input: { status: "away", delegateUserId: "usr_2" } },
    });
    expect(requestGraphQLMock.mock.calls[1]?.[0]).toMatchObject({
      variables: { input: { status: "available", delegateUserId: null } },
    });
  });

  it("says how many other sessions were signed out", async () => {
    requestGraphQLMock.mockResolvedValue({ revokeMyOtherSessions: 2 } as never);

    await expect(userService.revokeMyOtherSessions()).resolves.toEqual({ revoked: 2 });
  });
});
