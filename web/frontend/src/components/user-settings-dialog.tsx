/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
 */

import { User } from "@/types/accounts";
import { Credenza, CredenzaBody, CredenzaContent } from "./ui/credenza";
import UserProfile from "./user-settings/profile-page";

type UserSettingsDialogProps = {
  onOpenChange: () => void;
  open: boolean;
  user: User;
};

export function UserSettingsDialog({
  onOpenChange,
  open,
  user,
}: UserSettingsDialogProps) {
  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent className="max-w-[60em]">
        <CredenzaBody>
          <UserProfile user={user} />
        </CredenzaBody>
      </CredenzaContent>
    </Credenza>
  );
}
