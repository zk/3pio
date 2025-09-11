# Release Checklist

Use this checklist when preparing a new release.

## Pre-release

- [ ] All tests passing locally: `make test-all`
- [ ] Update `CHANGELOG.md` with release notes
- [ ] Update version references in documentation if needed
- [ ] Ensure all package binaries are up to date: `make package`
- [ ] Verify GitHub secrets are configured:
  - [ ] `NPM_TOKEN`
  - [ ] `PYPI_TOKEN`
  - [ ] `HOMEBREW_TAP_GITHUB_TOKEN`

## Release

- [ ] Create and push tag:
  ```bash
  git tag -a v0.0.X -m "Release v0.0.X"
  git push origin v0.0.X
  ```
- [ ] Monitor GitHub Actions: https://github.com/zk/3pio/actions
- [ ] Check GitHub release was created
- [ ] Verify release notes look correct

## Post-release Verification

- [ ] **GitHub Release**: Check https://github.com/zk/3pio/releases
- [ ] **Homebrew**:
  ```bash
  brew update && brew info zk/3pio/3pio
  ```
- [ ] **npm**:
  ```bash
  npm view @heyzk/3pio version
  ```
- [ ] **PyPI**:
  ```bash
  pip index versions threepio-test-runner
  ```

## Installation Testing

- [ ] Test Homebrew installation:
  ```bash
  brew tap zk/3pio
  brew install 3pio
  3pio --version
  ```
- [ ] Test npm installation:
  ```bash
  npm install -g @heyzk/3pio
  3pio --version
  ```
- [ ] Test pip installation:
  ```bash
  pip install threepio-test-runner
  3pio --version
  ```

## Communication

- [ ] Update project README if needed
- [ ] Tweet/announce if significant release
- [ ] Update any integration documentation

## If Issues Occur

See `docs/RELEASE.md` for rollback procedures.