// Package db embeds the goose migrations and seeds into the binary and opens
// database connections (pgx v5).
package db

import (
	"embed"
	"io/fs"
)

// The "all:" prefix includes .gitkeep, so the folders embed even while empty;
// goose only reads .sql and .go files.
//
//go:embed all:migrations
var embeddedMigrations embed.FS

//go:embed all:seeds
var embeddedSeeds embed.FS

// Migrations returns the migrations folder as the root of a file system, as goose expects.
func Migrations() fs.FS { return mustSub(embeddedMigrations, "migrations") }

// Seeds returns the demo seed files loaded by `ticketdoctor seed`.
func Seeds() fs.FS { return mustSub(embeddedSeeds, "seeds") }

func mustSub(f embed.FS, dir string) fs.FS {
	sub, err := fs.Sub(f, dir)
	if err != nil {
		panic(err) // only possible if the embed directive above is wrong
	}
	return sub
}
