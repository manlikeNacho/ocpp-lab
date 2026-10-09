# ocpp-lab

An OCPP 1.6J charge point simulator and a minimal central system (CSMS) in Go, built around one distributed-systems problem:

> A charger holds a WebSocket to **one** CSMS instance. An operator's command can hit **any** instance. How does the command reach the charger?

Keeping per-process WebSocket state fails the moment you run more than one worker: commands randomly report "charger not connected". This project solves it with a Redis ownership registry and pub/sub request/response.

## Status

| Piece | State |
|---|---|
| `internal/ocppj`: OCPP-J framing (CALL / CALLRESULT / CALLERROR) with strict validation | done, tested |
| `internal/registry`: Redis ownership registry with TTL, pub/sub RPC with correlation IDs and timeouts | next |
| `cmd/csms`: WebSocket server, BootNotification / Heartbeat / StatusNotification / Authorize / Start / Stop / MeterValues handlers, `POST /chargers/{id}/commands` | next |
| `cmd/chargepoint`: simulator running a scripted charging session | next |
| `docker-compose.yml`: Redis, two CSMS instances, N simulated chargers | next |

## The design

```
operator --POST /chargers/CP-1/commands--> csms-B
                                             |  who owns CP-1?   (Redis: charger:CP-1 -> csms-A, TTL)
                                             |  publish to channel instance:csms-A  {corrId, action, payload}
                                             v
                                           csms-A --CALL RemoteStartTransaction--> CP-1 (WebSocket)
                                             ^                                        |
                                             +------------ CALLRESULT ----------------+
                                           csms-A publishes reply on reply:<corrId>
operator <---------------- 200 {status: Accepted} ---------- csms-B
```

Planned behaviour:

- **Ownership** is `charger:{id} -> instanceID` with a TTL, refreshed on every heartbeat. If an instance dies, its claims expire on their own.
- **Reconnects** to a different instance take over the key, and the old owner drops its stale connection.
- **Failures are explicit.** A charger that isn't connected returns `404` immediately. A command that times out returns `504` and its pending state is cleaned up.

Edge cases that each get a test: owner crashes (TTL expiry), charger reconnects to another instance mid-command, the reply never arrives, and two instances racing to claim the same charger.

## Run the tests

```bash
go test ./...
```

## Scope

Core profile only (boot, heartbeat, status, authorize, transactions, meter values, remote start/stop). No smart charging, firmware management or security profiles. A small, correct subset is the point.
