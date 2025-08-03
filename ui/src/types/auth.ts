/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import type { UserSchema } from "@/lib/schemas/user-schema";

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
  user: UserSchema;
};
