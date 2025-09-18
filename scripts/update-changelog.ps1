Param(
    [string]$CommitHash
)

# Update CHANGELOG.md by adding the given commit under "Unreleased".
# If no commit hash is provided, use HEAD.
if (-not $CommitHash -or $CommitHash -eq '') {
    $CommitHash = (git rev-parse --short HEAD).Trim()
}

$raw = git show -s --format="%h %ad %s" --date=short $CommitHash
if (-not $raw) {
    Write-Error "Commit $CommitHash not found."
    exit 1
}

$parts = $raw -split ' ',3
$hash = $parts[0]
$date = $parts[1]
$message = $parts[2]

# Map common conventional commit prefixes to changelog sections
switch -Regex ($message) {
    '^(feat|feature)(\(|:| )' { $section = 'Added'; break }
    '^(fix|bugfix|fixes)(\(|:| )' { $section = 'Fixed'; break }
    '^(docs|doc)(\(|:| )' { $section = 'Documentation'; break }
    '^(chore|style|refactor)(\(|:| )' { $section = 'Changed'; break }
    Default { $section = 'Changed' }
}

$entry = "- $message ($hash) — $date"

$repoRoot = (git rev-parse --show-toplevel).Trim()
$changelogPath = Join-Path $repoRoot 'CHANGELOG.md'

if (-not (Test-Path $changelogPath)) {
    # Create a minimal changelog if none exists
    $header = "# Changelog`n`nAll notable changes to this project will be documented in this file.`n`n## [Unreleased]`n`n### $section`n$entry`n"
    Set-Content -Path $changelogPath -Value $header -Encoding UTF8
    Write-Output "Created CHANGELOG.md and added entry for $hash"
    exit 0
}

$content = Get-Content $changelogPath -Raw -ErrorAction Stop

# If the changelog already contains this commit hash, don't add it again
if ($content -match "\($hash\)") {
    Write-Output "Entry for $hash already exists in CHANGELOG.md — skipping"
    exit 0
}

if ($content -match '## \[Unreleased\]') {
    # If the section for this category exists, insert under it. Otherwise create it.
    if ($content -match "###\s+$section") {
        # Insert the entry after the first occurrence of the section header
        $lines = $content -split "`r?`n"
        $out = New-Object System.Collections.Generic.List[string]
        $inserted = $false
        for ($i=0; $i -lt $lines.Count; $i++) {
            $out.Add($lines[$i])
            if (-not $inserted -and $lines[$i] -match "^###\s+$([regex]::Escape($section))$") {
                $out.Add($entry)
                $inserted = $true
            }
        }
        $new = ($out -join "`n")
        Set-Content -Path $changelogPath -Value $new -Encoding UTF8
        Write-Output "Appended entry to existing Unreleased/$section for $hash"
    }
    else {
        # Add a new subsection under Unreleased
        $new = $content -replace '## \[Unreleased\](\r?\n)+', "## [Unreleased]`n`n### $section`n$entry`n`n"
        Set-Content -Path $changelogPath -Value $new -Encoding UTF8
        Write-Output "Created Unreleased/$section and added entry for $hash"
    }
}
else {
    # Prepend an Unreleased section
    $prefix = "## [Unreleased]`n`n### $section`n$entry`n`n"
    $new = $prefix + $content
    Set-Content -Path $changelogPath -Value $new -Encoding UTF8
    Write-Output "Prepended Unreleased section and added entry for $hash"
}

exit 0
