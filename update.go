package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/Masterminds/semver/v3"
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
	if Version == "dev" && !*forceUpdateCheck {
		fmt.Println("⚠️ Update-Check übersprungen (dev-Version)")
		return
	}

	var url string
	if *localUpdate {
		url = "http://localhost:9090/latest.json"
	} else {
		url = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", "chrisblk", "qrcodeapp")
	}

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("❌ Update-Check fehlgeschlagen:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Println("❌ Update-Check fehlgeschlagen: HTTP", resp.StatusCode)
		return
	}

	var rel release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		fmt.Println("❌ Fehler beim Lesen der Release-Daten:", err)
		return
	}

	// Semver-Vergleich
	currentVersion, err := semver.NewVersion(strings.TrimPrefix(Version, "v"))
	if err != nil {
		fmt.Println("❌ Ungültige aktuelle Version:", Version)
		return
	}
	latestVersion, err := semver.NewVersion(strings.TrimPrefix(rel.TagName, "v"))
	if err != nil {
		fmt.Println("❌ Ungültige Release-Version:", rel.TagName)
		return
	}

	if !latestVersion.GreaterThan(currentVersion) {
		fmt.Println("✅ Version aktuell:", Version)
		return
	}

	fmt.Printf("➡️ Neue Version verfügbar: %s (aktuell: %s)\n", latestVersion, currentVersion)

	// Passendes Binary finden
	target := fmt.Sprintf("qrapp-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		target += ".exe"
	}

	var downloadURL string
	for _, a := range rel.Assets {
		if a.Name == target {
			downloadURL = a.BrowserDownloadURL
			break
		}
	}
	if downloadURL == "" {
		fmt.Println("⚠️ Kein passendes Binary für diese Plattform gefunden")
		return
	}

	// Binary herunterladen
	tmpFile, err := os.CreateTemp("", "qrapp-update-*")
	if err != nil {
		fmt.Println("❌ Fehler beim Erstellen der Temp-Datei:", err)
		return
	}
	defer tmpFile.Close()

	resp2, err := http.Get(downloadURL)
	if err != nil {
		fmt.Println("❌ Fehler beim Download:", err)
		return
	}
	defer resp2.Body.Close()

	if _, err := io.Copy(tmpFile, resp2.Body); err != nil {
		fmt.Println("❌ Fehler beim Schreiben des Downloads:", err)
		return
	}

	// macOS/Linux: ausführbar machen
	if runtime.GOOS != "windows" {
		if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
			fmt.Println("❌ Fehler beim Setzen der Ausführungsrechte:", err)
			return
		}
	}

	// Update anwenden
	if err := update.Apply(tmpFile, update.Options{}); err != nil {
		fmt.Println("❌ Update fehlgeschlagen:", err)
		return
	}

	fmt.Println("✅ Update installiert → Neustart...")

	// App neu starten
	if err := exec.Command(os.Args[0]).Start(); err != nil {
		fmt.Println("❌ Fehler beim Neustart:", err)
	} else {
		os.Exit(0)
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
