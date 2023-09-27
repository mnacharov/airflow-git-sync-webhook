package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
)

func getRoot(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprintln(w, "<h1>airflow-git-sync-webhook</h1>"+
		"<ul>"+
		"<li><a href=\"/api/git-sync-webhook\">/api/git-sync-webhook</a></li>"+
		"<li><a href=\"/metrics\">/metrics</a></li>"+
		"<li><a href=\"/api/github-sync-action\">/api/github-sync-action</a></li>"+
		"</ul>")
}

func main() {
	http.HandleFunc("/", getRoot)
	http.HandleFunc("/api/git-sync-webhook", gitSyncWebhook)
	// todo: http.HandleFunc("/api/github-sync-action", gitSyncWebhook)
	// todo: http.HandleFunc("/metrics", promMetrics)
	err := http.ListenAndServe(":3000", nil)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
	// todo: graceful shutdown
}
