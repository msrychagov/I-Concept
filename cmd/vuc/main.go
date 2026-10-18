package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"vucdiagnostic/internal/app"
	"vucdiagnostic/internal/platform/database"
	"vucdiagnostic/internal/platform/gigachat"
	"vucdiagnostic/internal/usecase"
)

func main() {
	addr := flag.String("addr", env("VUC_ADDR", ":8080"), "HTTP address")
	data := flag.String("data", env("VUC_DATA", "data/vuc.json"), "data file")
	demo := flag.Bool("demo-reset", false, "back up existing data, replace with demo accounts and tests, then exit")
	inviteTeacher := flag.String("invite-teacher", "", "create a one-use teacher invitation for this email, then exit")
	vuc := flag.String("vuc", "", "military training center assigned by the administrator")
	createVUC := flag.String("create-vuc", "", "create a military training center, then exit")
	flag.Parse()
	if *createVUC != "" && (*inviteTeacher != "" || *demo) {
		log.Fatal("Use -create-vuc separately from -invite-teacher and -demo-reset")
	}

	a, err := app.New(app.Config{
		DataFile:      *data,
		DatabaseURL:   database.URLFromEnv(),
		AdminUser:     env("VUC_ADMIN_USER", "admin"),
		AdminPassword: env("VUC_ADMIN_PASSWORD", ""),
	})
	if err != nil {
		log.Fatal(err)
	}
	defer a.Close()
	if *demo {
		backup, err := a.ResetDemo()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Demo data ready. Backup: %s", backup)
		return
	}
	if *createVUC != "" {
		if _, err := a.CreateVUC(*createVUC); err != nil {
			log.Fatal(err)
		}
		log.Printf("VUC created: %s", *createVUC)
		return
	}
	if *inviteTeacher != "" {
		invite, err := a.InviteTeacher(*inviteTeacher, *vuc)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Teacher invitation for %s: %s (expires %s)", invite["email"], invite["code"], invite["expires"])
		return
	}

	if model := languageModel(); model != nil {
		a.UseModel(model)
		log.Printf("Language model: %s", model.Name())
	} else {
		log.Print("Language model: off (set GIGACHAT_AUTH_KEY and GIGACHAT_MODEL)")
	}
	srv := &http.Server{Addr: *addr, Handler: a, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 120 * time.Second}
	base := "http://localhost" + *addr
	if (*addr)[0] != ':' {
		base = "http://" + *addr
	}
	log.Printf("VUC web application: %s", base)
	log.Printf("Research cabinet: %s/research", base)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP shutdown: %v", err)
			_ = srv.Close()
		}
	}()
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Printf("HTTP server: %v", err)
	}
	stop()
	<-shutdownDone
}

// languageModel reads the GigaChat settings; without the key or the model it is off.
func languageModel() usecase.TextModel {
	config := gigachat.Config{
		AuthKey:  env("GIGACHAT_AUTH_KEY", ""),
		Model:    env("GIGACHAT_MODEL", ""),
		Scope:    env("GIGACHAT_SCOPE", ""),
		CAFile:   env("GIGACHAT_CA_FILE", ""),
		OAuthURL: env("GIGACHAT_OAUTH_URL", ""),
		APIURL:   env("GIGACHAT_API_URL", ""),
	}
	if !config.Enabled() {
		return nil
	}
	client, err := gigachat.New(config)
	if err != nil {
		log.Printf("Language model: %v", err)
		return nil
	}
	return client
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
