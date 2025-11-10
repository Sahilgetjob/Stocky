
# Design notes (short)

- Idempotency: `POST /reward` accepts an optional key. We check for an existing reward and there is a DB unique index on `rewards.idempotency_key` to guard against races. Handle DB unique-constraint errors (Postgres code 23505) as conflicts.

- Transactions: Reward + ledger writes run inside a DB transaction so failures are rolled back and no partial state remains.

- Missing price: If no price exists for a symbol the service falls back to a default price (for testing). In production you should surface an error or use a reliable price feed.

- Validation: Numeric inputs are parsed server-side; invalid numbers should return 400. (We can tighten this in code.)

- Scaling ideas (brief): cache prices (Redis), move price updates to a worker/cron job, add read replicas for heavy read endpoints, and keep the ledger invariant for auditability.

- Security: don't commit secrets; use `.env.example` as a template. Run Gin in release mode behind a proper proxy in production.
