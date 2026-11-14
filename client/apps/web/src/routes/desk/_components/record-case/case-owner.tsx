import { UserAutocompleteField } from "@/components/autocomplete-fields";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { ResolvedUserAvatar } from "@/components/resolved-user-avatar";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { assignDeskCase, releaseDeskCase, type DeskCaseOwnership } from "@/lib/graphql/desk-case";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { useT } from "@trenova/shared/i18n/use-t";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

const assignSchema = z.object({ ownerId: z.string().min(1) });
type AssignValues = z.infer<typeof assignSchema>;

/**
 * Whose case this is, in the case's head. A case has one owner, and only the
 * owner (or whoever covers for them) decides it. Here a person takes an
 * unowned case, lets their own go, and, when they may assign cases, gives one
 * to someone or takes it over.
 */
export function CaseOwner({
  kind,
  id,
  label,
  ownership,
  onChanged,
}: {
  kind: string;
  id: string;
  label: string;
  ownership: DeskCaseOwnership;
  onChanged: () => Promise<unknown>;
}) {
  const t = useT();
  const me = useAuthStore((state) => state.user?.id);
  const [assigning, setAssigning] = useState(false);
  const owner = ownership.owner ?? null;
  const mine = owner !== null && owner.id === me;

  const take = useApiMutation({
    mutationFn: () => assignDeskCase(kind, id, null),
    resourceName: "Case",
    scope: { id: `case-owner:${id}` },
    onSuccess: () => onChanged(),
  });
  const release = useApiMutation({
    mutationFn: () => releaseDeskCase(kind, id),
    resourceName: "Case",
    scope: { id: `case-owner:${id}` },
    onSuccess: () => onChanged(),
  });
  const busy = take.isPending || release.isPending;

  const canTake = owner === null ? ownership.canClaim : !mine && ownership.canAssign;
  const canRelease = owner !== null && (mine || ownership.canAssign);
  const hasMenu = canTake || canRelease || ownership.canAssign;

  const text =
    owner === null
      ? t("Unowned")
      : mine
        ? t("Your case")
        : ownership.covering
          ? t("Covering for {0}", owner.name)
          : t("Owned by {0}", owner.name);

  const chip = (
    <>
      {busy ? (
        <Spinner className="size-3.5" />
      ) : owner ? (
        <ResolvedUserAvatar
          userId={owner.id}
          name={owner.name}
          className="size-4 rounded-full after:rounded-full"
          imageClassName="rounded-full"
          fallbackClassName="rounded-full text-2xs"
        />
      ) : (
        <i className="dk-cz-own-none" aria-hidden />
      )}
      <span>{text}</span>
    </>
  );

  return (
    <>
      {hasMenu ? (
        <DropdownMenu>
          <DropdownMenuTrigger
            render={<button type="button" className="dk-cz-own ui-focus-ring" disabled={busy} />}
            aria-label={t("Case owner: {0}", text)}
          >
            {chip}
            <DeskIcon name="down" size={10} />
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start" sideOffset={6} className="w-56">
            {canTake && (
              <DropdownMenuItem
                title={owner === null ? t("Take this case") : t("Take it over")}
                description={
                  owner === null
                    ? t("You decide it; others can only read it.")
                    : t("{0} can no longer decide it.", owner.name)
                }
                onClick={() => void take.mutateAsync(undefined)}
              />
            )}
            {ownership.canAssign && (
              <DropdownMenuItem
                title={t("Assign to someone")}
                description={t("They are told, and they decide it from then on.")}
                onClick={() => setAssigning(true)}
              />
            )}
            {canRelease && (
              <DropdownMenuItem
                title={mine ? t("Let it go") : t("Leave it unowned")}
                description={t("Anyone who works cases can take it.")}
                onClick={() => void release.mutateAsync(undefined)}
              />
            )}
          </DropdownMenuContent>
        </DropdownMenu>
      ) : (
        <span className="dk-cz-own" data-static>
          {chip}
        </span>
      )}
      {ownership.canAssign && (
        <AssignCaseDialog
          open={assigning}
          kind={kind}
          id={id}
          label={label}
          onOpenChange={setAssigning}
          onChanged={onChanged}
        />
      )}
    </>
  );
}

function AssignCaseDialog({
  open,
  kind,
  id,
  label,
  onOpenChange,
  onChanged,
}: {
  open: boolean;
  kind: string;
  id: string;
  label: string;
  onOpenChange: (open: boolean) => void;
  onChanged: () => Promise<unknown>;
}) {
  const t = useT();
  const form = useForm<AssignValues>({
    resolver: zodResolver(assignSchema),
    defaultValues: { ownerId: "" },
  });
  const {
    control,
    handleSubmit,
    reset,
    formState: { isSubmitting },
  } = form;

  const close = () => {
    onOpenChange(false);
    reset({ ownerId: "" });
  };
  const { mutateAsync } = useApiMutation({
    mutationFn: (values: AssignValues) => assignDeskCase(kind, id, values.ownerId),
    resourceName: "Case",
    form,
    onSuccess: () => onChanged(),
  });

  return (
    <Dialog open={open} onOpenChange={(next) => !next && close()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("Assign this case")}</DialogTitle>
          <DialogDescription>
            {t(
              "Choose who owns the case on {0}. They are told, and only they decide it from then on.",
              label,
            )}
          </DialogDescription>
        </DialogHeader>
        <Form
          onSubmit={(event) => {
            event.stopPropagation();
            void handleSubmit(async (values) => {
              await mutateAsync(values);
              close();
            })(event);
          }}
        >
          <FormGroup cols={1} className="pb-4">
            <FormControl>
              <UserAutocompleteField control={control} name="ownerId" label={t("Owner")} required />
            </FormControl>
          </FormGroup>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={close}>
              {t("Cancel")}
            </Button>
            <Button type="submit" isLoading={isSubmitting} loadingText={t("Assigning...")}>
              {t("Assign")}
            </Button>
          </DialogFooter>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
