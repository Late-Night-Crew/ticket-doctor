// Package loggedmodule names the parts of the system that appear in log lines,
// so logs can be filtered by module (module=import, module=api...).
package loggedmodule

import "log/slog"

const (
	Server  = "server"
	API     = "api"
	Migrate = "migrate"
	Import  = "import"
	Engine  = "engine"
)

// Attr returns the log attribute for a module: logger.Info("...", loggedmodule.Attr(loggedmodule.API)).
func Attr(module string) slog.Attr { return slog.String("module", module) }
