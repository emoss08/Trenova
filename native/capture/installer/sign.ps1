<#
.SYNOPSIS
  Authenticode-signs Trenova Capture's executables and MSI.

.DESCRIPTION
  Trenova owns the certificate. Two ways of holding it are supported, chosen by
  TRENOVA_SIGNING, and nothing about either is passed on a command line:

  TRENOVA_SIGNING=trusted-signing (Azure Trusted Signing, what CI uses)
    TRENOVA_SIGNING_ENDPOINT      https://<region>.codesigning.azure.net
    TRENOVA_SIGNING_ACCOUNT       the Trusted Signing account
    TRENOVA_SIGNING_PROFILE       the certificate profile
    TRENOVA_SIGNING_DLIB          path to Azure.CodeSigning.Dlib.dll
    Azure credentials come from the environment (azure/login in CI, or az login).

  TRENOVA_SIGNING=certificate (a certificate in the machine's store, by thumbprint)
    TRENOVA_SIGNING_THUMBPRINT    SHA-1 thumbprint of the signing certificate

  TRENOVA_SIGNING=pfx (a file, for a build machine that is not domain-joined)
    TRENOVA_SIGNING_PFX           path to the .pfx
    TRENOVA_SIGNING_PFX_PASSWORD  its password

  The updater accepts a release only when its signer is the same publisher as
  the running updater, so every release must be signed with a certificate
  issued to the same name.

.PARAMETER Path
  The files to sign.
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string[]]$Path
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$timestamp = if ($env:TRENOVA_SIGNING_TIMESTAMP) { $env:TRENOVA_SIGNING_TIMESTAMP } else { 'http://timestamp.digicert.com' }
$mode = $env:TRENOVA_SIGNING
if (-not $mode) {
    throw 'TRENOVA_SIGNING is not set (trusted-signing, certificate or pfx).'
}

function Find-SignTool {
    $candidates = @(
        Get-ChildItem "${env:ProgramFiles(x86)}\Windows Kits\10\bin\*\x64\signtool.exe" -ErrorAction SilentlyContinue |
            Sort-Object FullName -Descending
    )
    if ($candidates.Count -gt 0) { return $candidates[0].FullName }
    $onPath = Get-Command signtool.exe -ErrorAction SilentlyContinue
    if ($onPath) { return $onPath.Source }
    throw 'signtool.exe was not found; install the Windows SDK.'
}

function Require([string]$name) {
    $value = [Environment]::GetEnvironmentVariable($name)
    if (-not $value) { throw "$name is not set." }
    $value
}

$signtool = Find-SignTool
$common = @('sign', '/fd', 'SHA256', '/tr', $timestamp, '/td', 'SHA256', '/d', 'Trenova Capture')

switch ($mode) {
    'trusted-signing' {
        $metadata = New-TemporaryFile
        @{
            Endpoint               = Require 'TRENOVA_SIGNING_ENDPOINT'
            CodeSigningAccountName = Require 'TRENOVA_SIGNING_ACCOUNT'
            CertificateProfileName = Require 'TRENOVA_SIGNING_PROFILE'
        } | ConvertTo-Json | Set-Content $metadata -Encoding UTF8
        $arguments = $common + @('/dlib', (Require 'TRENOVA_SIGNING_DLIB'), '/dmdf', $metadata.FullName)
    }
    'certificate' {
        $arguments = $common + @('/sha1', (Require 'TRENOVA_SIGNING_THUMBPRINT'), '/sm')
    }
    'pfx' {
        $arguments = $common + @('/f', (Require 'TRENOVA_SIGNING_PFX'), '/p', (Require 'TRENOVA_SIGNING_PFX_PASSWORD'))
    }
    default { throw "Unknown TRENOVA_SIGNING mode '$mode'." }
}

foreach ($file in $Path) {
    Write-Host "Signing $file"
    & $signtool @arguments $file
    if ($LASTEXITCODE -ne 0) { throw "signtool failed on $file with $LASTEXITCODE." }
    & $signtool verify /pa /q $file
    if ($LASTEXITCODE -ne 0) { throw "The signature on $file does not verify." }
}
