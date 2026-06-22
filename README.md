# contrib

Contrib contains optional integrations and adapters for
[`github.com/go-toho/toho`](https://github.com/go-toho/toho).

Use these packages when an application needs concrete providers for config,
logging, telemetry, HTTP/RPC middleware, or validation.

## Install

For example:

```sh
go get github.com/go-toho/contrib/config/vipero
```

Install the package or packages your application imports.

## Packages

| Package | Purpose |
| --- | --- |
| `config/vipero` | Load typed config with Viper, defaults, env binding, and mapstructure decoding. |
| `log/zapo` | Build Zap and sugared Zap loggers from Toho logger config. |
| `telemetry/prometheuso` | Prometheus registry and HTTP handler helpers. |
| `net/connectrpco` | ConnectRPC server helpers, health, reflection, auth, and CORS support. |
| `middleware/corso` | CORS config and middleware helpers. |
| `validation/validatoro` | `go-playground/validator` provider. |

Fx adapters live in matching `*ofx` subpackages.

## Fx Registry

Some Fx adapters register themselves with Toho's Fx registry when imported for
side effects. With `tohofx.NewCore()`, registered modules are provided
automatically.

```go
import (
	_ "github.com/go-toho/contrib/config/vipero/viperofx"
	_ "github.com/go-toho/contrib/telemetry/prometheuso/prometheusofx"

	"github.com/go-toho/toho"
	"github.com/go-toho/toho/tohofx"
)

func main() {
	app := toho.New(
		toho.AppCore(tohofx.NewCore()),
	)

	if err := app.Start(); err != nil {
		panic(err)
	}
}
```

## Development

```sh
go test ./...
```
