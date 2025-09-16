# QR-App mit Auto-Update & Webinterface

Dieses Projekt ist eine kleine Go-Anwendung, die QR-Codes generiert und zusätzlich einen eingebauten Webserver startet, damit man die QR-Codes bequem im Browser erzeugen kann.
Außerdem besitzt die App einen **Update-Mechanismus**, der über GitHub Releases oder lokal getestet werden kann.

---

## Features

* Generiert QR-Codes mit optionalem Logo und Text-Label
* Webinterface mit HTML/htmx für einfache Bedienung
* Automatisches Update über GitHub Releases
* Lokaler Update-Server zum Testen
* Cross-Platform Builds (Windows, macOS, Linux)

---

## Flags (Programm-Parameter)

| Flag           | Beschreibung                                                                                 | Default            |
| -------------- | -------------------------------------------------------------------------------------------- | ------------------ |
| `-localupdate` | Führt beim Start eine **Update-Prüfung** durch (gegen GitHub oder lokalen Server)            | false              |
| `-serve`       | Startet einen **lokalen Update-Server** (Port 9090), um Updates ohne GitHub testen zu können | false              |
| `-testupdates` | Aktiviert Updates auch für Version `"dev"` (nur für Entwicklungszwecke)                      | false              |

---

## Typische Usecases

### 1. Webinterface starten

```bash
./qrapp
```

➡️ Öffnet automatisch den Browser (`http://localhost:8080/static`) und ermöglicht die QR-Code-Erstellung über die HTML-Oberfläche.

### 2. Automatisches Update von GitHub

```bash
./qrapp -localupdate
```

➡️ Prüft beim Start die aktuell installierte Version gegen die letzte GitHub-Release-Version (`latest.json`).
➡️ Wenn eine neuere Version vorhanden ist, wird sie heruntergeladen.

### 3. Lokalen Update-Server starten

```bash
./qrapp -serve
```

➡️ Startet einen lokalen Server auf **[http://localhost:9090](http://localhost:9090)**, der eine `latest.json` sowie ein Test-Binary ausliefert.
➡️ Nützlich, um den Update-Mechanismus ohne GitHub zu testen.

### 4. Update lokal testen (Client + Server)

* In einem Terminal:

  ```bash
  ./qrapp -serve
  ```

  → startet lokalen Update-Server mit Version `1.0.2`.
* In einem zweiten Terminal:

  ```bash
  go build -ldflags "-X main.Version=1.0.0" -o qrapp
  ./qrapp -localupdate
  ```

  → erkennt, dass `1.0.2` neuer ist als `1.0.0` und führt Update durch.

### 5. Entwicklerversion (`dev`) testen

```bash
go build -ldflags "-X main.Version=dev" -o qrapp
./qrapp -localupdate -testupdates
```

➡️ Normalerweise ignoriert die App Updates, wenn `Version = dev`.
➡️ Mit `-testupdates` wird trotzdem ein Update durchgeführt.

---

## Build & Releases

Das Projekt kann lokal gebaut werden:

```bash
go build -ldflags "-X main.Version=1.0.0" -o qrapp
```

In GitHub Actions wird automatisch für **Windows, macOS und Linux** gebaut, und die Artefakte in den Releases abgelegt.
Die Datei `latest.json` verweist auf die jeweils neueste Version.

---

## Sicherheit

* Updates sollten nur von **vertrauenswürdigen Quellen** geladen werden (GitHub Releases oder eigener Server).
* Optional kann man die Downloads per **SHA256-Hash** oder **GPG-Signatur** absichern.
* Für private Updateserver kann der Zugriff über **API-Keys oder presigned URLs** eingeschränkt werden.

---

## Hinweise

* Für lokale Tests kann die Version im Code hardcoded bleiben (`newVersion := "1.0.2"`).
* Alte Versionen immer auf `1.0.0` bauen, dann wird beim Testserver automatisch ein Update erkannt.
* Webinterface nutzt `htmx` für dynamische Updates der Version oder QR-Codes.

---

**Viel Spaß beim Testen und Verwenden der QR-App!**
