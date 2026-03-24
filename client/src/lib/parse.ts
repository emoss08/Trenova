import type { ZodSchema } from "zod";
import { toast } from "sonner";

export async function safeParse<T>(
  schema: ZodSchema<T>,
  data: unknown,
  label?: string,
): Promise<T> {
  const result = await schema.safeParseAsync(data);
  if (!result.success) {
    console.error(`Failed to parse ${label ?? "response"}`, result.error);
    toast.error(`Failed to parse ${label ?? "response"}`, {
      description: "Contact your system administrator for assistance.",
    });

    throw result.error;
  }

  return result.data;
}
