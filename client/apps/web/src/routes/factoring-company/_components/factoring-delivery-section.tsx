import { EmailChipsField } from "@/components/fields/email-chips-field";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SegmentedField } from "@/components/fields/segmented-field";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { SensitiveTextareaField } from "@/components/fields/sensitive-textarea-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { SectionPanel } from "@/components/section-panel";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import {
  DEFAULT_SFTP_PORT,
  FACTORING_EMAIL_RECIPIENT_LIMIT,
  FACTORING_SFTP_AUTH_METHOD_LABELS,
  FACTORING_SFTP_AUTH_METHOD_VALUES,
  FACTORING_TRANSPORT_METHOD_DESCRIPTIONS,
  FACTORING_TRANSPORT_METHOD_LABELS,
  FACTORING_TRANSPORT_METHOD_VALUES,
} from "@/lib/factoring";
import {
  saveFactoringTransport,
  testFactoringTransport,
  type FactoringTransport,
} from "@/lib/graphql/factoring";
import { factoring as factoringQueries, invalidateFactoringQueries } from "@/lib/queries/factoring";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { PlugIcon, Settings01Icon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { translate } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeShort } from "@trenova/shared/lib/date";
import { describeError } from "@trenova/shared/lib/error-presentation";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useEffect, useState } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { toast } from "@trenova/shared/components/ui/toast";
import { z } from "zod";

const MAX_PORT = 65_535;
const DELIVERY_FORM_ID = "factoring-delivery-form";

const deliverySchema = z
  .object({
    method: z.enum(["Email", "Sftp"]),
    emailRecipients: z.array(z.string()),
    attachPackets: z.boolean(),
    sftpHost: z.string().trim(),
    sftpPort: z.number().int().nullable(),
    sftpUsername: z.string().trim(),
    sftpAuthMethod: z.enum(["Password", "PrivateKey"]),
    sftpSecret: z.string(),
    sftpKnownHostKey: z.string().trim(),
    sftpDirectory: z.string().trim(),
    secretStoredFor: z.enum(["Password", "PrivateKey"]).nullable(),
  })
  .superRefine((values, ctx) => {
    if (values.method === "Email") {
      if (values.emailRecipients.length === 0) {
        ctx.addIssue({
          code: "custom",
          path: ["emailRecipients"],
          message: translate("Add the address the factor takes schedules at"),
        });
      }
      if (values.emailRecipients.length > FACTORING_EMAIL_RECIPIENT_LIMIT) {
        ctx.addIssue({
          code: "custom",
          path: ["emailRecipients"],
          message: translate("Send to at most {0} addresses", FACTORING_EMAIL_RECIPIENT_LIMIT),
        });
      }
      return;
    }
    const required: [keyof typeof values, string][] = [
      ["sftpHost", translate("Enter the factor's SFTP host")],
      ["sftpUsername", translate("Enter the user name the factor gave you")],
      ["sftpKnownHostKey", translate("Paste the server's host key so its identity is checked")],
      ["sftpDirectory", translate("Enter the folder schedules are uploaded to")],
    ];
    for (const [field, message] of required) {
      if (String(values[field] ?? "").trim() === "") {
        ctx.addIssue({ code: "custom", path: [field], message });
      }
    }
    if (values.sftpPort == null || values.sftpPort < 1 || values.sftpPort > MAX_PORT) {
      ctx.addIssue({
        code: "custom",
        path: ["sftpPort"],
        message: translate("Enter a port from 1 to 65535"),
      });
    }
    const directory = values.sftpDirectory;
    if (directory !== "" && (!directory.startsWith("/") || directory.split("/").includes(".."))) {
      ctx.addIssue({
        code: "custom",
        path: ["sftpDirectory"],
        message: translate("Use an absolute folder such as /incoming, without .."),
      });
    }
    if (values.secretStoredFor !== values.sftpAuthMethod && values.sftpSecret.trim() === "") {
      ctx.addIssue({
        code: "custom",
        path: ["sftpSecret"],
        message:
          values.sftpAuthMethod === "Password"
            ? translate("Enter the password")
            : translate("Paste the private key"),
      });
    }
  });

