# Waddle

A Go-written Algorand **TestNet** contract, live on the Arcron keeper network.
**Unaudited. Live on TestNet: app `770742373`, Arcron upkeep `#111` (daily, 30,857 rounds).**

Waddle is the Go-lane sibling of [corvid-agent/plod](https://github.com/corvid-agent/plod)
(Python/AlgoKit): the same Arcron-hook contract shape, but the TEAL is
hand-written as templates and rendered by Go. Nothing here touches MainNet.

## What the contract does

Two ARC-4 methods, dispatched by selector (the selector is the **only** app
arg `tick` accepts):

| Method | Who | What |
| --- | --- | --- |
| `tick()uint64` | the Arcron keeper app only | increments global `calls`, records `Global.round` in `last_round` and the caller in `last_caller`, returns the new counter |
| `set_keeper(uint64)void` | the creator, exactly once | stores the Arcron keeper app id in global `keeper_id` |

Global state schema: 3 uint64 (`calls`, `last_round`, `keeper_id`) +
1 byte slice (`last_caller`).

## The traps this shape is built to avoid

1. **Keeper authorization by application address, never by raw id.**
   `tick` asserts `txn Sender == app_params_get AppAddress(keeper_id)` — the
   keeper's *application address*, i.e.
   `algosdk.logic.get_application_address(769891898)`
   (`M4YFP33L5VIFRF53X53WUMQWBOWSLYQNBSSAJV2SORGF43L36XBY7OREUA` for the
   Arcron TestNet keeper). Authorizing against `itob(keeper_id)` would let
   anyone forge the caller.
2. **Zero create args.** Create asserts `NumAppArgs == 0`. A uint64 create
   arg that gets mapped to the keeper app id freezes the cadence at ~68
   years — there is simply no create arg to mis-map.
3. **No compiled-in cadence.** The daily interval is an Arcron
   register-time field. This program knows nothing about intervals.

## Why hand-written TEAL templates rendered by Go (not SDK assembly)

The program is small, and every trap above lives in the TEAL text itself —
so the TEAL stays readable and reviewable as `contract/approval.tmpl.teal`.
Go (`contract/contract.go`) renders it from typed constants (TEAL version,
state keys, ARC-4 selectors derived with SHA-512/256), which removes
hand-transcription errors. Real assembly is still machine-checked:
`tools/tealcheck` assembles both rendered programs with **go-algorand's own
assembler** (the same one algod runs). That checker lives in its own module
because go-algorand's `logic` package transitively needs CGO + libsodium —
too heavy for the everyday test lane.

## Tests

```sh
go test ./...   # root module: stdlib only, no CGO, no network
```

Validates the ARC-4 selector vectors (`tick()uint64` → `0x4d4d5f0b`,
`set_keeper(uint64)void` → `0xc4c1d8f7`), the keeper application-address
derivation against `algosdk.logic.get_application_address`, and the
structural trap checks above.

```sh
cd tools/tealcheck && go test .   # optional deep check: real TEAL assembly
```

Requires Go ≥ 1.25, CGO, and go-algorand's libsodium dev setup
(`scripts/build_dev_deps.sh` in go-algorand). Last run in this workspace:
**PASS** on go1.27.0 with go-algorand @ `a9753af25360`.

## Layout

```
contract/approval.tmpl.teal   approval program template (the contract)
contract/clear.tmpl.teal      clear-state template (always approves)
contract/contract.go          renderer + shared constants + selector/address helpers
contract/contract_test.go     selector vectors, address vector, trap checks
tools/tealcheck/              separate module: assembles both programs with
                              go-algorand's assembler (CGO + libsodium)
docs/                         GitHub Pages split-flap status board (reads
                              Arcron keeper 769891898 box state on TestNet)
docs/deploy.json              board config — live: appId 770742373, upkeepId 111
```

## Deployment record (TestNet, 2026-08-31)

Deployed by corvid-agent under the operator's go-ahead:

1. Create (zero app args, global schema 3 uint / 1 bytes):
   tx `NH3SWSL7BUFZZME24E5B3GS4BHPY7OUBGQPJUJ43CNB4VYK45IVA` (round 66835218)
   → **app 770742373**.
2. `set_keeper(769891898)`, one-time, creator-only:
   tx `MAIAUPH7VBKCY6WEXJVPCUHZYT74F32BR7UYXGEYIZX2IHS56GJA` (round 66835241).
   Note: `set_keeper` needs the keeper app in `foreign_apps`
   (`app_params_get AppCreator` fails with "unavailable App" otherwise).
3. Registered on keeper `769891898`: **upkeep #111**, `tick()`, interval
   30,857 rounds (~daily), fee 4,000 µALGO, SKIP_AHEAD, 0.5 ALGO escrow:
   group app-call `QV4O74L6VEHFDQDTZ3YYZPA77ZHL6XSID7QVA3YKAZK6TC6O3F6A`.
4. `docs/deploy.json` flipped — the board reads it live.

## GitHub Pages board

`docs/` is a split-flap/CRT status board in the same aesthetic as
corvid-agent/plod. While `appId` is 0 it intentionally shows **NOT
DEPLOYED**; with the live id set it reads the upkeep box on keeper 769891898
whose `target_app` matches, decoding it exactly like the arrivals/plod
boards. Read-only, no wallet, TestNet only.

**Publish pending:** Pages from `docs/` needs enabling in repo settings, and
the token that wrote this repo lacks the `workflow` OAuth scope for
`.github/workflows/pages.yml` (copy it verbatim from
[corvid-agent/plod](https://github.com/corvid-agent/plod/blob/main/.github/workflows/pages.yml)
when a workflow-scoped token is available).

## Status

TestNet only · live as app 770742373 (upkeep #111) · unaudited · no secrets in this repo.
License: MIT.
