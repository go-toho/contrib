package connectrpco

import (
	"slices"
	"strings"

	"connectrpc.com/connect/v2"
)

// ServiceRegistrar registers one or more Connect services on a server,
// typically by calling a generated Register<Service>Handler function.
type ServiceRegistrar func(*connect.Server)

// serviceNames returns the fully-qualified names of the services registered
// on server, in registration order.
func serviceNames(server *connect.Server) []string {
	var names []string
	for spec := range server.Specs() {
		// procedures have the form "/package.Service/Method"
		name, _, _ := strings.Cut(strings.TrimPrefix(spec.Procedure, "/"), "/")
		if !slices.Contains(names, name) {
			names = append(names, name)
		}
	}
	return names
}
