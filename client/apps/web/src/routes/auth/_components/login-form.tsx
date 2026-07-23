import { InputField } from "@/components/fields/input-field";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { EntraLogo } from "@/components/logos/entra";
import { OktaLogo } from "@/components/logos/okta";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { authService } from "@trenova/shared/services/auth";
import { apiService } from "@/services/api";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { TenantLoginMetadata, UserOrganization } from "@trenova/shared/types/organization";
import { loginRequestSchema, type LoginRequest } from "@trenova/shared/types/user";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
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

  const providerQuery = useQuery({
    queryKey: ["auth-providers", organizationSlug],
    queryFn: async () => authService.listProviders(organizationSlug ?? ""),
    enabled: Boolean(organizationSlug),
  });
  const providers = providerQuery.data ?? [];
  const hasAnySso = providers.length > 0;
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
        {providers.map((provider) => (
          <Button
            key={provider.id}
            className="w-full"
            variant="outline"
            render={
              <a href={authService.getSSOStartUrl(provider.id, organizationSlug ?? "", returnTo)} />
            }
          >
            <ProviderLogo provider={provider.provider} />
            Continue with {provider.name}
          </Button>
        ))}
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

function ProviderLogo({ provider }: { provider: string }) {
  if (provider === "AzureAD") {
    return <EntraLogo className="size-4" />;
  }
  if (provider === "Okta") {
    return <OktaLogo className="h-4 w-auto" />;
  }
  return (
    <span className="flex size-4 items-center justify-center rounded-sm bg-primary/10 text-2xs font-semibold text-primary">
      SSO
    </span>
  );
}
