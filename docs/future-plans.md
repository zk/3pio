# Future Plans

This document tracks planned enhancements and features for 3pio.

## Supply Chain Security

### SBOM Generation
- Add Software Bill of Materials (SBOM) generation for all release artifacts
- Use Syft for SBOM creation in SPDX format
- Implementation notes:
  - Install `syft` in CI environment
  - Add back to `.goreleaser.yml`:
    ```yaml
    sboms:
      - artifacts: archive
        documents:
          - "{{ .ProjectName }}-{{ .Version }}-{{ .Os }}-{{ .Arch }}.spdx.sbom.json"
    ```

### Artifact Signing with Cosign
- Sign release artifacts and checksums with Cosign/Sigstore
- Enable keyless signing using GitHub Actions OIDC identity
- Implementation notes:
  - Ensure `cosign` is properly installed in CI
  - Fix template variables for GoReleaser v2 compatibility
  - Add back to `.goreleaser.yml` with correct syntax:
    ```yaml
    signs:
      - cmd: cosign
        env:
        - COSIGN_EXPERIMENTAL=1
        certificate_oidc_issuer_url: 'https://token.actions.githubusercontent.com'
        certificate_identity: 'https://github.com/{{ .Env.GITHUB_REPOSITORY }}/.github/workflows/release.yml@refs/tags/{{ .Tag }}'
        args:
          - sign-blob
          - '--output-certificate=${artifact}.pem'
          - '--output-signature=${artifact}.sig'
          - '${artifact}'
          - --yes
        artifacts: checksum
    ```

## Benefits of These Features

### SBOM Benefits
- Enables vulnerability scanning of dependencies
- Provides transparency for security audits
- Helps users track license compliance
- Industry best practice for supply chain security

### Cosign Benefits  
- Cryptographic proof that releases are authentic
- Protection against supply chain attacks
- Builds trust with security-conscious users
- Aligns with SLSA framework requirements

## Timeline
- Consider implementing after core functionality is stable
- Can be added incrementally (SBOM first, then signing)
- Test thoroughly in CI before enabling

## References
- [Syft Documentation](https://github.com/anchore/syft)
- [Cosign Documentation](https://docs.sigstore.dev/cosign/overview/)
- [GoReleaser Signing Guide](https://goreleaser.com/customization/sign/)
- [SLSA Framework](https://slsa.dev/)