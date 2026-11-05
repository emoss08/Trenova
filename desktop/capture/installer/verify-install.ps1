# Installs the MSI on a real Windows machine and checks what cannot be checked anywhere else:
# who owns the directories, that only the spooler can send a job, and that a repair keeps the
# recorded settings. CI runs this elevated on a throwaway runner.
param([Parameter(Mandatory)][string]$Msi)

$ErrorActionPreference = 'Stop'
$failures = [System.Collections.Generic.List[string]]::new()
function Check([bool]$passed, [string]$what) {
  if ($passed) { Write-Host "ok   $what" } else { Write-Host "FAIL $what"; $failures.Add($what) }
}

$root = Join-Path $env:ProgramData 'Trenova'
$shared = Join-Path $root 'Capture'
$key = 'HKLM:\SOFTWARE\Trenova\Capture'
$msiPath = (Resolve-Path $Msi).Path

function Install-Msi([string[]]$arguments, [string]$log) {
  $process = Start-Process msiexec.exe -ArgumentList ($arguments + @('/qn', '/norestart', '/l*v', $log)) -Wait -PassThru
  if ($process.ExitCode -notin 0, 3010) {
    Get-Content $log -Tail 60
    throw "msiexec exited $($process.ExitCode)"
  }
}

# A standard user gets to ProgramData before the installer does.
& net user capturesquatter 'Sq!uatter-Passw0rd#42' /add | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $shared 'updates'), (Join-Path $shared 'logs'), (Join-Path $shared 'spool') | Out-Null
& icacls $root /setowner capturesquatter /T /C | Out-Null
& icacls $root /grant 'capturesquatter:(OI)(CI)F' /T /C | Out-Null
Check ((Get-Acl $shared).Owner -like '*capturesquatter') 'the squatter owns the tree before install'

Install-Msi @('/i', $msiPath, 'TRENOVAURL=https://good.example', 'AUTOUPDATE=0') "$env:TEMP\capture-install.log"

foreach ($path in $root, $shared, (Join-Path $shared 'updates'), (Join-Path $shared 'logs'), (Join-Path $shared 'spool')) {
  $acl = Get-Acl $path
  Check ($acl.Owner -eq 'BUILTIN\Administrators') "$path is owned by Administrators (was $($acl.Owner))"
  Check (-not ($acl.Access.IdentityReference.Value -like '*capturesquatter')) "$path no longer lets the squatter in"
}

$service = Get-Service TrenovaCaptureSvc
Check ($service.Status -eq 'Running') 'the print service runs'
$account = (Get-CimInstance Win32_Service -Filter "Name='TrenovaCaptureSvc'").StartName
Check ($account -like '*LocalService') "it runs as LocalService (is $account)"
Check ($null -ne (Get-Printer -Name Trenova -ErrorAction SilentlyContinue)) 'the printer is installed'

# The real path: a person prints, the spooler sends it, the service attributes it.
$inboxRoot = Join-Path $shared 'spool'
'printed through the spooler' | Out-Printer -Name Trenova
$deadline = (Get-Date).AddSeconds(90)
do {
  Start-Sleep 2
  $delivered = Get-ChildItem $inboxRoot -Recurse -Filter *.pdf -ErrorAction SilentlyContinue
} until ($delivered -or (Get-Date) -gt $deadline)
Check ([bool]$delivered) 'a print through the spooler reaches an inbox under LocalService'
if (-not $delivered) {
  Get-PrintJob -PrinterName Trenova -ErrorAction SilentlyContinue | Format-Table Id,UserName,DocumentName,JobStatus -AutoSize
  Get-ChildItem (Join-Path $shared 'logs') -ErrorAction SilentlyContinue | ForEach-Object { Get-Content $_.FullName -Tail 30 }
}

# Inbox rights: the owner may list, read and delete, and may not write to the directory itself.
$inbox = Get-ChildItem $inboxRoot -Directory | Select-Object -First 1
if ($inbox) {
  $sid = $inbox.Name
  $rule = (Get-Acl $inbox.FullName).Access | Where-Object {
    try { $_.IdentityReference.Translate([System.Security.Principal.SecurityIdentifier]).Value -eq $sid } catch { $false }
  } | Where-Object { $_.PropagationFlags -notmatch 'InheritOnly' } | Select-Object -First 1
  $rights = [int]$rule.FileSystemRights
  Check (($rights -band 0x2) -eq 0 -and ($rights -band 0x100) -eq 0 -and ($rights -band 0x10000) -eq 0) "an inbox owner cannot set a mount point (rights 0x$('{0:x}' -f $rights))"
}

function Ipp([byte]$operation) {
  $text = [System.Text.Encoding]::ASCII
  $bytes = [System.Collections.Generic.List[byte]]::new()
  $bytes.AddRange([byte[]](2, 0, 0, $operation, 0, 0, 0, 7, 1))
  foreach ($attribute in @(@(0x47, 'attributes-charset', 'utf-8'), @(0x48, 'attributes-natural-language', 'en'), @(0x45, 'printer-uri', 'ipp://127.0.0.1:8631/ipp/print'))) {
    $bytes.Add([byte]$attribute[0])
    $bytes.AddRange([byte[]](0, $attribute[1].Length)); $bytes.AddRange($text.GetBytes($attribute[1]))
    $bytes.AddRange([byte[]](0, $attribute[2].Length)); $bytes.AddRange($text.GetBytes($attribute[2]))
  }
  $bytes.Add(3)
  $body = $bytes.ToArray()
  $reply = Invoke-WebRequest -Uri 'http://127.0.0.1:8631/ipp/print' -Method Post -ContentType 'application/ipp' -Body $body -TimeoutSec 20 -SkipHttpErrorCheck
  $content = [byte[]]$reply.Content
  ([int]$content[2] -shl 8) -bor [int]$content[3]
}

# Anyone else on this computer: what the printer is, and nothing of its jobs.
Check ((Ipp 0x0B) -eq 0) 'a local process may read the printer attributes'
Check ((Ipp 0x02) -eq 0x0403) 'a local process cannot send a job'
Check ((Ipp 0x0A) -eq 0x0403) 'a local process cannot list jobs'

$recorded = Get-ItemProperty $key
Install-Msi @('/fa', $msiPath, 'TRENOVAURL=https://evil.example', 'AUTOUPDATE=1', 'PRINTPORT=9999') "$env:TEMP\capture-repair.log"
$after = Get-ItemProperty $key
Check ($after.ServerUrl -eq 'https://good.example') "a repair keeps the server address (is $($after.ServerUrl))"
Check ($after.AutoUpdate -eq 0) "a repair keeps the update choice (is $($after.AutoUpdate))"
Check ("$($after.PrintPort)" -eq "$($recorded.PrintPort)") "a repair keeps the print port (is $($after.PrintPort))"

Install-Msi @('/x', $msiPath) "$env:TEMP\capture-remove.log"

if ($failures.Count -gt 0) {
  throw "$($failures.Count) installed-behaviour checks failed: $($failures -join '; ')"
}
Write-Host 'Every installed-behaviour check passed.'
