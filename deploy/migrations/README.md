# Aivrae Production Migrations

## User Levels And Route Groups

`aivrae_access_policy_expand.py` arms the expand-stage migration that separates
user levels from route groups. It fingerprints the complete legacy routing
state and prints the exact non-`default` route-group code set that `standard`
will receive. It does not expose channel credentials or include raw database
state in its report.

Run and review the preview before deploying the expand image:

```bash
python3 aivrae_access_policy_expand.py \
  --dry-run \
  --report /opt/new-api-stack/migration-reports/access-policy-expand-dry-run.json
```

Arm only the two reviewed fingerprints:

```bash
python3 aivrae_access_policy_expand.py \
  --apply \
  --expected-state-fingerprint <state-sha256> \
  --expected-route-group-fingerprint <route-group-sha256> \
  --report /opt/new-api-stack/migration-reports/access-policy-expand-apply.json
```

Apply creates and validates a fresh backup, locks all legacy migration inputs,
rechecks the complete state, and writes a singleton migration guard in one
transaction. The expand image verifies the same guard under transaction locks
and marks it applied only after the complete backfill succeeds.

## Smart Routing Cleanup

`aivrae_smart_routing_cleanup.py` is a one-off PostgreSQL migration for the
smart-routing rollout. It resolves production targets by channel name and
does not contain historical channel IDs. Existing disabled historical
channels and their disabled Ability rows are fingerprinted and preserved.
Active tokens receive a non-`default` compatibility group chain before the
enabled `default` routes are removed, so an emergency rollback remains usable.

Run a preview first:

```bash
python3 aivrae_smart_routing_cleanup.py \
  --dry-run \
  --report /opt/new-api-stack/migration-reports/smart-routing-dry-run.json
```

Apply only the exact reviewed fingerprint:

```bash
python3 aivrae_smart_routing_cleanup.py \
  --apply \
  --expected-fingerprint <sha256> \
  --report /opt/new-api-stack/migration-reports/smart-routing-apply.json
```

Apply creates and validates a fresh database backup through
`/opt/new-api-stack/backup-db.sh` unless `--backup-file` names an existing
absolute `.sql.gz` backup. It then locks every relevant table, compares the
full target state with the preview, and applies all changes in one
transaction. Reports are mode `0600` and never contain channel credentials.
