// Package storage contains VALDR persistent storage adapters.
//
// v0.2 active storage uses BadgerDB with an explicit indexed schema.
// The v0.1 blockchain.json FileStore remains read-compatible only for
// one-way migration and regression tests.
package storage
