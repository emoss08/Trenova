import { describe, expect, it } from "vitest";
import { companionAcceptsServer, developmentInstallScript } from "../capture-release";

describe("companionAcceptsServer", () => {
  it.each([
    "https://tms.acme.test",
    "https://acme.test/tms",
    "http://localhost:8080",
    "http://LOCALHOST:8080",
    "http://127.0.0.1:8080",
    "http://127.8.9.10",
    "http://[::1]:8080",
  ])("accepts %s", (url) => {
    expect(companionAcceptsServer(url)).toBe(true);
  });

  it.each([
    "http://tms.acme.test",
    "http://192.168.1.20:8080",
    "http://localhost.acme.test",
    "http://[::2]:8080",
    "ftp://localhost",
    "not a url",
  ])("refuses %s, as the companion does", (url) => {
    expect(companionAcceptsServer(url)).toBe(false);
  });
});

describe("developmentInstallScript", () => {
  const script = developmentInstallScript("http://localhost:8080");

  it("installs the newest development installer in the folder, unblocked", () => {
    const lines = script.split("\n");
    expect(lines[0]).toBe(
      "$msi = (Get-ChildItem .\\TrenovaCapture-*-x64.msi | Sort-Object LastWriteTime | Select-Object -Last 1).FullName",
    );
    expect(lines[1]).toBe("Unblock-File $msi");
    expect(lines[2]).toMatch(/^msiexec \/i \$msi /);
  });

  it("points the install at this server, with self-update off and a log to read on failure", () => {
    expect(script).toContain(" TRENOVAURL=http://localhost:8080 ");
    expect(script).toContain(" AUTOUPDATE=0 ");
    expect(script).toContain('/l*v "$env:TEMP\\trenova-capture-install.log"');
  });

  it("quotes an address PowerShell or msiexec would otherwise split", () => {
    expect(developmentInstallScript("https://acme.test/my tms")).toContain(
      ' TRENOVAURL="https://acme.test/my tms" ',
    );
  });
});
