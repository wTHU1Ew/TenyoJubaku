# Design: Automatic Config Merge

## Context
TenyoJubaku uses a YAML configuration file (`config.yaml`) that users customize with their API credentials and settings. When new features are added (like V5.1 Dynamic SL), new configuration sections are added to `config.template.yaml`. Users must manually add these sections to their server's `config.yaml`, which is error-prone.

## Goals
- Automatically add new config keys during deployment
- Never overwrite user's existing configuration values
- Provide clear feedback about what changed
- Fail safely with rollback capability

## Non-Goals
- Schema validation (out of scope, handled by application)
- Config value migration/transformation
- Interactive merge (fully automatic)

## Decisions

### Decision 1: Use Python for YAML Merge
**Why:** Python has excellent YAML support via PyYAML. Go could work but would require compiling a separate tool. Bash alone cannot handle nested YAML structures reliably.

**Alternatives considered:**
- Bash + sed/awk: Too fragile for nested YAML
- yq (jq for YAML): Would add another dependency; Python already available
- Go tool: Requires separate compilation step

### Decision 2: Recursive Deep Merge Strategy
**Algorithm:**
```python
def deep_merge(template, config):
    result = copy(config)
    for key, value in template.items():
        if key not in result:
            # New key from template - add it
            result[key] = value
        elif isinstance(value, dict) and isinstance(result[key], dict):
            # Both are dicts - recurse
            result[key] = deep_merge(value, result[key])
        # else: keep existing config value
    return result
```

### Decision 3: Backup Before Merge
**Strategy:** Create timestamped backup before any modification:
```bash
cp config.yaml config.yaml.backup.$(date +%Y%m%d_%H%M%S)
```
**Retention:** Keep last 5 backups, delete older ones.

### Decision 4: Integration Point in Deploy Script
**Location:** After `git pull` (Step 4/7), before compilation (Step 5/7)
```
Step 4/7: Clone/Update code
         └─> NEW: Step 4.5/7: Merge config files
Step 5/7: Compile
```

## File Locations
- Merge script: `$SOURCE_DIR/scripts/merge_config.py`
- Template: `$SOURCE_DIR/configs/config.template.yaml`
- User config: `$WORK_DIR/configs/config.yaml`

## Error Handling

| Scenario | Action |
|----------|--------|
| Python not installed | Install via yum |
| PyYAML not installed | pip install PyYAML |
| Merge script fails | Restore backup, abort deploy |
| Template missing | Skip merge (first deploy) |
| Config missing | Error (existing behavior) |

## Risks / Trade-offs
- **Risk:** PyYAML dependency adds complexity
  - Mitigation: PyYAML is stable, widely used, minimal footprint
- **Risk:** Incorrect merge could break config
  - Mitigation: Backup + dry-run mode + clear output

## Example Output
```
步骤 4.5/7: 合并配置文件...
备份: config.yaml → config.yaml.backup.20260128_143022
合并结果:
  + 新增: dynamic_sl.enabled
  + 新增: dynamic_sl.profit_step_pct
  + 新增: dynamic_sl.sl_move_step_pct
  = 保留: okx.api_key (已存在)
  = 保留: tpsl.volatility_pct (已存在)
✓ 配置合并完成 (3 个新字段已添加)
```
