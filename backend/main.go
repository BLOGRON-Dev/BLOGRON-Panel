package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"blogron/api"
	"blogron/middleware"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Post("/api/auth/login", api.Login)
	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200); w.Write([]byte(`{"status":"ok"}`))
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.JWTAuth)

		r.Get("/api/system/stats", api.GetSystemStats)
		r.Get("/api/system/services", api.GetServices)
		r.Post("/api/system/services/{name}/restart", api.RestartService)
		r.Post("/api/system/services/{name}/stop", api.StopService)
		r.Post("/api/system/services/{name}/start", api.StartService)
		r.Get("/api/system/logs", api.GetLogs)

		r.Get("/api/users", api.ListUsers)
		r.Post("/api/users", api.CreateUser)
		r.Put("/api/users/{username}", api.UpdateUser)
		r.Delete("/api/users/{username}", api.DeleteUser)
		r.Post("/api/users/{username}/suspend", api.SuspendUser)
		r.Post("/api/users/{username}/activate", api.ActivateUser)

		r.Get("/api/vhosts", api.ListVhosts)
		r.Post("/api/vhosts", api.CreateVhost)
		r.Delete("/api/vhosts/{domain}", api.DeleteVhost)
		r.Post("/api/vhosts/{domain}/enable", api.EnableVhost)
		r.Post("/api/vhosts/{domain}/disable", api.DisableVhost)
		r.Post("/api/vhosts/{domain}/ssl", api.EnableSSL)

		r.Get("/api/databases", api.ListDatabases)
		r.Post("/api/databases", api.CreateDatabase)
		r.Delete("/api/databases/{name}", api.DropDatabase)
		r.Get("/api/databases/{name}/tables", api.ListTables)

		r.Get("/api/files", api.ListFiles)
		r.Post("/api/files/mkdir", api.MakeDirectory)
		r.Delete("/api/files", api.DeleteFile)
		r.Post("/api/files/rename", api.RenameFile)
		r.Get("/api/files/read", api.ReadFile)
		r.Post("/api/files/write", api.WriteFile)
		r.Post("/api/files/upload", api.UploadFile)

		r.Get("/api/email/domains", api.ListMailDomains)
		r.Post("/api/email/domains", api.AddMailDomain)
		r.Delete("/api/email/domains/{domain}", api.DeleteMailDomain)
		r.Get("/api/email/mailboxes", api.ListMailboxes)
		r.Post("/api/email/mailboxes", api.CreateMailbox)
		r.Delete("/api/email/mailboxes/{email}", api.DeleteMailbox)
		r.Get("/api/email/queue", api.GetMailQueue)
		r.Post("/api/email/queue/flush", api.FlushMailQueue)

		r.Get("/api/dns", api.ListDNSZones)
		r.Post("/api/dns", api.CreateDNSZone)
		r.Get("/api/dns/{domain}", api.GetDNSZone)
		r.Delete("/api/dns/{domain}", api.DeleteDNSZone)
		r.Post("/api/dns/{domain}/records", api.AddDNSRecord)
		r.Delete("/api/dns/{domain}/records", api.DeleteDNSRecord)

		r.Get("/api/cron", api.ListCronJobs)
		r.Post("/api/cron", api.CreateCronJob)
		r.Put("/api/cron/{id}", api.UpdateCronJob)
		r.Delete("/api/cron/{id}", api.DeleteCronJob)
		r.Post("/api/cron/{id}/run", api.RunCronNow)

		r.Get("/api/ftp", api.ListFTPUsers)
		r.Post("/api/ftp", api.CreateFTPUser)
		r.Put("/api/ftp/{username}", api.UpdateFTPPassword)
		r.Delete("/api/ftp/{username}", api.DeleteFTPUser)
	})

	log.Printf("BLOGRON Panel API listening on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
