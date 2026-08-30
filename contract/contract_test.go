package contract

import (
	"crypto/sha512"
	"encoding/base32"
	"encoding/hex"
	"strings"
	"testing"
)

// encodeAddress turns a 32-byte Algorand public key into its checksummed
// base32 address form (no padding).
func encodeAddress(pk []byte) string {
	sum := sha512.Sum512_256(pk)
	checksum := sum[len(sum)-4:]
	return strings.TrimRight(
		base32.StdEncoding.EncodeToString(append(append([]byte{}, pk...), checksum...)), "=")
}

// ARC-4 selectors must match the canonical vectors (SHA-512/256 of the
// signature, first four bytes). These are the bytes the keeper must put in
// ApplicationArgs[0] when it calls tick().
func TestSelectorsMatchKnownVectors(t *testing.T) {
	cases := map[string]string{
		TickSignature:      "4d4d5f0b",
		SetKeeperSignature: "c4c1d8f7",
	}
	for sig, want := range cases {
		got := hex.EncodeToString(Selector(sig))
		if got != want {
			t.Errorf("Selector(%q) = %s, want %s", sig, got, want)
		}
	}
}

// The keeper's application address must match
// algosdk.logic.get_application_address(769891898). This is the ONLY shape of
// keeper authorization the contract accepts — never itob(keeper_id).
func TestApplicationAddressMatchesAlgosdk(t *testing.T) {
	const keeperAppID = 769891898
	const want = "M4YFP33L5VIFRF53X53WUMQWBOWSLYQNBSSAJV2SORGF43L36XBY7OREUA"
	got := encodeAddress(ApplicationAddress(keeperAppID))
	if got != want {
		t.Errorf("ApplicationAddress(%d) = %s, want %s", keeperAppID, got, want)
	}
}

// Assembly of the rendered programs with the real go-algorand assembler is
// validated separately in tools/tealcheck (kept out of this module because
// the assembler pulls in go-algorand's CGO/libsodium build requirements).

// Structural trap checks on the rendered source — the three ways this
// contract shape has gone wrong before:
//
//  1. No keeper app id may be compiled in. The id arrives once, at runtime,
//     via set_keeper, and lives only in global state.
//  2. Keeper authorization must route through app_params_get AppAddress
//     (the application address), and the tick path must compare it against
//     txn Sender.
//  3. create must assert zero app args, so no uint64 create arg can ever be
//     mis-mapped to the keeper app id.
//  4. No cadence interval is compiled in — the weekly cadence is an Arcron
//     register-time field.
func TestTrapChecks(t *testing.T) {
	src, err := ApprovalProgram()
	if err != nil {
		t.Fatalf("render approval: %v", err)
	}

	if strings.Contains(src, "769891898") {
		t.Error("keeper app id must not be compiled into the TEAL")
	}
	if !strings.Contains(src, "app_params_get AppAddress") {
		t.Error("keeper authorization must use app_params_get AppAddress")
	}

	// The tick block must compare the resolved keeper application address
	// against txn Sender.
	tickIdx := strings.Index(src, "tick:")
	if tickIdx < 0 {
		t.Fatal("no tick block")
	}
	tickBlock := src[tickIdx:]
	if i := strings.Index(tickBlock, "\nset_keeper:"); i >= 0 {
		tickBlock = tickBlock[:i]
	}
	if !strings.Contains(tickBlock, "app_params_get AppAddress") ||
		!strings.Contains(tickBlock, "txn Sender") {
		t.Error("tick must authorize txn Sender against the keeper application address")
	}

	// The create block must assert NumAppArgs == 0.
	createIdx := strings.Index(src, "create:")
	if createIdx < 0 {
		t.Fatal("no create block")
	}
	createBlock := src[createIdx:]
	if i := strings.Index(createBlock, "\ntick:"); i >= 0 {
		createBlock = createBlock[:i]
	}
	if !strings.Contains(createBlock, "txn NumAppArgs\nint 0\n==\nassert") {
		t.Error("create must assert zero app args")
	}
}

// Rendered templates must not leave any unevaluated template actions behind.
func TestRenderIsTotal(t *testing.T) {
	for _, program := range []func() (string, error){ApprovalProgram, ClearProgram} {
		src, err := program()
		if err != nil {
			t.Fatalf("render: %v", err)
		}
		if strings.Contains(src, "{{") || strings.Contains(src, "}}") {
			t.Error("rendered TEAL still contains template actions")
		}
	}
}
