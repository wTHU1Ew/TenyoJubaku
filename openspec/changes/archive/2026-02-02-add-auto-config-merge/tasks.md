# Tasks: Add Automatic Config Merge

## 1. Embed Config Merge Logic in Deployment Script
- [x] 1.1 Add Python3 and PyYAML dependency check in `deploy_ec2.sh`
- [x] 1.2 Embed Python merge script as heredoc (no external file needed)
- [x] 1.3 Implement recursive deep merge (handle nested YAML structures)
- [x] 1.4 Add config backup step (with timestamp)
- [x] 1.5 Add error handling and rollback on failure
- [x] 1.6 Display merge summary to user (added/preserved keys)
- [x] 1.7 Clean old backups (keep last 5)

## 2. Testing
- [x] 2.1 Verify script syntax is valid
- [x] 2.2 Verify embedded Python heredoc works correctly

## Notes
- All changes contained in single file: `./debug/deploy_ec2.sh`
- No external scripts created (Python logic embedded as heredoc)
- Not pushed to remote repository
