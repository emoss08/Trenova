import { InputField } from "@/components/fields/input-field";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextChipsField } from "@/components/fields/text-chips-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { agentExtension } from "@/lib/queries/agent-extension";
import { apiService } from "@/services/api";
import {
  portalSiteFormSchema,
  type PortalSite,
  type PortalSiteFormValues,
} from "@/types/agent-extension";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { useEffect } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { siteFormDefaults, siteRequestOf } from "./portal-model";

type PortalSiteDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** The site being edited, or null to add one. */
  site: PortalSite | null;
};

/**
 * Adds or changes a portal agents may use: where it lives, how to sign in and whether
 * agents may act there or only read it. Every server refusal lands on its field.
 */
export function PortalSiteDialog({ open, onOpenChange, site }: PortalSiteDialogProps) {
  const t = useT();
  const queryClient = useQueryClient();
  const form = useForm<PortalSiteFormValues>({
    resolver: zodResolver(portalSiteFormSchema) as Resolver<PortalSiteFormValues>,
    defaultValues: siteFormDefaults(site),
  });
  const { control, handleSubmit, reset } = form;
  const clearSignIn = useWatch({ control, name: "clearSignIn" });
  const clearAuthenticator = useWatch({ control, name: "clearAuthenticator" });

  useEffect(() => {
    if (open) {
      reset(siteFormDefaults(site));
    }
  }, [open, site, reset]);

  const { mutateAsync, isPending } = useApiMutation<
    PortalSite,
    PortalSiteFormValues,
    unknown,
    PortalSiteFormValues
  >({
    form,
    resourceName: t("Portal site"),
    mutationFn: (values) => {
      const request = siteRequestOf(values, site);
      return site
        ? apiService.agentExtensionService.updatePortalSite(site.id, request)
        : apiService.agentExtensionService.createPortalSite(request);
    },
    onSuccess: async () => {
      onOpenChange(false);
      await queryClient.invalidateQueries({ queryKey: agentExtension._def });
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="lg">
        <DialogHeader>
          <DialogTitle>{site ? t("Edit {0}", site.name) : t("Add a portal site")}</DialogTitle>
          <DialogDescription>
            {t(
              "Agents can only reach the addresses you list here, and the browser is stopped if it leaves them.",
            )}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            className="flex flex-col gap-4"
            onSubmit={(submitEvent) => {
              submitEvent.preventDefault();
              submitEvent.stopPropagation();
              void handleSubmit((values) => mutateAsync(values))(submitEvent);
            }}
          >
            <FormGroup cols={2}>
              <FormControl>
                <InputField<PortalSiteFormValues>
                  control={control}
                  name="name"
                  label={t("Name")}
                  placeholder={t("e.g. Acme DC dock appointments")}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <InputField<PortalSiteFormValues>
                  control={control}
                  name="startUrl"
                  label={t("Start page")}
                  placeholder="https://portal.example.com/login"
                  description={t("Where the browser opens; it must be on one of the addresses.")}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl cols="full">
                <TextChipsField<PortalSiteFormValues>
                  control={control}
                  name="domains"
                  label={t("Addresses")}
                  placeholder="portal.example.com"
                  description={t(
                    "Every hostname the portal uses, such as its sign-in page's. Use *.example.com for all of a company's subdomains.",
                  )}
                  maxItems={10}
                  maxLength={253}
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField<PortalSiteFormValues>
                  control={control}
                  name="purpose"
                  label={t("What it is for")}
                  placeholder={t("e.g. Booking and moving dock appointments at Acme's Dallas DC")}
                  description={t("Agents read this to choose the right portal.")}
                />
              </FormControl>
              {site?.hasSignIn && (
                <FormControl cols="full">
                  <SwitchField<PortalSiteFormValues>
                    control={control}
                    name="clearSignIn"
                    label={t("Remove the stored sign-in")}
                    description={t("The portal is used without signing in from now on.")}
                  />
                </FormControl>
              )}
              {!clearSignIn && (
                <>
                  <FormControl>
                    <InputField<PortalSiteFormValues>
                      control={control}
                      name="username"
                      label={t("Username")}
                      autoComplete="off"
                      placeholder={
                        site?.hasSignIn ? t("Leave blank to keep the stored one") : undefined
                      }
                    />
                  </FormControl>
                  <FormControl>
                    <SensitiveField<PortalSiteFormValues>
                      control={control}
                      name="password"
                      label={t("Password")}
                      autoComplete="new-password"
                      placeholder={
                        site?.hasSignIn ? t("Leave blank to keep the stored one") : undefined
                      }
                    />
                  </FormControl>
                  {site?.hasAuthenticator && (
                    <FormControl cols="full">
                      <SwitchField<PortalSiteFormValues>
                        control={control}
                        name="clearAuthenticator"
                        label={t("Remove the stored authenticator key")}
                        description={t(
                          "The browser stops answering the portal's authenticator codes; the username and password stay.",
                        )}
                      />
                    </FormControl>
                  )}
                  {!clearAuthenticator && (
                    <FormControl cols="full">
                      <SensitiveField<PortalSiteFormValues>
                        control={control}
                        name="authenticatorKey"
                        label={t("Authenticator key")}
                        autoComplete="off"
                        placeholder={
                          site?.hasAuthenticator
                            ? t("Leave blank to keep the stored one")
                            : t("Optional")
                        }
                        description={t(
                          "For a portal that asks for a code from an authenticator app at sign-in: the setup key it showed when the app was added, often under Can't scan the code. Not a six-digit code. The browser makes a fresh code each time, and no model sees the key.",
                        )}
                      />
                    </FormControl>
                  )}
                </>
              )}
              <FormControl cols="full">
                <InputField<PortalSiteFormValues>
                  control={control}
                  name="browserProfileId"
                  label={t("Browser profile")}
                  placeholder={t("Optional")}
                  description={t(
                    "A profile in your browser vendor's account that is already signed in to this portal, for portals that send sign-in codes by text message or email.",
                  )}
                />
              </FormControl>
              <FormControl cols="full">
                <SwitchField<PortalSiteFormValues>
                  control={control}
                  name="allowActions"
                  label={t("Let agents act on this portal")}
                  description={t(
                    "Agents may propose booking, uploading or accepting here. A person approves every one; off, agents only read it.",
                  )}
                />
              </FormControl>
              <FormControl cols="full">
                <SwitchField<PortalSiteFormValues>
                  control={control}
                  name="enabled"
                  label={t("Available to agents")}
                  description={t("Turn it off to keep the site and its history without agents using it.")}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button type="submit" isLoading={isPending}>
                {site ? t("Save") : t("Add site")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
