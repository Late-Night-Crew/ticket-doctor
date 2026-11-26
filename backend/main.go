// Command ticketdoctor is the Ticket Doctor backend: an HTTP API plus the
// commands that migrate the database and import football data.
package main

import (
	"os"

	"ticketdoctor/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		// cobra has already printed the error
		os.Exit(1)
	}
}
