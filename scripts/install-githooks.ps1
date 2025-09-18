Param()

$repoRoot = (git rev-parse --show-toplevel).Trim()
Write-Output "Setting core.hooksPath to .githooks in repository $repoRoot"
git config core.hooksPath .githooks
Write-Output "Done. To enable hooks for existing clones, run this script in the repo." 
