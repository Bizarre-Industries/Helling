// Package main dumps generated OpenAPI from Huma registrations to stdout.
package main

import (
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"gopkg.in/yaml.v3"

	hellingapi "github.com/Bizarre-Industries/Helling/apps/hellingd/api"
)

var stdout io.Writer = os.Stdout

var stderr io.Writer = os.Stderr

var exitFunc = os.Exit

func run(w io.Writer) error {
	mux := http.NewServeMux()
	config := hellingapi.NewConfig()
	api := humago.New(mux, config)
	hellingapi.RegisterOperations(api)
	hellingapi.EnrichOpenAPI(api.OpenAPI())

	encoder := yaml.NewEncoder(w)
	encoder.SetIndent(2)
	if err := encoder.Encode(api.OpenAPI()); err != nil {
		_ = encoder.Close()
		return err
	}

	return encoder.Close()
}

func main() {
	logger := slog.New(slog.NewTextHandler(stderr, nil))
	if err := run(stdout); err != nil {
		logger.Error("dump openapi", slog.Any("err", err))
		exitFunc(1)
	}
}
