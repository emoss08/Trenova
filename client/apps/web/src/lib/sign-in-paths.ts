import type { SignInPaths } from "@trenova/shared/lib/router/return-path";

/** Where the TMS signs people in, and where it starts once they have. */
export const webSignInPaths: SignInPaths = { loginPath: "/login", home: "/" };
