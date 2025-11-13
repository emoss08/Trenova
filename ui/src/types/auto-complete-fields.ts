import { AccountTypeSchema } from "@/lib/schemas/account-type-schema";

export type UserSelectOptionResponse = {
  id: string;
  name: string;
  emailAddress: string;
  profilePicUrl: string;
};

export type GLAccountSelectOptionResponse = {
  id: string;
  accountCode: string;
  name: string;
};

export type AccountTypeSelectOptionResponse = {
  id: string;
  code: string;
  name: string;
  category: AccountTypeSchema["category"];
};
