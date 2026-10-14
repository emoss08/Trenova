import { describe, expect, it } from "vitest";
import type { UserEditFormValues } from "@trenova/shared/types/user";
import {
  buildUserPatch,
  timeFormatFromGraphQL,
  timeFormatToGraphQL,
  userCreateInput,
  userDetailToFormValues,
  type UserDetail,
} from "../users";

const detail: UserDetail = {
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
  locale: "es",
  isLocked: true,
  mustChangePassword: true,
  jobTitle: "Dispatcher",
  phoneNumber: "555-0100",
  avatarStyle: "dusk",
  availabilityStatus: "away",
  version: 4,
  createdAt: 1_760_000_000,
  updatedAt: 1_760_000_100,
  lastLoginAt: null,
  passwordChangedAt: null,
};

describe("time format", () => {
  it("round trips between the form and GraphQL", () => {
    expect(timeFormatToGraphQL("12-hour")).toBe("TwelveHour");
    expect(timeFormatToGraphQL("24-hour")).toBe("TwentyFourHour");
    expect(timeFormatFromGraphQL("TwelveHour")).toBe("12-hour");
    expect(timeFormatFromGraphQL("TwentyFourHour")).toBe("24-hour");
  });
});

describe("userDetailToFormValues", () => {
  it("carries the user's own choices and never-set timestamps into the form", () => {
    const values = userDetailToFormValues(detail);

    expect(values).toEqual({
      id: "usr_1",
      version: 4,
      status: "Active",
      name: "Dana Whitfield",
      username: "dwhitfield",
      emailAddress: "dana@example.com",
      timezone: "America/Chicago",
      timeFormat: "24-hour",
      locale: "es",
      isLocked: true,
      mustChangePassword: true,
      jobTitle: "Dispatcher",
      phoneNumber: "555-0100",
      lastLoginAt: null,
      passwordChangedAt: null,
      createdAt: 1_760_000_000,
      updatedAt: 1_760_000_100,
    });
  });

  it("falls back to the default language for one the client does not ship", () => {
    expect(userDetailToFormValues({ ...detail, locale: "xx" }).locale).toBe("en");
  });
});

describe("buildUserPatch", () => {
  const original: UserEditFormValues = userDetailToFormValues(detail);

  it("sends nothing when nothing changed", () => {
    expect(buildUserPatch(original, { ...original })).toEqual({});
  });

  it("sends false toggles that were turned off", () => {
    const patch = buildUserPatch(original, {
      ...original,
      isLocked: false,
      mustChangePassword: false,
    });

    expect(patch).toEqual({ isLocked: false, mustChangePassword: false });
  });

  it("sends an emptied text field as an empty string", () => {
    expect(buildUserPatch(original, { ...original, jobTitle: "" })).toEqual({ jobTitle: "" });
  });

  it("sends the time format in GraphQL's form", () => {
    expect(buildUserPatch(original, { ...original, timeFormat: "12-hour" })).toEqual({
      timeFormat: "TwelveHour",
    });
  });

  it("never sends the username, version or timestamps", () => {
    const patch = buildUserPatch(original, {
      ...original,
      username: "renamed",
      version: 9,
      lastLoginAt: 1_760_000_500,
      createdAt: 1,
    });

    expect(patch).toEqual({});
  });
});

describe("userCreateInput", () => {
  it("maps the create form to CreateUserInput with role ids", () => {
    expect(
      userCreateInput({
        status: "Active",
        name: " Dana Whitfield ",
        username: "dwhitfield ",
        emailAddress: " dana@example.com",
        timezone: "America/Chicago",
        timeFormat: "12-hour",
        locale: "en",
        roleIds: ["rol_1"],
      }),
    ).toEqual({
      status: "Active",
      name: "Dana Whitfield",
      username: "dwhitfield",
      emailAddress: "dana@example.com",
      timezone: "America/Chicago",
      timeFormat: "TwelveHour",
      locale: "en",
      roleIds: ["rol_1"],
    });
  });
});
