import { createContext, use } from "react";

/**
 * What a sign-in screen may ask of the stage beside it. Forms depend on this and
 * nothing else, so they never touch WebGL and work unchanged without a stage.
 */
export type AuthStageControls = {
  /** Sends the glyph shockwave across the stage; call once a submit passes validation. */
  burst: () => void;
  /** Widens the ribbon once the person is signed in, and narrows it again on the way back. */
  setDone: (done: boolean) => void;
};

function noop() {}

export const NOOP_AUTH_STAGE: AuthStageControls = Object.freeze({ burst: noop, setDone: noop });

export const AuthStageContext = createContext<AuthStageControls>(NOOP_AUTH_STAGE);

export function useAuthStage(): AuthStageControls {
  return use(AuthStageContext);
}
