import type { SignInPaths } from "@trenova/shared/lib/router/return-path";

/** Where the driver portal signs people in, and where it starts once they have. */
export const dashSignInPaths: SignInPaths = { loginPath: "/dash/login", home: "/dash" };
