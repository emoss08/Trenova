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
