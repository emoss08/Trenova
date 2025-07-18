import * as z from "zod/v4";

export const loginSchema = z.object({
  emailAddress: z.email("Invalid email address").min(1, {
    error: "Email is required",
  }),
  password: z.string().min(1, { error: "Password is required" }),
  rememberMe: z.boolean().optional(),
});

export type LoginSchema = z.infer<typeof loginSchema>;

export const checkEmailSchema = z.object({
  emailAddress: z.email("Invalid email address").min(1, {
    error: "Email is required",
  }),
});

export type CheckEmailSchema = z.infer<typeof checkEmailSchema>;

export const resetPasswordSchema = z.object({
  emailAddress: z.email("Invalid email address").min(1, {
    error: "Email is required",
  }),
});

export type ResetPasswordSchema = z.infer<typeof resetPasswordSchema>;

export const changePasswordSchema = z
  .object({
    currentPassword: z
      .string()
      .min(1, { error: "Current password is required" }),
    newPassword: z
      .string()
      .min(8, { error: "Password must be at least 8 characters long" })
      .regex(/[A-Z]/, {
        error: "Password must contain at least one uppercase letter",
      })
      .regex(/[a-z]/, {
        error: "Password must contain at least one lowercase letter",
      })
      .regex(/[0-9]/, { error: "Password must contain at least one number" })
      .regex(/[^A-Za-z0-9]/, {
        error: "Password must contain at least one special character",
      }),
    confirmPassword: z
      .string()
      .min(1, { error: "Confirm password is required" }),
  })
  .check((ctx) => {
    if (ctx.value.newPassword !== ctx.value.confirmPassword) {
      ctx.issues.push({
        code: "custom",
        message: "New password and confirm password do not match",
        input: ctx.value.confirmPassword,
        path: ["confirmPassword"],
      });
      ctx.issues.push({
        code: "custom",
        message: "New password and confirm password do not match",
        input: ctx.value.newPassword,
        path: ["newPassword"],
      });
    }
  });

export type ChangePasswordSchema = z.infer<typeof changePasswordSchema>;
