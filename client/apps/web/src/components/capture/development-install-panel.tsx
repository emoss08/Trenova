import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import {
  CAPTURE_DEVELOPMENT_ARTIFACT,
  CAPTURE_DEVELOPMENT_BUILDS_URL,
  companionAcceptsServer,
  developmentInstallScript,
} from "@/lib/capture-release";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { serverBaseUrl } from "@trenova/shared/lib/api-url";
import { CheckIcon, CopyIcon, ExternalLinkIcon, WrenchIcon } from "lucide-react";
import { useMemo } from "react";

/**
 * How to put Trenova Capture on a Windows computer against a development
 * server, which publishes no signed release to download. The install command
 * is written for this server's address, so it can be pasted as it is.
 */
export function CaptureDevelopmentInstallPanel() {
  const t = useT();
  const { copy, isCopied } = useCopyToClipboard();
  const server = useMemo(() => serverBaseUrl(), []);
  const script = developmentInstallScript(server);

  return (
    <SectionPanel
      title={t("Install a development build")}
      icon={<WrenchIcon aria-hidden />}
      hint={t("Development server")}
    >
      <ol className="flex list-decimal flex-col gap-3 py-3 pr-3 pl-8 text-sm">
        <li className="flex flex-col items-start gap-2">
          <span>
            {t(
              "Download the installer artifact from the newest Trenova Capture build and extract it on the Windows computer, or build the installer there with installer/build.ps1.",
            )}
          </span>
          <div className="flex flex-wrap items-center gap-x-4 gap-y-1">
            <a
              href={CAPTURE_DEVELOPMENT_BUILDS_URL}
              target="_blank"
              rel="noreferrer"
              className="ui-focus-ring text-brand inline-flex items-center gap-1 hover:underline"
            >
              {t("Trenova Capture builds")}
              <ExternalLinkIcon className="size-3.5" aria-hidden />
            </a>
            <span className="text-foreground-muted inline-flex items-center gap-1.5 text-xs">
              {t("Artifact")}
              <code className="bg-sunken text-foreground rounded-sm px-1 font-mono">
                {CAPTURE_DEVELOPMENT_ARTIFACT}
              </code>
            </span>
          </div>
        </li>
        <li className="flex flex-col gap-2">
          <span>
            {t(
              "In that folder, open PowerShell as an administrator and run these commands. They point Trenova Capture at this server and turn self-updating off.",
            )}
          </span>
          <div className="bg-sunken relative rounded-md">
            <pre className="overflow-x-auto py-2.5 pr-11 pl-3 font-mono text-xs whitespace-pre">
              {script}
            </pre>
            <Button
              type="button"
              size="icon-xs"
              variant="ghost"
              className="absolute top-1.5 right-1.5"
              aria-label={t("Copy the install commands")}
              onClick={() => void copy(script, { timeout: 2000, withToast: true })}
            >
              {isCopied ? (
                <CheckIcon className="size-3.5" aria-hidden />
              ) : (
                <CopyIcon className="size-3.5" aria-hidden />
              )}
            </Button>
          </div>
          {!companionAcceptsServer(server) && (
            <Alert variant="warning" size="sm">
              <AlertDescription>
                {t(
                  "Trenova Capture connects over HTTPS, or over plain HTTP only to the computer it runs on. Open this site at localhost on that computer, or serve it over HTTPS, before installing.",
                )}
              </AlertDescription>
            </Alert>
          )}
        </li>
        <li>
          {t(
            "Choose Sign in from the Trenova Capture icon in the notification area and approve the code it shows. If the Trenova printer is missing, choose Add the Trenova printer from the same menu.",
          )}
        </li>
      </ol>
      <SectionPanelQuiet>
        {t(
          "If the install fails, the reason is in trenova-capture-install.log in your temporary folder. A server running in WSL is reached from Windows at localhost.",
        )}
      </SectionPanelQuiet>
    </SectionPanel>
  );
}
