/**
 * What a Trenova Capture release asks of the computer it installs on. A
 * release names the oldest Windows build it runs on; people know Windows by
 * its release name, so the build is turned back into one.
 */

export type WindowsRequirement = {
  /** The release the build belongs to, or null for a build older than any named here. */
  name: string | null;
  build: number;
};

/** Named Windows releases by the build they start at, newest first. */
const WINDOWS_RELEASES: ReadonlyArray<readonly [number, string]> = [
  [26100, "Windows 11 24H2"],
  [22631, "Windows 11 23H2"],
  [22621, "Windows 11 22H2"],
  [22000, "Windows 11 21H2"],
  [19045, "Windows 10 22H2"],
  [19044, "Windows 10 21H2"],
];

export function windowsRequirement(build: number): WindowsRequirement {
  const release = WINDOWS_RELEASES.find(([start]) => build >= start);
  return { name: release ? release[1] : null, build };
}

/** Where the development installer is built: the Trenova Capture workflow's runs on master. */
export const CAPTURE_DEVELOPMENT_BUILDS_URL =
  "https://github.com/emoss08/Trenova/actions/workflows/native-capture.yml?query=branch%3Amaster";

/** The name of the development installer's build artifact. */
export const CAPTURE_DEVELOPMENT_ARTIFACT = "trenova-capture-msi";

/**
 * Whether Trenova Capture will take this as its server address. It speaks
 * HTTPS, and plain HTTP only to this computer, the same rule the companion
 * applies when it is given the address.
 */
export function companionAcceptsServer(serverUrl: string): boolean {
  let url: URL;
  try {
    url = new URL(serverUrl);
  } catch {
    return false;
  }
  if (url.protocol === "https:") {
    return true;
  }
  if (url.protocol !== "http:") {
    return false;
  }
  const host = url.hostname;
  return host === "localhost" || host === "[::1]" || /^127(\.\d{1,3}){3}$/.test(host);
}

/**
 * PowerShell that installs the newest development installer in the current
 * folder against this server: unblocked (a downloaded file is marked as from
 * the internet), self-update off (a development server publishes no release),
 * and a verbose log to read if Windows Installer rolls it back.
 */
export function developmentInstallScript(serverUrl: string): string {
  const address = /\s/.test(serverUrl) ? `"${serverUrl}"` : serverUrl;
  return [
    "$msi = (Get-ChildItem .\\TrenovaCapture-*-x64.msi | Sort-Object LastWriteTime | Select-Object -Last 1).FullName",
    "Unblock-File $msi",
    // i18n-ignore: PowerShell install command
    `msiexec /i $msi TRENOVAURL=${address} AUTOUPDATE=0 /l*v "$env:TEMP\\trenova-capture-install.log"`,
  ].join("\n");
}
