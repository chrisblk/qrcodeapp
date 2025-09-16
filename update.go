package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	"github.com/inconshreveable/go-update"
)

// --------------------------------------------------
// Version (Standardwert, kann via -ldflags überschrieben werden)
// --------------------------------------------------
var Version = "dev"

// --------------------------------------------------
// GitHub-kompatibles Release-Format
// --------------------------------------------------
type release struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		BrowserDownloadURL string `json:"browser_download_url"`
		Name               string `json:"name"`
	} `json:"assets"`
}

// --------------------------------------------------
// Update-Check (GitHub oder lokal)
// --------------------------------------------------
func checkForUpdate() {
	var url string

	if Version == "dev" && !*forceUpdateCheck {
		fmt.Println("⚠️ Update-Check übersprungen (dev-Version)")
		return
	}

	if *localUpdate {
		url = "http://localhost:9090/latest.json"
	} else {
		url = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", "chrisblk", "qrcode")
	}

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("❌ Update-Check fehlgeschlagen:", err)
		return
	}
	defer resp.Body.Close()

	var rel release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		fmt.Println("❌ Fehler beim Lesen der Release-Daten:", err)
		return
	}

	if rel.TagName != "v"+Version {
		fmt.Println("➡️ Neue Version gefunden:", rel.TagName)

		target := fmt.Sprintf("qrapp-%s-%s", runtime.GOOS, runtime.GOARCH)
		var downloadURL string
		for _, a := range rel.Assets {
			if a.Name == target {
				downloadURL = a.BrowserDownloadURL
				break
			}
		}

		if downloadURL == "" {
			fmt.Println("⚠️ Kein passendes Binary gefunden")
			return
		}

		resp, err := http.Get(downloadURL)
		if err != nil {
			fmt.Println("❌ Fehler beim Download:", err)
			return
		}
		defer resp.Body.Close()

		if err := update.Apply(resp.Body, update.Options{}); err != nil {
			fmt.Println("❌ Update fehlgeschlagen:", err)
			return
		}

		fmt.Println("✅ Update installiert → Neustart...")

		exec.Command(os.Args[0]).Start()
		os.Exit(0)
	} else {
		fmt.Println("✅ Version aktuell:", Version)
	}
}

// --------------------------------------------------
// Lokaler Build + Update-Server (nur Testmodus)
// --------------------------------------------------
func buildAndStartLocalUpdateServer() {

	mux := http.NewServeMux()

	newVersion := "1.0.2" // manuell oder später dynamisch via git tag

	// Binary-Namen generieren
	binaryName := fmt.Sprintf("qrapp-%s-%s", runtime.GOOS, runtime.GOARCH)

	// Neues Binary bauen
	fmt.Println("🔨 Baue Test-Binary:", binaryName, "mit Version", newVersion)
	cmd := exec.Command("go", "build", "-o", binaryName, "-ldflags", fmt.Sprintf("-X main.Version=%s", newVersion))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("❌ Fehler beim Bauen des Test-Binaries:", err)
		return
	}

	// JSON bereitstellen
	mux.HandleFunc("/latest.json", func(w http.ResponseWriter, r *http.Request) {
		rel := release{
			TagName: "v" + newVersion,
			Assets: []struct {
				BrowserDownloadURL string `json:"browser_download_url"`
				Name               string `json:"name"`
			}{
				{
					BrowserDownloadURL: "http://localhost:9090/" + binaryName,
					Name:               binaryName,
				},
			},
		}
		json.NewEncoder(w).Encode(rel)
	})

	// Binary ausliefern
	mux.Handle("/", http.FileServer(http.Dir(".")))

	go func() {
		fmt.Println("⬆️ Lokaler Update-Server läuft auf http://localhost:9090")
		if err := http.ListenAndServe(":9090", mux); err != nil {
			panic(err)
		}
	}()
}