type DeliveryValues = z.infer<typeof deliverySchema>;

function deliveryDefaults(transport: FactoringTransport | null): DeliveryValues {
  const storedAuth = transport?.sftpAuthMethod ?? null;
  return {
    method: transport?.method ?? "Email",
    emailRecipients: transport?.emailRecipients ?? [],
    attachPackets: transport?.attachPackets ?? true,
    sftpHost: transport?.sftpHost ?? "",
    sftpPort: transport?.sftpPort ?? DEFAULT_SFTP_PORT,
    sftpUsername: transport?.sftpUsername ?? "",
    sftpAuthMethod: storedAuth ?? "Password",
    sftpSecret: "",
    sftpKnownHostKey: transport?.sftpKnownHostKey ?? "",
    sftpDirectory: transport?.sftpDirectory ?? "",
    secretStoredFor: transport?.sftpSecretSet ? storedAuth : null,
  };
}

/**
 * How this factor takes schedules of accounts: an inbox, or an SFTP folder
 * checked against a pinned host key. The password or key is stored sealed
 * and never shown again; a test connects with the saved settings without
 * sending anything.
 */
export function FactoringDeliverySection({ factoringCompanyId }: { factoringCompanyId: string }) {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed: canUpdate } = usePermission(Resource.FactoringCompany, Operation.Update);
  const { data, isLoading, isError, refetch } = useQuery(
    factoringQueries.transport(factoringCompanyId),
  );
  const [editing, setEditing] = useState(false);

  const test = useMutation({
    mutationFn: () => testFactoringTransport(factoringCompanyId),
    onSuccess: (result) => {
      queryClient.setQueryData(
        factoringQueries.transport(factoringCompanyId).queryKey,
        result.transport,
      );
      if (result.passed) {
        toast.success(t("Connected to the factor"), { description: result.message });
      } else {
        toast.error(t("Could not connect to the factor"), { description: result.message });
      }
    },
    onError: (error) => {
      const described = describeError(error);
      toast.error(t("The connection was not tested"), {
        description: described.detail ?? described.message,
      });
    },
  });

  const action = canUpdate ? (
    <div className="flex items-center gap-1">
      {data ? (
        <Button
          type="button"
          size="xs"
          variant="ghost"
          isLoading={test.isPending}
          loadingText={t("Testing...")}
          onClick={() => test.mutate()}
        >
          <PlugIcon className="size-3" />
          {t("Test")}
        </Button>
      ) : null}
      <Button type="button" size="xs" variant="ghost" onClick={() => setEditing(true)}>
        <Settings01Icon className="size-3" />
        {data ? t("Edit") : t("Set up")}
      </Button>
    </div>
  ) : null;

  return (
    <SectionPanel
      title={t("Schedule delivery")}
      action={action}
      help={t(
        "Where schedules of accounts are sent when invoices are submitted to this factor. Nothing can be submitted until delivery is set up.",
      )}
    >
      <div className="p-3">
        {isLoading ? (
          <Skeleton className="h-16 w-full" />
        ) : isError ? (
          <Alert size="sm" variant="destructive">
            <AlertDescription className="flex items-center justify-between gap-3">
              <span>{t("The delivery settings could not be loaded.")}</span>
              <Button type="button" size="sm" variant="outline" onClick={() => void refetch()}>
                {t("Retry")}
              </Button>
            </AlertDescription>
          </Alert>
        ) : data ? (
          <DeliverySummary transport={data} />
        ) : (
          <p className="text-muted-foreground text-sm">
            {t("Not set up. Invoices for this factor can't be submitted until it is.")}
          </p>
        )}
      </div>
      {editing ? (
        <DeliveryDialog
          factoringCompanyId={factoringCompanyId}
          transport={data ?? null}
          onOpenChange={(open) => {
            if (!open) setEditing(false);
          }}
          onSaved={() => {
            invalidateFactoringQueries(queryClient);
            setEditing(false);
          }}
        />
      ) : null}
    </SectionPanel>
  );
}

