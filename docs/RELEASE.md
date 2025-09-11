# Release Process

This document describes the release process for 3pio.

## Prerequisites

### Required GitHub Secrets

The following secrets must be configured in the GitHub repository settings:

1. **`NPM_TOKEN`** - npm authentication token
   - Get from: https://www.npmjs.com/settings/YOUR_USERNAME/tokens
   - Create a "Publish" token
   - Add to GitHub: Settings → Secrets → Actions → New repository secret

2. **`PYPI_TOKEN`** - PyPI authentication token
   - Get from: https://pypi.org/manage/account/token/
   - Create a token scoped to "threepio-test-runner" project
   - Add to GitHub with name `PYPI_TOKEN`

3. **`HOMEBREW_TAP_GITHUB_TOKEN`** - GitHub PAT for updating homebrew tap
   - Create at: https://github.com/settings/tokens/new
   - Required permissions:
     - `repo` (Full control of private repositories)
     - `workflow` (Update GitHub Action workflows)
   - Add to GitHub with name `HOMEBREW_TAP_GITHUB_TOKEN`

## Release Steps

### 1. Update Version and Changelog

```bash
# Update CHANGELOG.md with release notes
vim CHANGELOG.md

# Commit changes
git add CHANGELOG.md
git commit -m "Prepare for v0.0.3 release"
git push origin main
```

### 2. Create and Push Tag

```bash
# Create annotated tag
git tag -a v0.0.3 -m "Release v0.0.3"

# Push tag to trigger release workflow
git push origin v0.0.3
```

### 3. Monitor Release

The GitHub Actions workflow will automatically:

1. **Test** - Run all tests
2. **Build** - Create binaries for all platforms using GoReleaser
3. **Sign** - Sign releases with cosign (optional)
4. **GitHub Release** - Create release with binaries
5. **Homebrew** - Update formula in `homebrew-3pio` tap
6. **npm** - Publish `@heyzk/3pio` package
7. **PyPI** - Publish `threepio-test-runner` package

Monitor progress at: https://github.com/zk/3pio/actions

### 4. Verify Release

After successful release, verify:

```bash
# Homebrew
brew update
brew upgrade 3pio
3pio --version

# npm
npm view @heyzk/3pio version
npm install -g @heyzk/3pio@latest

# pip
pip install --upgrade threepio-test-runner
```

## Manual Release (if needed)

If automated release fails, you can release manually:

### Homebrew
```bash
# GoReleaser will update automatically with HOMEBREW_TAP_GITHUB_TOKEN
```

### npm
```bash
make package
cd packaging/npm
npm version 0.0.3
npm publish
```

### PyPI
```bash
make package
cd packaging/pip
# Update version in setup.py and __init__.py
python -m build
twine upload dist/*
```

## Rollback

If a release has issues:

1. Delete the GitHub release
2. Delete the git tag: `git push --delete origin v0.0.3`
3. Yank npm package: `npm unpublish @heyzk/3pio@0.0.3`
4. Yank PyPI package: (must be done through PyPI web interface)
5. Revert homebrew formula in `homebrew-3pio` repo

## Version Numbering

We follow semantic versioning:
- **Major** (1.0.0): Breaking changes
- **Minor** (0.1.0): New features, backward compatible
- **Patch** (0.0.1): Bug fixes, backward compatible

Pre-release versions:
- Alpha: `v0.1.0-alpha.1`
- Beta: `v0.1.0-beta.1`
- RC: `v0.1.0-rc.1`