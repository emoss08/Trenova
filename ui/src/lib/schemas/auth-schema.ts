import { z } from "zod";

export const loginSchema = z.object({
  emailAddress: z
    .string()
    .email("Invalid email address")
    .min(1, "Email is required"),
  password: z.string().min(1, "Password is required"),
  rememberMe: z.boolean().optional(),
});

export type LoginSchema = z.infer<typeof loginSchema>;

export const checkEmailSchema = z.object({
  emailAddress: z
    .string()
    .email("Invalid email address")
    .min(1, "Email is required"),
});

export type CheckEmailSchema = z.infer<typeof checkEmailSchema>;

export const resetPasswordSchema = z.object({
  emailAddress: z
    .string()
    .email("Invalid email address")
    .min(1, "Email is required"),
});

export type ResetPasswordSchema = z.infer<typeof resetPasswordSchema>;
