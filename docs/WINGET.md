# WinGet Packaging

This document captures the Windows Package Manager plan from
[GitHub Discussion #1552](https://github.com/rorkai/App-Store-Connect-CLI/discussions/1552).

The goal is to make the common install command work:

```powershell
winget install asc
```

Keep the exact package identifier documented as the scripting fallback:

```powershell
winget install --id Rorkai.ASC --exact
```

## Package Identity

Use this identity for the first `microsoft/winget-pkgs` submission:

```yaml
PackageIdentifier: Rorkai.ASC
PackageName: asc
Moniker: asc
Commands:
  - asc
```

`Moniker: asc` is what lets `winget install asc` resolve to this package when
there is no ambiguity. `PackageIdentifier: Rorkai.ASC` remains the stable,
non-ambiguous form for automation.

## First Submission

WinGet packages live in the public
[`microsoft/winget-pkgs`](https://github.com/microsoft/winget-pkgs) repository,
not in this repository. For the initial submission, create a portable package
from the Windows release asset:

```yaml
PackageIdentifier: Rorkai.ASC
PackageVersion: 1.5.0
DefaultLocale: en-US
ManifestType: version
ManifestVersion: 1.6.0
```

```yaml
PackageIdentifier: Rorkai.ASC
PackageVersion: 1.5.0
PackageLocale: en-US
Publisher: Rork
PackageName: asc
PackageUrl: https://github.com/rorkai/App-Store-Connect-CLI
License: MIT
LicenseUrl: https://github.com/rorkai/App-Store-Connect-CLI/blob/main/LICENSE
ShortDescription: Fast, scriptable CLI for the App Store Connect API.
Moniker: asc
Tags:
  - app-store-connect
  - app-store-connect-api
  - cli
  - ios
  - testflight
ReleaseNotesUrl: https://github.com/rorkai/App-Store-Connect-CLI/releases/tag/1.5.0
ManifestType: defaultLocale
ManifestVersion: 1.6.0
```

```yaml
PackageIdentifier: Rorkai.ASC
PackageVersion: 1.5.0
InstallerType: portable
Commands:
  - asc
Installers:
  - Architecture: x64
    InstallerUrl: https://github.com/rorkai/App-Store-Connect-CLI/releases/download/1.5.0/asc_1.5.0_windows_amd64.exe
    InstallerSha256: REPLACE_WITH_RELEASE_SHA256
ManifestType: installer
ManifestVersion: 1.6.0
```

Use the SHA256 from `asc_<version>_checksums.txt` in the GitHub release.

## Acceptance Check

After the manifest PR is merged and WinGet sources update, run the manual
`WinGet smoke` workflow or use a clean Windows machine:

```powershell
winget source update
winget search asc --source winget
winget install asc --source winget --accept-source-agreements --accept-package-agreements
asc --help
asc version
winget uninstall --id Rorkai.ASC --exact
```

If `winget install asc` prompts for disambiguation, keep the README fallback as
the documented automation-safe command:

```powershell
winget install --id Rorkai.ASC --exact
```
