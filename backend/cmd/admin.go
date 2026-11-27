package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"ticketdoctor/db"
	"ticketdoctor/services"
)

// The admin commands manage admin accounts (B6). There is no sign-up page:
// this is the only way to make one. Passwords are never taken as arguments
// (they'd end up in shell history); they're typed at a hidden prompt, or piped
// in on stdin for scripts.
//
//	ticketdoctor admin create joseph
//	ticketdoctor admin password joseph
//	ticketdoctor admin disable joseph
//	ticketdoctor admin enable joseph
//	ticketdoctor admin list
var adminCmd = &cobra.Command{
	Use:   "admin",
	Short: "Manage admin accounts",
}

func init() {
	rootCmd.AddCommand(adminCmd)
	adminCmd.AddCommand(
		&cobra.Command{
			Use:   "create USERNAME",
			Short: "Create an admin account (asks for the password)",
			Args:  cobra.ExactArgs(1),
			RunE: withAuth(func(ctx context.Context, auth *services.AuthService, args []string) error {
				pw, err := readNewPassword()
				if err != nil {
					return err
				}
				a, err := auth.CreateAdmin(ctx, args[0], pw)
				if err != nil {
					return err
				}
				fmt.Printf("Created admin %q. Sign in at /admin.\n", a.Username)
				return nil
			}),
		},
		&cobra.Command{
			Use:   "password USERNAME",
			Short: "Set a new password (signs the account out everywhere)",
			Args:  cobra.ExactArgs(1),
			RunE: withAuth(func(ctx context.Context, auth *services.AuthService, args []string) error {
				pw, err := readNewPassword()
				if err != nil {
					return err
				}
				if err := auth.SetPassword(ctx, args[0], pw); err != nil {
					return err
				}
				fmt.Printf("Password changed for %q.\n", args[0])
				return nil
			}),
		},
		&cobra.Command{
			Use:   "disable USERNAME",
			Short: "Stop an account signing in (ends its sessions)",
			Args:  cobra.ExactArgs(1),
			RunE: withAuth(func(ctx context.Context, auth *services.AuthService, args []string) error {
				if err := auth.SetDisabled(ctx, args[0], true); err != nil {
					return err
				}
				fmt.Printf("Disabled %q.\n", args[0])
				return nil
			}),
		},
		&cobra.Command{
			Use:   "enable USERNAME",
			Short: "Let a disabled account sign in again",
			Args:  cobra.ExactArgs(1),
			RunE: withAuth(func(ctx context.Context, auth *services.AuthService, args []string) error {
				if err := auth.SetDisabled(ctx, args[0], false); err != nil {
					return err
				}
				fmt.Printf("Enabled %q.\n", args[0])
				return nil
			}),
		},
		&cobra.Command{
			Use:   "list",
			Short: "List admin accounts",
			Args:  cobra.NoArgs,
			RunE: withAuth(func(ctx context.Context, auth *services.AuthService, _ []string) error {
				users, err := auth.List(ctx)
				if err != nil {
					return err
				}
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				fmt.Fprintln(w, "USERNAME\tROLE\tSTATUS\tLAST SIGN-IN")
				for _, u := range users {
					status, last := "active", "never"
					if u.Disabled {
						status = "disabled"
					}
					if u.LastLoginAt != nil {
						last = u.LastLoginAt.Local().Format("2006-01-02 15:04")
					}
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", u.Username, u.Role, status, last)
				}
				return w.Flush()
			}),
		},
	)
}

// withAuth connects to the database for an admin command. Only the account
// tables are touched, so no session secret is needed.
func withAuth(run func(context.Context, *services.AuthService, []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		conn := cfg.Database.ConnString()
		if conn == "" {
			return errNoDatabase
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute) // includes typing the password
		defer cancel()
		pool, err := db.OpenPool(ctx, conn, 1)
		if err != nil {
			return err
		}
		defer pool.Close()
		err = run(ctx, services.NewAuthService(pool, nil, 0, time.Now), args)
		if ue, ok := services.AsUserError(err); ok {
			return errors.New(ue.Msg) // a plain message, no stack of wrapping
		}
		return err
	}
}

// readNewPassword asks twice at a hidden prompt, or reads one line from stdin
// when it isn't a terminal (e.g. `pass show td | ticketdoctor admin create x`).
func readNewPassword() (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", err
		}
		return strings.TrimRight(line, "\r\n"), nil
	}
	ask := func(prompt string) (string, error) {
		fmt.Fprint(os.Stderr, prompt)
		b, err := term.ReadPassword(fd)
		fmt.Fprintln(os.Stderr)
		return string(b), err
	}
	pw, err := ask(fmt.Sprintf("Password (at least %d characters): ", services.MinPasswordLength))
	if err != nil {
		return "", err
	}
	again, err := ask("Same password again: ")
	if err != nil {
		return "", err
	}
	if pw != again {
		return "", errors.New("the passwords don't match")
	}
	return pw, nil
}
