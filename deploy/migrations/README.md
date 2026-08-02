# Aivrae Production Migrations

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