function DeliverySummary({ transport }: { transport: FactoringTransport }) {
  const t = useT();

  return (
    <div className="flex flex-col gap-3">
      <DescriptionList columns={2}>
        <DescriptionItem label={t("Method")}>
          {FACTORING_TRANSPORT_METHOD_LABELS[transport.method]}
        </DescriptionItem>
        {transport.method === "Email" ? (
          <>
            <DescriptionItem label={t("Sent to")}>
              <span className="break-all">{transport.emailRecipients.join(", ")}</span>
            </DescriptionItem>
            <DescriptionItem label={t("Invoice packets")}>
              {transport.attachPackets ? t("Attached") : t("Not attached")}
            </DescriptionItem>
          </>
        ) : (
          <>
            <DescriptionItem label={t("Server")}>
              <span className="font-mono text-xs break-all">
                {transport.sftpUsername}@{transport.sftpHost}:{transport.sftpPort}
              </span>
            </DescriptionItem>
            <DescriptionItem label={t("Folder")}>
              <span className="font-mono text-xs break-all">{transport.sftpDirectory}</span>
            </DescriptionItem>
            <DescriptionItem label={t("Sign-in")}>
              {transport.sftpAuthMethod
                ? FACTORING_SFTP_AUTH_METHOD_LABELS[transport.sftpAuthMethod]
                : null}
              {transport.sftpSecretSet ? null : ` · ${t("missing")}`}
            </DescriptionItem>
          </>
        )}
        <DescriptionItem label={t("Last test")}>
          {transport.lastTestedAt ? (
            transport.lastTestError ? (
              t("Failed {0}", formatUnixDateTimeShort(transport.lastTestedAt))
            ) : (
              t("Passed {0}", formatUnixDateTimeShort(transport.lastTestedAt))
            )
          ) : (
            <DescriptionEmpty />
          )}
        </DescriptionItem>
      </DescriptionList>
      {transport.lastTestedAt && transport.lastTestError ? (
        <Alert size="sm" variant="destructive">
          <AlertDescription>{transport.lastTestError}</AlertDescription>
        </Alert>
      ) : null}
    </div>
  );
}

