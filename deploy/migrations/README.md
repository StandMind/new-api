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

## Access Policy Contract

`aivrae_access_policy_contract.py` removes the legacy user-group columns and
the six retired access-policy Options after both production slots run a
contract-ready image. This stage is irreversible: images that still read the
legacy columns cannot be used after apply.

Run and review the dry-run after the contract-ready image has been stable in
both slots:

```bash
python3 aivrae_access_policy_contract.py \
  --dry-run \
  --report /opt/new-api-stack/migration-reports/access-policy-contract-dry-run.json
```

Apply only the exact reviewed state fingerprint:

```bash
python3 aivrae_access_policy_contract.py \
  --apply \
  --expected-state-fingerprint <state-sha256> \
  --report /opt/new-api-stack/migration-reports/access-policy-contract-apply.json
```

The tool verifies every new/legacy user and subscription level pair, all level
references, the frozen `standard` authorization set, and retirement of the
`default` route group. Apply creates and validates a fresh backup, rechecks the
fingerprint under exclusive table locks, then drops the six columns, deletes
the six old Options, and records `contracted_at` in one transaction. Repeated
runs report `already_contracted` without changing the database.

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
