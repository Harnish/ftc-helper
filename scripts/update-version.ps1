Param(
    [string]$Tag
)

if (-not $Tag -or $Tag -eq '') {
    Write-Error "Tag is required: update-version.ps1 <tag>"
    exit 1
}

$repoRoot = (git rev-parse --show-toplevel).Trim()
$verPath = Join-Path $repoRoot 'VERSION'
Set-Content -Path $verPath -Value $Tag -Encoding UTF8
git -C $repoRoot add VERSION
if (git -C $repoRoot diff --cached --quiet) {
    Write-Output "No changes to commit."
    exit 0
}
git -C $repoRoot commit -m "chore: update VERSION to $Tag"
git -C $repoRoot push origin HEAD
Write-Output "Updated VERSION to $Tag and pushed commit."
