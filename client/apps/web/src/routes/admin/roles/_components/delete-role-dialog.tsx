import { useT } from "@trenova/shared/i18n/use-t";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { deleteRole, getRoleImpact } from "@/lib/role-api";
import { useMutation, useQuery } from "@tanstack/react-query";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Trash01Icon } from "@trenova/shared/components/icons";

type DeleteRoleDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  roleId: string;
  roleName: string;
  onDeleted: () => Promise<void> | void;
};

export function DeleteRoleDialog({
  open,
  onOpenChange,
  roleId,
  roleName,
  onDeleted,
}: DeleteRoleDialogProps) {
  const t = useT();

  const { data: holders, isLoading } = useQuery({
    queryKey: ["role-impact", roleId],
    queryFn: () => getRoleImpact(roleId),
    enabled: open,
  });
  const holderCount = holders?.length ?? 0;

  const { mutate, isPending } = useMutation({
    mutationFn: () => deleteRole(roleId, { force: holderCount > 0 }),
    onSuccess: async () => {
      await onDeleted();
      onOpenChange(false);
    },
    onError: (error) => handleMutationError({ error, resourceName: "Role" }),
  });

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogMedia className="bg-danger-subtle text-destructive">
            <Trash01Icon />
          </AlertDialogMedia>
          <AlertDialogTitle>{t("Delete {0}?", roleName)}</AlertDialogTitle>
          <AlertDialogDescription>
            {holderCount > 0
              ? t(
                  "{0, plural, one {# person holds} other {# people hold}} this role. Deleting it takes it from them, and anything they could do only through it stops working at once.",
                  holderCount,
                )
              : t("Nobody holds this role. Roles that inherit it lose what it granted them.")}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={isPending}>{t("Keep role")}</AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            onClick={() => mutate()}
            disabled={isPending || isLoading}
          >
            {isPending
              ? t("Deleting...")
              : holderCount > 0
                ? t("Delete and unassign")
                : t("Delete role")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
