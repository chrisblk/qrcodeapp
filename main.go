package main

import (
	"bytes"
	"embed"
	"encoding/base64"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"

	"github.com/nfnt/resize"
	"github.com/skip2/go-qrcode"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

//go:embed static/* assets/*
var embeddedFiles embed.FS

var (
	localUpdate      = flag.Bool("localupdate", false, "Starte lokalen Update-Server für Tests (Port 9090)")
	serve            = flag.Bool("serve", false, "Starte Webserver auf Port 9090 für QR-App")
	uiPort           = flag.Int("uiport", 8080, "Port für Web-UI")
	forceUpdateCheck = flag.Bool("forceupdatecheck", false, "Prüft Updates auch bei Version 'dev'")
)

func openBrowser(url string) {

	fmt.Printf("Opening browser at %s\n", url)
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin": // macOS
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default: // Linux, BSD, ...
		cmd = exec.Command("xdg-open", url)
	}

	if err := cmd.Start(); err != nil {
		fmt.Println("Fehler beim Öffnen des Browsers:", err)
	}
}

func main() {

	flag.Parse()

	if *serve {
		go buildAndStartLocalUpdateServer()
	}

	if *localUpdate {
		checkForUpdate()
	}

	fmt.Println("✅ QR-App läuft, Version:", Version)

	go startWebServer(*uiPort)

	select {}
}

func startWebServer(port int) {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(embeddedFiles)))
	mux.HandleFunc("/generate", generateHandler)
	// API: Version
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "%s", Version)
	})

	// Browser öffnen
	go openBrowser(fmt.Sprintf("http://localhost:%s/static", strconv.Itoa(port)))

	go func() {
		fmt.Println("🌍 Webserver läuft auf http://localhost:" + strconv.Itoa(port))
		if err := http.ListenAndServe(fmt.Sprintf(":%d", port), mux); err != nil {
			panic(err)
		}
	}()
}

func generateHandler(w http.ResponseWriter, r *http.Request) {
	content := r.FormValue("content")
	label := r.FormValue("label")
	file, _, _ := r.FormFile("logo")

	if content == "" {
		http.Error(w, "content fehlt", http.StatusBadRequest)
		return
	}

	size := 512
	radius := 40
	labelHeight := 70
	padding := 15

	// QR-Code
	qr, _ := qrcode.New(content, qrcode.Highest)
	qrImage := qr.Image(size)
	qrRounded := roundCornersAntiAlias(qrImage, radius)

	// Logo optional
	if file != nil {
		defer file.Close()
		logoImg, err := png.Decode(file)
		if err == nil {
			logoSize := qrRounded.Bounds().Dx() / 5
			logoResized := resize.Resize(uint(logoSize), 0, logoImg, resize.Lanczos3)

			offset := image.Pt(
				(qrRounded.Bounds().Dx()-logoResized.Bounds().Dx())/2,
				(qrRounded.Bounds().Dy()-logoResized.Bounds().Dy())/2,
			)

			bgRect := image.Rect(
				offset.X-8,
				offset.Y-8,
				offset.X+logoResized.Bounds().Dx()+8,
				offset.Y+logoResized.Bounds().Dy()+8,
			)
			draw.Draw(qrRounded, bgRect, &image.Uniform{color.White}, image.Point{}, draw.Src)
			draw.Draw(qrRounded, logoResized.Bounds().Add(offset), logoResized, image.Point{}, draw.Over)
		}
	}

	// Karte
	cardWidth := qrRounded.Bounds().Dx() + 2*padding
	cardHeight := qrRounded.Bounds().Dy() + labelHeight + 2*padding
	cardRect := image.Rect(0, 0, cardWidth, cardHeight)
	card := image.NewRGBA(cardRect)
	drawRoundedRect(card, cardRect, radius, color.Black)
	qrOffset := image.Pt(padding, padding)
	draw.Draw(card, qrRounded.Bounds().Add(qrOffset), qrRounded, image.Point{}, draw.Over)
	drawRoundedBorder(card, cardRect, radius, 8, color.Black)

	// Label
	labelRect := image.Rect(padding, cardHeight-labelHeight, cardWidth-padding, cardHeight-padding)
	_ = drawLabelOT(card, label, "assets/Roboto-Regular.ttf", labelRect)

	// In Base64 umwandeln
	var buf bytes.Buffer
	_ = png.Encode(&buf, card)
	base64Img := base64.StdEncoding.EncodeToString(buf.Bytes())

	// HTML zurückgeben (htmx ersetzt #output damit)
	fmt.Fprintf(w, `<img src="data:image/png;base64,%s" alt="QR Code" />`, base64Img)
}

// -------------------- Hilfsfunktionen --------------------

func roundCornersAntiAlias(img image.Image, radius int) *image.RGBA {
	bounds := img.Bounds()
	out := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if insideRoundedRect(x, y, bounds, radius) {
				out.Set(x, y, img.At(x, y))
			} else {
				out.Set(x, y, color.Transparent)
			}
		}
	}
	return out
}

func insideRoundedRect(x, y int, rect image.Rectangle, radius int) bool {
	if x >= rect.Min.X+radius && x < rect.Max.X-radius {
		return y >= rect.Min.Y && y < rect.Max.Y
	}
	if y >= rect.Min.Y+radius && y < rect.Max.Y-radius {
		return x >= rect.Min.X && x < rect.Max.X
	}
	dx := 0
	if x < rect.Min.X+radius {
		dx = rect.Min.X + radius - x
	} else if x >= rect.Max.X-radius {
		dx = x - (rect.Max.X - radius - 1)
	}
	dy := 0
	if y < rect.Min.Y+radius {
		dy = rect.Min.Y + radius - y
	} else if y >= rect.Max.Y-radius {
		dy = y - (rect.Max.Y - radius - 1)
	}
	return dx*dx+dy*dy <= radius*radius
}

func drawRoundedRect(img *image.RGBA, rect image.Rectangle, radius int, fill color.Color) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			if insideRoundedRect(x, y, rect, radius) {
				img.Set(x, y, fill)
			}
		}
	}
}

func drawRoundedBorder(img *image.RGBA, rect image.Rectangle, radius, thickness int, col color.Color) {
	for y := rect.Min.Y - thickness; y < rect.Max.Y+thickness; y++ {
		for x := rect.Min.X - thickness; x < rect.Max.X+thickness; x++ {
			if insideRoundedRect(x, y, rect, radius) &&
				!insideRoundedRect(x, y, rect.Inset(thickness), radius-thickness) {
				img.Set(x, y, col)
			}
		}
	}
}

func drawLabelOT(img *image.RGBA, text, fontPath string, rect image.Rectangle) error {
	fontBytes, err := embeddedFiles.ReadFile(fontPath)
	if err != nil {
		return err
	}
	otFont, err := opentype.Parse(fontBytes)
	if err != nil {
		return err
	}
	const dpi = 72
	fontSize := 48.0
	for fontSize > 5 {
		face, _ := opentype.NewFace(otFont, &opentype.FaceOptions{Size: fontSize, DPI: dpi, Hinting: font.HintingFull})
		d := &font.Drawer{Dst: img, Src: image.White, Face: face}
		textWidth := d.MeasureString(text).Ceil()
		if textWidth <= rect.Dx()-10 {
			pt := fixed.Point26_6{
				X: fixed.I(rect.Min.X + (rect.Dx()-textWidth)/2),
				Y: fixed.I(rect.Min.Y + rect.Dy()/2 + int(fontSize/3)),
			}
			d.Dot = pt
			d.DrawString(text)
			return nil
		}
		fontSize *= 0.9
	}
	return nil
}
