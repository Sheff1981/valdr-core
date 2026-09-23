// Package storage contains VALDR persistent storage adapters.
//
// v0.1 uses an atomic on-disk blockchain snapshot with fsync + rename.
// LevelDB/BadgerDB remain preferred future replacements once the chain grows.
package storage
