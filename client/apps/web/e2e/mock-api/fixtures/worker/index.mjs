// The worker page's mutable state: every worker's file by id, built fresh for each scenario so
// writes the page makes (verify a credential, approve time off, record an event) stay until the
// scenario is reset. The roster lists Maria first, then the others by last name.
import { MARIA_ID, buildMaria } from "./maria.mjs";
import { NEW_HIRE_ID, NOT_CLEARED_ID, buildRoster } from "./roster.mjs";

export { MARIA_ID, NEW_HIRE_ID, NOT_CLEARED_ID };
export { WORKER_NOW } from "./base.mjs";

export function createWorkerState() {
  const files = new Map();
  for (const file of [buildMaria(), ...buildRoster()]) files.set(file.worker.id, file);
  return { files, created: 0 };
}
