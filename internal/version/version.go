// Package version is what the four tools answer when asked which one they are.
//
// It is one string in one place because it is one answer: a release is the
// four of them built together, and a qpack that says something a qdaad does not
// is a bug report nobody can read. Before this there were two numbers hardcoded
// in two files — 0.1.0 in the compiler and 0.1.0b in the decompiler — and the
// other two tools had none at all.
//
// It is a var and not a const on purpose. A const is folded into the code that
// reads it and the linker can no longer reach it, so the build cannot put the
// real number in:
//
//	go build -ldflags "-X github.com/jorgefuertes/QDAAD/internal/version.Version=v1.0.1"
//
// which is what the Makefile does, from the tag git describes. See dist.
package version

// Version is the release this was built as.
//
// The default is what a build nobody stamped deserves to say. It is not a
// number, deliberately: a binary built straight out of a working tree is not
// v1.0.1 and should not claim to be, and "dev" in a bug report says more than
// a version that turns out to be wrong.
var Version = "dev"
