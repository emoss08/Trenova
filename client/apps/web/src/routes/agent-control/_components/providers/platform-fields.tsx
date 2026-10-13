import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import type { AIProviderPlatform, AIProviderPlatformAuth } from "@/types/ai-provider";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext, useWatch } from "react-hook-form";
import { Callout } from "../edit/callout";
import { aicFieldTrigger } from "../edit/field-trigger";
import {
  defaultAuth,
  offeredAuths,
  PLATFORM_AUTH_DESCRIPTIONS,
  PLATFORM_AUTH_LABELS,
  PLATFORM_DESCRIPTIONS,
  PLATFORM_LABELS,
  PLATFORM_SETTING_DESCRIPTIONS,
  PLATFORM_SETTING_LABELS,
  platformDescriptor,
  platformFields,
  settingPlaceholder,
  settingRequired,
} from "./platform-model";
import {
  editorSignIn,
  type PlatformCatalog,
  type ProviderEditorValues,
} from "./provider-editor-model";

type PlatformFieldsProps = {
  catalog: PlatformCatalog;
  /** How the provider signed in when the editor opened, offered even where it is no longer allowed. */
  savedAuth: AIProviderPlatformAuth | "";
};

/**
 * Where an Anthropic Messages provider reaches Claude, how it signs in there, and the
 * settings that platform and way of signing in ask for. A switch of platform or of the
 * way of signing in clears a typed credential, which was meant for the one before.
 */
export function PlatformFields({ catalog, savedAuth }: PlatformFieldsProps) {
  const t = useT();
  const { control, setValue } = useFormContext<ProviderEditorValues>();
  const [kind, platform, platformSettings, baseUrl] = useWatch({
    control,
    name: ["kind", "platform", "platformSettings", "baseUrl"],
  });
  const signIn = editorSignIn({ kind, platform, platformSettings }, catalog);
  const auths = offeredAuths(signIn.descriptor, catalog.serverCloudCredentials, savedAuth);
  const fields = platformFields(signIn.descriptor, signIn.auth);

  const clearCredential = () => setValue("apiKey", "", { shouldDirty: true, shouldValidate: true });

  const choosePlatform = (next: string) => {
    const descriptor = platformDescriptor(catalog.platforms, next as AIProviderPlatform);
    const keeps = offeredAuths(descriptor, catalog.serverCloudCredentials, savedAuth).includes(
      platformSettings.auth as AIProviderPlatformAuth,
    );
    if (!keeps) {
      setValue("platformSettings.auth", defaultAuth(descriptor, catalog.serverCloudCredentials), {
        shouldDirty: true,
        shouldValidate: true,
      });
    }
    clearCredential();
  };

  return (
    <>
      <SelectField<ProviderEditorValues>
        control={control}
        name="platform"
        rules={{ required: true }}
        description={t("Where requests reach Claude.")}
        label={t("Platform")}
        placeholder={t("Platform")}
        options={catalog.platforms.map((entry) => ({
          value: entry.platform,
          label: PLATFORM_LABELS[entry.platform],
          description: PLATFORM_DESCRIPTIONS[entry.platform],
        }))}
        onValueChange={choosePlatform}
        triggerClassName={aicFieldTrigger}
        layout="inline"
      />
      {signIn.platform !== "Direct" && (
        <>
          <SelectField<ProviderEditorValues>
            control={control}
            name="platformSettings.auth"
            rules={{ required: true }}
            description={t("How Trenova proves who it is to the platform.")}
            label={t("Sign in with")}
            placeholder={t("Sign in with")}
            options={auths.map((auth) => ({
              value: auth,
              label: PLATFORM_AUTH_LABELS[auth],
              description: PLATFORM_AUTH_DESCRIPTIONS[auth],
            }))}
            onValueChange={clearCredential}
            triggerClassName={aicFieldTrigger}
            layout="inline"
          />
          {fields.map((setting) => (
            <InputField<ProviderEditorValues>
              key={setting}
              control={control}
              name={`platformSettings.${setting}`}
              rules={
                settingRequired(signIn.platform, setting, baseUrl) ? { required: true } : undefined
              }
              label={PLATFORM_SETTING_LABELS[setting]}
              description={PLATFORM_SETTING_DESCRIPTIONS[setting]}
              placeholder={settingPlaceholder(signIn.platform, setting)}
              autoComplete="off"
              inputClassProps={aicFieldTrigger}
              layout="inline"
            />
          ))}
          {signIn.auth === "Server" &&
            (catalog.serverCloudCredentials ? (
              <Callout icon="shield">
                {t(
                  "Trenova signs in as the server it runs on, such as its IAM role or attached service account. No credential is stored, and a saved one is removed.",
                )}
              </Callout>
            ) : (
              <Callout tone="w">
                {t(
                  "This server no longer lets providers use its own credentials. Choose another way to sign in.",
                )}
              </Callout>
            ))}
        </>
      )}
    </>
  );
}
