# TenyoJubaku Version History

## Version Numbering Convention

**Format:** `VX.Y`
- **X (Major):** Incremented for new feature implementations
- **Y (Minor):** Incremented for bug fixes and small updates

## Version History

### V3.0 (2025-12-25)

**Type:** Critical Bug Fix + Documentation Restructuring

**Changes:**
1. **TPSL Infinite Loop Bug Fix** (Critical)
   - Fixed bug where TPSL orders were created infinitely when position size increased
   - Changed coverage analysis from taking max order size to accumulating all order sizes
   - File: `internal/tpsl/manager.go`
   - See: `docs/features/feature1-tpsl/TPSL_INFINITE_LOOP_FIX_V3.0_2025-12-25.md`

2. **Documentation Restructuring**
   - Organized all documentation into `docs/features/` by feature category
   - Created feature-specific folders: feature1-tpsl, feature2-position-management, feature3-order-control, infrastructure, architecture, archived
   - Established naming convention: `<NAME>_<TYPE>_V<VERSION>_<DATE>.md`

3. **Version Management**
   - Added version package: `internal/version/version.go`
   - Version now displayed on application startup
   - Created this version history document

**Documentation Location:**
- TPSL Fix: `docs/features/feature1-tpsl/TPSL_INFINITE_LOOP_FIX_V3.0_2025-12-25.md`

---

### V2.0 (Previous)

**Features Implemented:**
- Feature 1: TPSL System (Take-Profit/Stop-Loss management)
- Feature 2: Position Management (Stale position filtering, configurable expiration)
- Feature 3: Order Control System (Frequency limiting, maker-only orders)
- Infrastructure: AWS deployment, CLI interface
- Architecture: Real-time API integration, database optimization

**Note:** This version number is retroactively assigned based on the major features present before V3.0.

---

## Future Versions

### Planned for V3.x (Minor updates)
- Bug fixes
- Performance optimizations
- Small enhancements to existing features

### Planned for V4.0 (Next major version)
- Potential new features:
  - Order cancellation API integration
  - Advanced TPSL strategies
  - Multi-account support
  - Web dashboard

---

## Maintenance Notes

- Always update this file when releasing a new version
- Use proper semantic versioning (VX.Y)
- Document all significant changes
- Reference related documentation in `docs/features/`
