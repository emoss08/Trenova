import { getUserDatePreferences, setUserDatePreferences } from "@trenova/shared/lib/date";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { User } from "@trenova/shared/types/user";
import { afterEach, describe, expect, it } from "vitest";

const userWith = (timezone: string, timeFormat: User["timeFormat"]) =>
  ({ timezone, timeFormat }) as User;

afterEach(() => {
  useAuthStore.getState().setUser(null);
  setUserDatePreferences({});
});

describe("auth store date preferences", () => {
  it("mirrors the signed-in user's zone and clock into the date module", () => {
    useAuthStore.getState().setUser(userWith("America/Denver", "24-hour"));

    expect(getUserDatePreferences()).toEqual({
      timezone: "America/Denver",
      timeFormat: "24-hour",
    });
  });

  it("falls back to the browser zone once the user is cleared", () => {
    useAuthStore.getState().setUser(userWith("America/Denver", "24-hour"));
    useAuthStore.getState().setUser(null);

    expect(getUserDatePreferences()).toEqual({ timezone: "auto", timeFormat: "12-hour" });
  });
});