function DeliveryDialog({
  factoringCompanyId,
  transport,
  onOpenChange,
  onSaved,
}: {
  factoringCompanyId: string;
  transport: FactoringTransport | null;
  onOpenChange: (open: boolean) => void;
  onSaved: () => void;
}) {
  const t = useT();
  const form = useForm<DeliveryValues>({
    resolver: zodResolver(deliverySchema) as Resolver<DeliveryValues>,
    defaultValues: deliveryDefaults(transport),
  });
  const { control, reset } = form;
  const method = useWatch({ control, name: "method" });
  const authMethod = useWatch({ control, name: "sftpAuthMethod" });
  const secretStoredFor = useWatch({ control, name: "secretStoredFor" });

  useEffect(() => {
    reset(deliveryDefaults(transport));
  }, [transport, reset]);

  const mutation = useApiMutation<FactoringTransport, DeliveryValues, unknown, DeliveryValues>({
    form,
    resourceName: "delivery settings",
    mutationFn: (values) => {
      const sftp = values.method === "Sftp";
      const secret = values.sftpSecret.trim();
      return saveFactoringTransport({
        factoringCompanyId,
        version: transport?.version ?? null,
        method: values.method,
        emailRecipients: sftp ? [] : values.emailRecipients,
        attachPackets: values.attachPackets,
        sftpHost: sftp ? values.sftpHost : null,
        sftpPort: sftp ? values.sftpPort : null,
        sftpUsername: sftp ? values.sftpUsername : null,
        sftpAuthMethod: sftp ? values.sftpAuthMethod : null,
        sftpSecret: sftp && secret !== "" ? secret : null,
        sftpKnownHostKey: sftp ? values.sftpKnownHostKey : null,
        sftpDirectory: sftp ? values.sftpDirectory : null,
      });
    },
    onSuccess: () => {
      onSaved();
    },
  });

  const methodOptions = FACTORING_TRANSPORT_METHOD_VALUES.map((value) => ({
    value,
    label: FACTORING_TRANSPORT_METHOD_LABELS[value],
  }));
  const authOptions = FACTORING_SFTP_AUTH_METHOD_VALUES.map((value) => ({
    value,
    label: FACTORING_SFTP_AUTH_METHOD_LABELS[value],
  }));
  const secretKept = secretStoredFor === authMethod;

  return (
    <Dialog open onOpenChange={onOpenChange}>
      <DialogContent size="lg">
        <DialogHeader>
          <DialogTitle>{t("Schedule delivery")}</DialogTitle>
          <DialogDescription>{FACTORING_TRANSPORT_METHOD_DESCRIPTIONS[method]}</DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            id={DELIVERY_FORM_ID}
            onSubmit={(event) => {
              event.stopPropagation();
              void form.handleSubmit((values) => mutation.mutate(values))(event);
            }}
          >
            <FormGroup cols={2}>
              <FormControl cols="full">
                <SegmentedField
                  control={control}
                  name="method"
                  label={t("Send by")}
                  options={methodOptions}
                />
              </FormControl>
              {method === "Email" ? (
                <>
                  <FormControl cols="full">
                    <EmailChipsField
                      control={control}
                      name="emailRecipients"
                      label={t("Recipients")}
                      description={t(
                        "The addresses the factor takes schedules at, up to {0}.",
                        FACTORING_EMAIL_RECIPIENT_LIMIT,
                      )}
                    />
                  </FormControl>
                  <FormControl cols="full">
                    <SwitchField
                      control={control}
                      name="attachPackets"
                      label={t("Attach invoice packets")}
                      description={t(
                        "Each invoice's PDF with its supporting documents. Large schedules are split across several emails.",
                      )}
                      outlined
                    />
                  </FormControl>
                </>
              ) : (
                <>
                  <FormControl>
                    <InputField
                      control={control}
                      name="sftpHost"
                      label={t("Host")}
                      placeholder="sftp.factor.com"
                      rules={{ required: true }}
                    />
                  </FormControl>
                  <FormControl>
                    <NumberField
                      control={control}
                      name="sftpPort"
                      label={t("Port")}
                      min={1}
                      max={MAX_PORT}
                      rules={{ required: true }}
                    />
                  </FormControl>
                  <FormControl>
                    <InputField
                      control={control}
                      name="sftpUsername"
                      label={t("User name")}
                      rules={{ required: true }}
                    />
                  </FormControl>
                  <FormControl>
                    <InputField
                      control={control}
                      name="sftpDirectory"
                      label={t("Folder")}
                      placeholder="/incoming"
                      description={t("Each schedule is uploaded to a folder named for it inside.")}
                      rules={{ required: true }}
                    />
                  </FormControl>
                  <FormControl cols="full">
                    <SegmentedField
                      control={control}
                      name="sftpAuthMethod"
                      label={t("Sign in with")}
                      options={authOptions}
                    />
                  </FormControl>
                  <FormControl cols="full">
                    {authMethod === "Password" ? (
                      <SensitiveField
                        control={control}
                        name="sftpSecret"
                        label={t("Password")}
                        placeholder={secretKept ? t("Stored. Leave blank to keep it.") : undefined}
                        rules={{ required: !secretKept }}
                      />
                    ) : (
                      <SensitiveTextareaField
                        control={control}
                        name="sftpSecret"
                        label={t("Private key")}
                        placeholder={
                          secretKept
                            ? t("Stored. Leave blank to keep it.")
                            : "-----BEGIN OPENSSH PRIVATE KEY-----"
                        }
                        rules={{ required: !secretKept }}
                      />
                    )}
                  </FormControl>
                  <FormControl cols="full">
                    <TextareaField
                      control={control}
                      name="sftpKnownHostKey"
                      label={t("Host key")}
                      placeholder="ssh-ed25519 AAAA…"
                      description={t(
                        "The server's public key from the factor, in known_hosts form. A server presenting any other key is refused.",
                      )}
                      rows={2}
                      rules={{ required: true }}
                    />
                  </FormControl>
                </>
              )}
            </FormGroup>
          </Form>
        </FormProvider>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button type="submit" form={DELIVERY_FORM_ID} isLoading={mutation.isPending}>
            {t("Save")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
