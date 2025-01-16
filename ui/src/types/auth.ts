import { type User } from "./user";

export type CheckEmailResponse = {
  valid: boolean;
};

export type ResetPasswordResponse = {
  message: string;
};

export type LoginRequest = {
  emailAddress: string;
  password: string;
};

export type LoginResponse = {
  sessionID: string;
  expiresAt: string;
  user: User;
};
