import { InputField } from "@/components/fields/input-field";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { MicrosoftLogo } from "@/components/logos/microsoft";
import { OktaLogo } from "@/components/logos/okta";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { authService } from "@/services/auth";
import { apiService } from "@/services/api";
import { useAuthStore } from "@/stores/auth-store";
import { usePermissionStore } from "@/stores/permission-store";
import type { TenantLoginMetadata, UserOrganization } from "@/types/organization";
import { loginRequestSchema, type LoginRequest } from "@/types/user";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { useNavigate, useSearchParams } from "react-router";

export function LoginForm({
  organizationSlug,
  tenantMetadata,
  onOrganizationSelectionRequired,
}: {
  organizationSlug?: string;
  tenantMetadata?: TenantLoginMetadata;
  onOrganizationSelectionRequired?: (organizations: UserOrganization[]) => void;
}) {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const ssoError = searchParams.get("sso_error");
  const setUser = useAuthStore((state) => state.setUser);
  const fetchManifest = usePermissionStore((state) => state.fetchManifest);
  const clearPermissions = usePermissionStore((state) => state.clearPermissions);

  const providers = tenantMetadata?.enabledProviders ?? [];
  const hasMicrosoft = providers.includes("AzureAD");
  const hasOkta = providers.includes("Okta");
  const hasAnySso = hasMicrosoft || hasOkta;
  const returnTo = typeof window !== "undefined" ? `${window.location.origin}/` : "/";

  const { control, handleSubmit, setError } = useForm<LoginRequest>({
    resolver: zodResolver(loginRequestSchema),
    defaultValues: {
      emailAddress: "",
      password: "",
      organizationSlug,
    },
  });

  const { mutateAsync, isPending } = useApiMutation({
    mutationFn: authService.login,
    setFormError: setError,
    resourceName: "Login",
    onSuccess: async (data) => {
      setUser(data.user);
      if (!organizationSlug) {
        const availableOrganizations = await apiService.userService.getUserOrganizations();
        if (availableOrganizations.length > 1) {
          clearPermissions();
          onOrganizationSelectionRequired?.(availableOrganizations);
          return;
        }
      }

      await fetchManifest();
      void navigate("/", { replace: true });
    },
  });

  const onSubmit = (data: LoginRequest) => {
    void mutateAsync(data);
  };

  return (
    <Form onSubmit={handleSubmit(onSubmit)}>
      <FormGroup cols={1}>
        {ssoError && (
          <Alert
            variant="destructive"
            role="alert"
            className="cursor-pointer"
            onClick={() =>
              setSearchParams((p) => {
                p.delete("sso_error");
                return p;
              })
            }
          >
            <AlertDescription>{ssoError}</AlertDescription>
          </Alert>
        )}
        {hasMicrosoft && organizationSlug && (
          <Button
            className="w-full"
            variant="outline"
            render={<a href={authService.getSSOStartUrl("AzureAD", organizationSlug, returnTo)} />}
          >
            <MicrosoftLogo className="size-4" />
            Continue with Microsoft
          </Button>
        )}
        {hasOkta && organizationSlug && (
          <Button
            className="w-full"
            variant="outline"
            render={<a href={authService.getSSOStartUrl("Okta", organizationSlug, returnTo)} />}
          >
            <OktaLogo className="h-4 w-auto" />
            Continue with Okta
          </Button>
        )}
        {hasAnySso && tenantMetadata?.passwordEnabled && (
          <div className="text-center text-xs tracking-[0.2em] text-muted-foreground uppercase">
            Or use password
          </div>
        )}
        {(tenantMetadata?.passwordEnabled ?? true) && (
          <>
            <FormControl cols="full">
              <InputField
                name="emailAddress"
                control={control}
                label="Email Address"
                placeholder="name@work-email.com"
                rules={{ required: true }}
              />
            </FormControl>
            <FormControl cols="full">
              <SensitiveField
                name="password"
                control={control}
                placeholder="*****"
                label="Password"
                rules={{ required: true }}
              />
            </FormControl>
            <Button
              type="submit"
              className="w-full"
              isLoading={isPending}
              loadingText="Signing in..."
            >
              Sign in
            </Button>
          </>
        )}
      </FormGroup>
    </Form>
  );
}
