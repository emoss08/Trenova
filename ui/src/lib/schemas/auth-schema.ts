import { boolean, InferType, object, string } from "yup";

export const loginSchema = object({
  emailAddress: string()
    .email("Invalid email address")
    .required("Email is required"),
  password: string().required("Password is required"),
  rememberMe: boolean().optional(),
});

export type LoginSchema = InferType<typeof loginSchema>;

export const checkEmailSchema = object({
  emailAddress: string()
    .email("Invalid email address")
    .required("Email is required"),
});

export type CheckEmailSchema = InferType<typeof checkEmailSchema>;

export const resetPasswordSchema = object({
  emailAddress: string()
    .email("Invalid email address")
    .required("Email is required"),
});

export type ResetPasswordSchema = InferType<typeof resetPasswordSchema>;
