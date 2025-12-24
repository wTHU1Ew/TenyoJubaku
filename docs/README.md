# TenyoJubaku Documentation

This directory contains all project documentation organized by feature and type.

## Directory Structure

```
docs/
├── VERSION_HISTORY.md          # Version history and changelog
├── README.md                   # This file
└── features/                   # Feature-specific documentation
    ├── feature1-tpsl/          # Feature 1: TPSL System
    ├── feature2-position-management/  # Feature 2: Position Management
    ├── feature3-order-control/ # Feature 3: Order Control System
    ├── infrastructure/         # Deployment and infrastructure docs
    ├── architecture/           # Architecture and refactoring docs
    └── archived/              # Archived/deprecated documents
```

## Documentation Naming Convention

All feature documentation follows this naming pattern:

```
<NAME>_<TYPE>_V<VERSION>_<DATE>.md
```

**Examples:**
- `TPSL_INFINITE_LOOP_FIX_V3.0_2025-12-25.md`
- `ORDER_CONTROL_IMPLEMENT_V2.0_2025-12-01.md`
- `AWS_MIGRATION_GUIDE_V2.1_2025-11-15.md`

**Field Descriptions:**
- `<NAME>`: Descriptive name of the document (e.g., TPSL_INFINITE_LOOP)
- `<TYPE>`: Document type - `FIX`, `IMPLEMENT`, `GUIDE`, `REFACTOR`, etc.
- `<VERSION>`: Version when this change was made (e.g., V3.0, V2.1)
- `<DATE>`: Date in YYYY-MM-DD format

## Feature Categories

### Feature 1: TPSL System (`feature1-tpsl/`)
Documentation related to the Take-Profit/Stop-Loss management system.

**Key Documents:**
- TPSL coverage analysis
- Price validation
- Emergency SL adjustments
- API constraint fixes

### Feature 2: Position Management (`feature2-position-management/`)
Documentation related to position tracking and management.

**Key Documents:**
- Stale position filtering
- Configurable position expiration
- Position data retrieval

### Feature 3: Order Control System (`feature3-order-control/`)
Documentation related to order frequency and execution controls.

**Key Documents:**
- Frequency limiting
- Maker-only orders
- Architecture and implementation phases

### Infrastructure (`infrastructure/`)
Deployment, AWS, and operational documentation.

**Key Documents:**
- AWS migration guides
- Deployment procedures
- CLI usage guides

### Architecture (`architecture/`)
System architecture, refactoring, and design documentation.

**Key Documents:**
- Architecture reviews
- Interface refactoring
- Real-time API integration
- Margin mode fixes

### Archived (`archived/`)
Deprecated or outdated documentation kept for historical reference.

## Version Information

Current Version: **V3.0**

See [VERSION_HISTORY.md](VERSION_HISTORY.md) for complete version history and changelog.

## Contributing to Documentation

When adding new documentation:

1. **Determine the feature category** - Place in the appropriate feature folder
2. **Follow naming convention** - Use the `<NAME>_<TYPE>_V<VERSION>_<DATE>.md` format
3. **Update VERSION_HISTORY.md** - Add entry for significant changes
4. **Use clear structure** - Include sections like Problem, Solution, Testing, Deployment

### Document Template

```markdown
# <Document Title>

**Version:** VX.Y
**Date:** YYYY-MM-DD
**Type:** Fix/Feature/Refactor/Guide
**Feature:** Feature X - <Name>

## Problem Description / Overview
[Describe the problem or feature]

## Solution / Implementation
[Describe the solution or implementation]

## Changes Made
[List specific files and changes]

## Testing
[Testing procedures and results]

## Deployment
[Deployment steps if applicable]

## Related Issues
[Links to related docs or issues]
```

## Quick Links

- [Main README](../README.md)
- [Version History](VERSION_HISTORY.md)
- [Latest Changes](features/feature1-tpsl/TPSL_INFINITE_LOOP_FIX_V3.0_2025-12-25.md)

## Questions?

For questions about documentation organization or to suggest improvements, please create an issue or contact the project maintainer.
