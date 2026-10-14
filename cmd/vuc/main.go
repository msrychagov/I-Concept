package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"vucdiagnostic/internal/app"
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
		AdminUser:     env("VUC_ADMIN_USER", "admin"),
		AdminPassword: env("VUC_ADMIN_PASSWORD", ""),
	})
	if err != nil {
		log.Fatal(err)
	}
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

	srv := &http.Server{Addr: *addr, Handler: a, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 120 * time.Second}
	base := "http://localhost" + *addr
	if (*addr)[0] != ':' {
		base = "http://" + *addr
	}
	log.Printf("VUC web application: %s", base)
	log.Printf("Research cabinet: %s/research", base)
	log.Fatal(srv.ListenAndServe())
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
