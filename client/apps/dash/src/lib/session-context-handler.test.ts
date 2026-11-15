import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { handleDashSessionContextStatus, leaveDashExpiredSession } from "./session-context-handler";

describe("handleDashSessionContextStatus", () => {
  const assign = vi.fn();
  const clearAuth = vi.fn();

  beforeEach(() => {
    assign.mockClear();
    clearAuth.mockClear();
    useAuthStore.setState({ clearAuth });
    Object.defineProperty(window, "location", {
      configurable: true,
      value: { ...window.location, pathname: "/", search: "", assign },
    });
  });

  function openPortalAt(pathname: string, search = "") {
    Object.defineProperty(window, "location", {
      configurable: true,
      value: { ...window.location, pathname, search, assign },
    });
  }

  it("leaves for the portal sign-in, saying why, when the person signs out in another tab", () => {
    handleDashSessionContextStatus("signed-out");

    expect(clearAuth).toHaveBeenCalledTimes(1);
    expect(assign).toHaveBeenCalledWith("/dash/login?signed_out=elsewhere");
  });

  it.each(["user-changed", "organization-changed"] as const)(
    "loads the portal fresh for the session the cookie holds now (%s)",
    (status) => {
      handleDashSessionContextStatus(status);

      expect(clearAuth).toHaveBeenCalledTimes(1);
      expect(assign).toHaveBeenCalledWith("/dash");
    },
  );

  it("does nothing while the session still acts for this tab", () => {
    handleDashSessionContextStatus("current");

    expect(clearAuth).not.toHaveBeenCalled();
    expect(assign).not.toHaveBeenCalled();
  });

  it("leaves for the portal sign-in, saying the session ended, when the server ends it", () => {
    leaveDashExpiredSession();

    expect(clearAuth).toHaveBeenCalledTimes(1);
    expect(assign).toHaveBeenCalledWith("/dash/login?signed_out=expired");
  });

  it("carries the portal page the session expired on to the portal sign-in", () => {
    openPortalAt("/dash/pay/stl_1", "?tab=deductions");

    leaveDashExpiredSession();

    expect(assign).toHaveBeenCalledWith(
      "/dash/login?signed_out=expired&redirect=%2Fdash%2Fpay%2Fstl_1%3Ftab%3Ddeductions",
    );
  });

  it("carries nothing when the session expired on the portal's home", () => {
    openPortalAt("/dash");

    leaveDashExpiredSession();

    expect(assign).toHaveBeenCalledWith("/dash/login?signed_out=expired");
  });
});
