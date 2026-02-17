# Change: Add Automatic Config Merge to Deployment Script

## Why
When deploying new versions, the `config.template.yaml` may contain new configuration fields that don't exist in the server's `config.yaml`. Currently, users must manually identify and add these fields, which is error-prone and can cause features to not work (e.g., V5.1 Dynamic SL was disabled because `dynamic_sl` section wasn't added to server config).

## What Changes
- Modify `debug/deploy_ec2.sh` to automatically merge config files during deployment
- Python merge logic embedded directly in the bash script (no external files)
- New keys from `config.template.yaml` are inserted into `config.yaml`
- Existing values in `config.yaml` are preserved (not overwritten)
- Backup original config before merge with automatic cleanup

## Impact
- Affected files: `debug/deploy_ec2.sh` only
- No new files created (Python logic embedded as heredoc)
- This is a **tooling-only** change (deployment automation)
- **Not pushed to remote repository**

## Behavior
1. During deployment, after pulling latest code:
   - Backup current `config.yaml` to `config.yaml.backup.<timestamp>`
   - Run embedded Python merge logic
   - For each key in template that doesn't exist in config: add it with template's default value
   - For each key in config that exists: keep existing value (don't overwrite)
   - Write merged result to `config.yaml`
2. Display summary of added keys (for user awareness)
3. If merge fails, restore from backup
4. Keep only last 5 backups (auto-cleanup)
