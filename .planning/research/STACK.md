# Stack Research: Terraform Provider ESXi

**Domain:** Terraform provider with govmomi migration
**Researched:** 2026-02-12
**Overall confidence:** HIGH (based on existing codebase analysis)

## Current Stack (Keep As-Is)

This is a cleanup/fix project, not a stack upgrade. The stack should remain stable.

| Component | Current Version | Recommendation | Confidence |
|-----------|----------------|----------------|------------|
| Go | 1.23.0 | Keep | HIGH |
| govmomi | v0.52.0 | Keep | HIGH |
| Terraform SDK | v0.12.17 | Keep (out of scope) | HIGH |
| golang.org/x/crypto | v0.40.0 | Keep (SSH still needed) | HIGH |
| github.com/tmc/scp | v0.0.0-2017... | Keep (used in guest create) | HIGH |
| github.com/jszwec/csvutil | v1.5.1 | Keep | HIGH |

## Rationale: No Stack Changes

**Why not upgrade Terraform SDK?**
- Explicitly out of scope per project requirements
- Would require rewriting all schema definitions (massive change)
- Mixing SDK upgrade with SSH removal creates compound risk
- Current SDK works fine for existing functionality

**Why not upgrade govmomi?**
- v0.52.0 already supports all needed ESXi operations
- Upgrading risks introducing API breaking changes during migration
- No known bugs blocking the current work
- Upgrade can be a separate future project

**Why keep x/crypto?**
- SSH code is being kept where no govmomi alternative exists
- guest-create.go still uses SSH for OVF/clone operations
- config.go uses SSH for connectivity test
- Cannot remove until full govmomi coverage exists

## What NOT to Change

| DO NOT | Reason |
|--------|--------|
| Upgrade to terraform-plugin-framework | Out of scope, massive rewrite |
| Upgrade govmomi version | Unnecessary risk during cleanup |
| Add new dependencies | This is a removal project, not addition |
| Remove x/crypto or tmc/scp | Still needed for SSH paths being kept |

## Build Configuration

Keep existing build configuration:
```
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '-w -extldflags "-static"' -o terraform-provider-esxi_$(cat version)
```

No changes needed to build process.

---

*Stack research: 2026-02-12*
