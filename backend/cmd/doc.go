// Package cmd holds the ticketdoctor CLI (cobra), mirroring nsms/parrot:
//
//	ticketdoctor server    start the HTTP API
//	ticketdoctor migrate   apply embedded goose migrations
//	ticketdoctor seed      load demo data (DEVELOPMENT only)
//	ticketdoctor import    fetch results and fixtures, settle legs, recompute profiles (FR-1, FR-2, FR-7.2)
//
// The daily import runs from a GitHub Actions schedule, not inside the server (SRS 2.3).
package cmd
