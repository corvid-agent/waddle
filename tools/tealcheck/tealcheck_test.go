// Package tealcheck assembles the rendered Waddle TEAL programs with the
// real go-algorand assembler — the same assembler algod runs — and checks
// the resulting bytecode.
//
// This lives in its own module on purpose: go-algorand's logic package
// transitively requires CGO and libsodium (Algorand's fork), which is too
// heavy a price for the everyday `go test ./...` in the root module. Run it
// deliberately:
//
//	cd tools/tealcheck
//	go test .
//
// Requirements: Go >= 1.25, CGO enabled, and go-algorand's dev dependency
// setup (libsodium static build under crypto/libs/<os>/<arch>/ — see
// go-algorand's scripts/build_dev_deps.sh). On a machine without libsodium,
// the root module's tests still validate selectors, the keeper application
// address derivation, and every structural trap check.
package tealcheck

import (
	"bytes"
	"testing"

	"github.com/algorand/go-algorand/data/transactions/logic"

	"github.com/corvid-agent/waddle/contract"
)

func assemble(t *testing.T, name, src string) *logic.OpStream {
	t.Helper()
	ops, err := logic.AssembleStringWithVersion(src, contract.TealVersion)
	if err != nil {
		t.Fatalf("assemble %s: %v", name, err)
	}
	if len(ops.Errors) != 0 {
		t.Fatalf("assembler reported %d errors for %s", len(ops.Errors), name)
	}
	if len(ops.Program) == 0 {
		t.Fatalf("assembled %s program is empty", name)
	}
	return ops
}

// The rendered approval program must assemble cleanly at the pinned TEAL
// version, be stateful, and carry both ARC-4 method selectors into the
// bytecode.
func TestApprovalAssembles(t *testing.T) {
	src, err := contract.ApprovalProgram()
	if err != nil {
		t.Fatalf("render approval: %v", err)
	}
	ops := assemble(t, "approval", src)
	if !ops.HasStatefulOps {
		t.Error("approval program should be stateful")
	}
	for name, sig := range map[string]string{
		"tick":       contract.TickSignature,
		"set_keeper": contract.SetKeeperSignature,
	} {
		if !bytes.Contains(ops.Program, contract.Selector(sig)) {
			t.Errorf("assembled program does not contain %s selector %x",
				name, contract.Selector(sig))
		}
	}
}

func TestClearAssembles(t *testing.T) {
	src, err := contract.ClearProgram()
	if err != nil {
		t.Fatalf("render clear: %v", err)
	}
	assemble(t, "clear", src)
}
