
# API (short)

Base URL: http://localhost:8080

POST /reward
- Purpose: award shares to a user and create ledger entries.
- Example request:

  {
    "idempotencyKey": "smoke-1",   // optional
    "userId": 1,
    "symbol": "RELIANCE",
    "units": "1.500000"
  }

- Success (200):

  { "status": "ok", "fees": { "brokeragePct": 0.005, "sttPct": 0.001, "gstPct": 0.18 } }

- Duplicate idempotency key (200):

  { "id": 23, "status": "duplicate_ignored" }

GET /today-stocks/:userId — rewards for today
GET /historical-inr/:userId — daily INR totals for past rewards
GET /stats/:userId — today's grouped shares and current portfolio value
GET /portfolio/:userId — current holdings (symbol, units, price, inr)
GET /health — { "ok": true }

For quick manual testing import `postman/Stocky.postman_collection.json`.
