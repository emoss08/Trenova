import { z } from "zod";

export const HttpMethod = z.enum(["GET", "POST", "PUT", "DELETE", "PATCH"]);
export type HttpMethodSchema = z.infer<typeof HttpMethod>;
