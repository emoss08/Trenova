import { monotonicFactory } from "ulid";

const ulid = monotonicFactory();

export function generateRequestID() {
  return `req_${ulid(150000)}`;
}
