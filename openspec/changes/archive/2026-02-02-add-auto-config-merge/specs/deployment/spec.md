## ADDED Requirements

### Requirement: Automatic Config Merge
The deployment script SHALL automatically merge new configuration fields from `config.template.yaml` into the server's `config.yaml` while preserving existing user values.

#### Scenario: New config field added in template
- **WHEN** a new version adds new config fields to `config.template.yaml`
- **AND** the server's `config.yaml` does not have these fields
- **THEN** the deployment script adds the new fields with their default values from the template

#### Scenario: Existing config values preserved
- **WHEN** a config field exists in both template and server config
- **THEN** the server's existing value is preserved (not overwritten by template default)

#### Scenario: Nested config structures
- **WHEN** a config section contains nested structures (e.g., `dynamic_sl.profit_step_pct`)
- **AND** the parent section exists but nested key is missing
- **THEN** only the missing nested key is added, existing sibling keys are preserved

#### Scenario: Config backup before merge
- **WHEN** the merge process starts
- **THEN** a timestamped backup of the original config is created
- **AND** if merge fails, the backup can be restored

### Requirement: Merge Status Reporting
The deployment script SHALL display a summary of configuration changes after merging.

#### Scenario: Display added fields
- **WHEN** new fields are added to the config
- **THEN** display a list of added field names with "+" indicator

#### Scenario: Display preserved fields
- **WHEN** existing fields are preserved
- **THEN** display a summary count of preserved fields

#### Scenario: Display merge completion
- **WHEN** merge completes successfully
- **THEN** display total count of new fields added
