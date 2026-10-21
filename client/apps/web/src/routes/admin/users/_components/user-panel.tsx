import { useT } from "@trenova/shared/i18n/use-t";
import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import {
  buildUserPatch,
  createUser,
  patchUser,
  userCreateInput,
  userDetailToFormValues,
} from "@/lib/graphql/users";
import type { UserRow } from "@/lib/graphql/user-table";
import { rootField } from "@trenova/shared/lib/reference-access";
import { user as userQueries } from "@/lib/queries/user";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { AlertCircleIcon } from "@trenova/shared/components/icons";
import { DEFAULT_LOCALE } from "@trenova/shared/i18n/generated/locales";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import {
  userCreateSchema,
  userEditSchema,
  type UserCreateFormValues,
  type UserEditFormValues,
} from "@trenova/shared/types/user";
import { parseAsString, useQueryState } from "nuqs";
import { useMemo } from "react";
import { useForm, type Resolver } from "react-hook-form";
import { UserForm, UserFormSkeleton } from "./user-form";

export const USER_LIST_QUERY_KEY = "user-list";

const EMPTY_EDIT_VALUES: UserEditFormValues = {
  status: "Active",
  name: "",
  username: "",
  emailAddress: "",
  timezone: "",
  timeFormat: "12-hour",
  locale: DEFAULT_LOCALE,
  isLocked: false,
  mustChangePassword: false,
  jobTitle: "",
  phoneNumber: "",
  lastLoginAt: null,
  passwordChangedAt: null,
};

export function UserPanel({ open, onOpenChange, mode, row }: DataTablePanelProps<UserRow>) {
  const [panelEntityId] = useQueryState("panelEntityId", parseAsString);
  const editUserId = row?.id ?? panelEntityId ?? undefined;

  if (mode === "edit") {
    return <UserEditPanel open={open} onOpenChange={onOpenChange} userId={editUserId} />;
  }

  return <UserCreatePanel open={open} onOpenChange={onOpenChange} />;
}

function UserCreatePanel({
  open,
  onOpenChange,
}: Pick<DataTablePanelProps<UserRow>, "open" | "onOpenChange">) {
  const t = useT();

  const form = useForm<UserCreateFormValues>({
    resolver: zodResolver(userCreateSchema) as Resolver<UserCreateFormValues>,
    defaultValues: {
      status: "Active",
      name: "",
      username: "",
      emailAddress: "",
      timezone: "",
      timeFormat: "12-hour",
      locale: DEFAULT_LOCALE,
      roleIds: [],
    },
  });

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      queryKey={USER_LIST_QUERY_KEY}
      title={t("User")}
      mutationFn={(values) => createUser(userCreateInput(values))}
      successToast={(created) =>
        created.status === "Active"
          ? {
              title: t("User created"),
              description: t(
                "An invitation to set up their account was sent to {0}.",
                created.emailAddress,
              ),
            }
          : {
              title: t("User created"),
              description: t(
                "No invitation was sent to {0} because the account is inactive.",
                created.emailAddress,
              ),
            }
      }
      formComponent={<UserForm mode="create" />}
    />
  );
}

function UserEditPanel({
  open,
  onOpenChange,
  userId,
}: Pick<DataTablePanelProps<UserRow>, "open" | "onOpenChange"> & { userId?: string }) {
  const t = useT();
  const queryClient = useQueryClient();

  const { data, isError } = useQuery({
    ...userQueries.account(userId ?? ""),
    enabled: open && Boolean(userId),
  });
  const record = useMemo(() => (data ? userDetailToFormValues(data) : null), [data]);

  const form = useForm<UserEditFormValues>({
    resolver: zodResolver(userEditSchema) as Resolver<UserEditFormValues>,
    defaultValues: EMPTY_EDIT_VALUES,
  });

  return (
    <FormEditPanel<UserEditFormValues, UserEditFormValues>
      open={open}
      onOpenChange={onOpenChange}
      row={record}
      form={form}
      queryKey={USER_LIST_QUERY_KEY}
      title={t("User")}
      fieldKey="username"
      placeholder={
        isError ? (
          <Alert variant="destructive" size="sm">
            <AlertCircleIcon />
            <AlertTitle>{t("This user could not be loaded")}</AlertTitle>
            <AlertDescription>
              {t("Close the panel and open it again. If it keeps failing, the user may have been removed.")}
            </AlertDescription>
          </Alert>
        ) : (
          <UserFormSkeleton />
        )
      }
      mutationFn={async (values, original) => {
        const updated = await patchUser(original.id ?? "", buildUserPatch(original, values));
        queryClient.setQueryData(userQueries.account(updated.id).queryKey, updated);
        return userDetailToFormValues(updated);
      }}
      formComponent={
        data ? (
          <UserForm
            mode="edit"
            userId={data.id}
            emailAddress={rootField(data.emailAddress, "User.emailAddress")}
            isActive={data.status === "Active"}
            awaitingSetup={data.lastLoginAt == null && data.passwordChangedAt == null}
          />
        ) : null
      }
    />
  );
}
