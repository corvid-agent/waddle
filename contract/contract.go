// Package contract renders the Waddle TEAL programs from hand-written
// templates and exposes the constants shared by the renderer and the tests.
//
// Why hand-written TEAL templates instead of SDK assembly: the program is
// small and every trap it avoids (keeper-by-application-address, zero create
// args, no compiled-in interval) lives in the TEAL itself, so keeping the
// TEAL readable and reviewable matters more than generating it
// programmatically. Go renders the template (version, state keys, ARC-4
// selectors) from typed constants, and the test suite assembles the result
// with the real go-algorand assembler in tools/tealcheck — so the TEAL is
// both human-auditable and machine-checked.
package contract

import (
	"crypto/sha512"
	_ "embed"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
	"text/template"
)

//go:embed approval.tmpl.teal
var approvalTemplate string

//go:embed clear.tmpl.teal
var clearTemplate string

const (
	// TealVersion is the AVM version both programs target.
	TealVersion = 10

	// Global state keys. The Pages board reads KeyCalls.
	KeyCalls      = "calls"
	KeyLastRound  = "last_round"
	KeyLastCaller = "last_caller"
	KeyKeeperID   = "keeper_id"

	// ARC-4 method signatures exposed by the approval program.
	TickSignature      = "tick()uint64"
	SetKeeperSignature = "set_keeper(uint64)void"

	// GlobalUints / GlobalBytes describe the global state schema the app
	// must be created with: calls, last_round, keeper_id (uint64) and
	// last_caller (byte slice).
	GlobalUints = 3
	GlobalBytes = 1
)

// templateParams is the data the templates are rendered with.
type templateParams struct {
	TealVersion          int
	KeyCalls             string
	KeyLastRound         string
	KeyLastCaller        string
	KeyKeeperID          string
	TickSignature        string
	SetKeeperSignature   string
	TickSelectorHex      string
	SetKeeperSelectorHex string
}

func params() templateParams {
	return templateParams{
		TealVersion:          TealVersion,
		KeyCalls:             KeyCalls,
		KeyLastRound:         KeyLastRound,
		KeyLastCaller:        KeyLastCaller,
		KeyKeeperID:          KeyKeeperID,
		TickSignature:        TickSignature,
		SetKeeperSignature:   SetKeeperSignature,
		TickSelectorHex:      hex.EncodeToString(Selector(TickSignature)),
		SetKeeperSelectorHex: hex.EncodeToString(Selector(SetKeeperSignature)),
	}
}

// Selector returns the ARC-4 method selector for a method signature:
// the first four bytes of the SHA-512/256 digest of the signature.
func Selector(signature string) []byte {
	sum := sha512.Sum512_256([]byte(signature))
	return sum[:4]
}

// ApplicationAddress derives the escrow address of an Algorand application,
// equivalent to algosdk.logic.get_application_address: the SHA-512/256
// digest of "appID" || big-endian uint64 app id, expressed as a 32-byte
// public key. Address encoding (checksum + base32) is left to callers.
func ApplicationAddress(appID uint64) []byte {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], appID)
	digest := sha512.Sum512_256(append([]byte("appID"), buf[:]...))
	return digest[:]
}

func render(name, src string, p templateParams) (string, error) {
	t, err := template.New(name).Parse(src)
	if err != nil {
		return "", fmt.Errorf("parse %s: %w", name, err)
	}
	var b strings.Builder
	if err := t.Execute(&b, p); err != nil {
		return "", fmt.Errorf("render %s: %w", name, err)
	}
	return b.String(), nil
}

// ApprovalProgram returns the rendered approval TEAL source.
func ApprovalProgram() (string, error) {
	return render("approval", approvalTemplate, params())
}

// ClearProgram returns the rendered clear-state TEAL source.
func ClearProgram() (string, error) {
	return render("clear", clearTemplate, params())
}
