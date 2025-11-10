# Database (short)

Uses Postgres; DB name is `assignment`.

Core tables (key columns):

- users: id, name, email
- rewards: id, user_id, symbol, units (numeric(18,6)), event_time, idempotency_key (unique)
- ledger_entries: id, user_id, account, symbol, units, inr_amount, meta
- stock_prices: id, symbol, price (numeric(18,4)), as_of

Notes

- There is a unique index on `rewards.idempotency_key` to make idempotency reliable.
- `ledger_entries` implements a simple double-entry pattern: stock_units vs cash and fee accounts.

Example migration snippets are in the repository if you need them (see docs folder).
  - event_time (timestamptz)
