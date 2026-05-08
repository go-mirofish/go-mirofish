package headless_test

import (
	"context"
	"log"
	"net/http"

	"github.com/go-mirofish/go-mirofish/gateway/sdk/headless"
)

func ExampleRun() {
	go func() {
		if err := headless.Run(context.Background()); err != nil {
			log.Print(err)
		}
	}()
}

func ExampleNew() {
	cfg, err := headless.LoadConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	app, err := headless.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/mirofish/", http.StripPrefix("/mirofish", app.Handler()))
}
