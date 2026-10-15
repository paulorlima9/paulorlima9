// Command render writes the profile SVGs for github.com/paulorlima9 and
// github.com/hyz-is, in the HYZIS design system (DESIGN.md of hyz.is):
// three values (#000, #FFF, #222), Geist, pills, 24px cards, 10% hairlines,
// and colour only inside real product screens.
//
//	go run . -profile paulo -out ../assets
//	go run . -profile hyzis -out ../../hyz-is/profile/assets
package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"html"
	"log"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

// ---------------------------------------------------------------- themes

type theme struct {
	name                   string
	ink, paper             string
	text2, text3           string
	line, lineStrong       string
	surface, tag, composer string
	chrome, glow           string
	shadow                 float64
	caption                string
}

var themes = []theme{
	{
		name: "light", ink: "#000000", paper: "#ffffff",
		text2: "rgba(0,0,0,.64)", text3: "rgba(0,0,0,.60)",
		line: "rgba(0,0,0,.10)", lineStrong: "rgba(0,0,0,.18)",
		surface: "rgba(34,34,34,.035)", tag: "rgba(0,0,0,.045)", composer: "rgba(34,34,34,.06)",
		chrome: "#ececec", glow: "rgba(34,34,34,.10)", shadow: .22,
		caption: "rgba(255,255,255,.92)",
	},
	{
		name: "dark", ink: "#ffffff", paper: "#000000",
		text2: "rgba(255,255,255,.64)", text3: "rgba(255,255,255,.56)",
		line: "rgba(255,255,255,.10)", lineStrong: "rgba(255,255,255,.18)",
		surface: "rgba(255,255,255,.03)", tag: "rgba(255,255,255,.06)", composer: "rgba(255,255,255,.07)",
		chrome: "#222222", glow: "rgba(255,255,255,.08)", shadow: .55,
		caption: "rgba(17,17,17,.92)",
	},
}

const ease = "cubic-bezier(0.16,1,0.3,1)"

// ---------------------------------------------------------------- type

type faces struct{ regular, medium, bold, mono *opentype.Font }

var ft faces

func loadFont(name string) *opentype.Font {
	b, err := os.ReadFile(filepath.Join("fonts", name))
	if err != nil {
		log.Fatal(err)
	}
	f, err := opentype.Parse(b)
	if err != nil {
		log.Fatal(err)
	}
	return f
}

// measure returns the advance width of s set in f at size px with tracking in em.
func measure(f *opentype.Font, size float64, tracking float64, s string) float64 {
	face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		log.Fatal(err)
	}
	defer face.Close()
	w := float64(font.MeasureString(face, s)) / 64
	return w + tracking*size*float64(len([]rune(s)))
}

func wrap(f *opentype.Font, size, tracking, width float64, s string) []string {
	var lines []string
	cur := ""
	for _, w := range strings.Fields(s) {
		try := strings.TrimSpace(cur + " " + w)
		if cur != "" && measure(f, size, tracking, try) > width {
			lines = append(lines, cur)
			cur = w
			continue
		}
		cur = try
	}
	return append(lines, cur)
}

func fontCSS() string {
	enc := func(name string) string {
		b, err := os.ReadFile(filepath.Join("fonts", name))
		if err != nil {
			log.Fatal(err)
		}
		return base64.StdEncoding.EncodeToString(b)
	}
	return fmt.Sprintf(`@font-face{font-family:G;src:url(data:font/woff2;base64,%s) format("woff2");font-weight:300 800}
@font-face{font-family:GM;src:url(data:font/woff2;base64,%s) format("woff2");font-weight:400 600}
text{font-family:G,"Helvetica Neue",Arial,sans-serif;font-feature-settings:"ss01","cv11";white-space:pre}
.m{font-family:GM,"SF Mono",Menlo,monospace}`, enc("geist-300-800-latin.woff2"), enc("geist-mono-400-600-latin.woff2"))
}

func esc(s string) string { return html.EscapeString(s) }

func f2(v float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", v), "0"), ".")
}

// ---------------------------------------------------------------- brand marks

var (
	pathRe = regexp.MustCompile(`<path\b[^>]*>`)
	dRe    = regexp.MustCompile(`\sd="([^"]+)"`)
	ruleRe = regexp.MustCompile(`fill-rule="([^"]+)"`)
	vbRe   = regexp.MustCompile(`viewBox="([^"]+)"`)
)

type mark struct {
	vbX, vbY, vbW, vbH float64
	paths              []string // d, fill-rule
	rules              []string
}

func loadMark(file string) mark {
	b, err := os.ReadFile(file)
	if err != nil {
		log.Fatal(err)
	}
	m := mark{vbW: 24, vbH: 24}
	if v := vbRe.FindSubmatch(b); v != nil {
		fmt.Sscanf(string(v[1]), "%f %f %f %f", &m.vbX, &m.vbY, &m.vbW, &m.vbH)
	}
	for _, p := range pathRe.FindAll(b, -1) {
		d := dRe.FindSubmatch(p)
		if d == nil {
			continue
		}
		rule := "nonzero"
		if r := ruleRe.FindSubmatch(p); r != nil {
			rule = string(r[1])
		}
		m.paths = append(m.paths, string(d[1]))
		m.rules = append(m.rules, rule)
	}
	return m
}

// draw places the mark with its height h at (x, y) and returns the svg and its width.
func (m mark) draw(x, y, h float64, fill string) (string, float64) {
	s := h / m.vbH
	var b strings.Builder
	fmt.Fprintf(&b, `<g transform="translate(%s %s) scale(%s) translate(%s %s)" fill="%s">`, f2(x), f2(y), strconv(s), f2(-m.vbX), f2(-m.vbY), fill)
	for i, d := range m.paths {
		fmt.Fprintf(&b, `<path fill-rule="%s" clip-rule="%s" d="%s"/>`, m.rules[i], m.rules[i], d)
	}
	b.WriteString(`</g>`)
	return b.String(), m.vbW * s
}

func strconv(v float64) string { return fmt.Sprintf("%.5f", v) }

// ---------------------------------------------------------------- svg shell

func svgOpen(w, h float64, title, desc string, css string) string {
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%s" height="%s" viewBox="0 0 %s %s" role="img" aria-labelledby="t d">
<title id="t">%s</title><desc id="d">%s</desc>
<style>%s
%s
@media (prefers-reduced-motion: reduce){*{animation:none!important}}
</style>
`, f2(w), f2(h), f2(w), f2(h), esc(title), esc(desc), fontCSS(), css)
}

func write(dir, name, s string) {
	if err := os.WriteFile(filepath.Join(dir, name), []byte(s+"</svg>\n"), 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%-28s %6.0f KB\n", name, float64(len(s))/1024)
}

// ---------------------------------------------------------------- hero

type screen struct {
	file, url, client, title string
}

type hero struct {
	brand     string // "paulo" or "hyzis"
	label     string
	headline  []string
	lead      string
	ask       string
	prompts   []string
	screens   []screen
	title     string
	desc      string
	hyzisMark mark
}

const (
	W        = 1280.0
	padX     = 64.0
	winW     = 500.0
	winX     = 716.0
	chromeH  = 36.0
	period   = 4.8
	typeStep = 0.045
	segment  = 5.6
)

func renderHero(h hero, t theme) string {
	var b strings.Builder
	var css strings.Builder

	// Left column.
	y := 92.0
	var left strings.Builder
	if h.brand == "hyzis" {
		m, mw := h.hyzisMark.draw(padX, y-26, 34, t.ink)
		left.WriteString(m)
		fmt.Fprintf(&left, `<text x="%s" y="%s" font-size="17" font-weight="700" letter-spacing="2.38" fill="%s">HYZIS</text>`, f2(padX+mw+12), f2(y), t.ink)
	} else {
		fmt.Fprintf(&left, `<text x="%s" y="%s" font-size="16" font-weight="500" fill="%s">%s</text>`, f2(padX), f2(y), t.text3, esc(h.label))
	}

	size := 62.0
	for _, l := range h.headline {
		for measure(ft.medium, size, -0.04, l) > 590 {
			size--
		}
	}
	y += 30 + size
	for i, l := range h.headline {
		if i > 0 {
			y += size * 1.04
		}
		fmt.Fprintf(&left, `<text x="%s" y="%s" font-size="%s" font-weight="500" letter-spacing="%s" fill="%s">%s</text>`,
			f2(padX), f2(y), f2(size), f2(-0.04*size), t.ink, esc(l))
	}
	y += 22
	for _, l := range wrap(ft.regular, 19, 0, 540, h.lead) {
		y += 30
		fmt.Fprintf(&left, `<text x="%s" y="%s" font-size="19" fill="%s">%s</text>`, f2(padX), f2(y), t.text2, esc(l))
	}

	// Tayi composer with typed prompts.
	cTop := y + 40
	cW, cH := 560.0, 60.0
	fmt.Fprintf(&left, `<rect x="%s" y="%s" width="%s" height="%s" rx="30" fill="%s" stroke="%s"/>`, f2(padX+.5), f2(cTop+.5), f2(cW-1), f2(cH-1), t.composer, t.lineStrong)
	tx, ty := padX+22, cTop+37
	fmt.Fprintf(&left, `<text x="%s" y="%s" font-size="17" fill="%s">%s</text>`, f2(tx), f2(ty), t.text3, esc(h.ask))
	px := tx + measure(ft.regular, 17, 0, h.ask)
	total := segment * float64(len(h.prompts))
	visible := segment - 0.5
	vis := visible / total * 100
	fmt.Fprintf(&css, "@keyframes ty{0%%,%s%%{fill-opacity:1;opacity:1}%s%%,100%%{fill-opacity:0;opacity:0}}\n", f2(vis), f2(vis+0.01))
	caretFrac := typeStep / total * 100
	fmt.Fprintf(&css, "@keyframes cr{0%%,%s%%{opacity:1}%s%%,100%%{opacity:0}}\n", f2(caretFrac), f2(caretFrac+0.01))
	fmt.Fprintf(&css, "@keyframes bl{0%%,49%%{opacity:1}50%%,100%%{opacity:0}}\n")
	fmt.Fprintf(&css, ".ty{animation:ty %ss linear infinite}.cr{animation:cr %ss linear infinite}.bl{animation:bl 1s steps(1) infinite}\n", f2(total), f2(total))
	fmt.Fprintf(&css, "@media (prefers-reduced-motion: reduce){.p0,.p0 tspan{opacity:1;fill-opacity:1}}\n")
	delay := func(s float64) string {
		if s <= 0 {
			return "0s"
		}
		return f2(s-total) + "s"
	}
	for k, p := range h.prompts {
		start := float64(k) * segment
		if px+measure(ft.regular, 17, 0, p)+8 > padX+cW-64 {
			log.Fatalf("prompt too wide: %q", p)
		}
		fmt.Fprintf(&left, `<g class="ty p%d" opacity="0" style="animation-delay:%s">`, k, delay(start))
		fmt.Fprintf(&left, `<text x="%s" y="%s" font-size="17" fill="%s">`, f2(px), f2(ty), t.ink)
		runes := []rune(p)
		for i, r := range runes {
			ch := string(r)
			if r == ' ' {
				ch = " "
			}
			fmt.Fprintf(&left, `<tspan class="ty" fill-opacity="0" style="animation-delay:%s">%s</tspan>`, delay(start+0.35+float64(i)*typeStep), esc(ch))
		}
		left.WriteString(`</text>`)
		// Caret: one per character while typing, then a blinking one.
		for i := range runes {
			cx := px + measure(ft.regular, 17, 0, string(runes[:i+1])) + 3
			fmt.Fprintf(&left, `<rect class="cr" opacity="0" style="animation-delay:%s" x="%s" y="%s" width="1.5" height="20" fill="%s"/>`,
				delay(start+0.35+float64(i)*typeStep), f2(cx), f2(ty-15), t.ink)
		}
		cx := px + measure(ft.regular, 17, 0, p) + 3
		done := start + 0.35 + float64(len(runes))*typeStep
		fmt.Fprintf(&left, `<g class="ty" opacity="0" style="animation-delay:%s"><rect class="bl" x="%s" y="%s" width="1.5" height="20" fill="%s"/></g>`,
			delay(done), f2(cx), f2(ty-15), t.ink)
		left.WriteString(`</g>`)
	}
	sx, sy := padX+cW-31, cTop+30
	fmt.Fprintf(&left, `<circle cx="%s" cy="%s" r="22" fill="%s"/>`, f2(sx), f2(sy), t.ink)
	fmt.Fprintf(&left, `<path d="M%s %sv-13m-5.5 5.5 5.5-5.5 5.5 5.5" fill="none" stroke="%s" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>`, f2(sx), f2(sy+6.5), t.paper)
	leftBottom := cTop + cH

	// Stage: the HYZIS product stage, stacked browser windows that cycle.
	n := len(h.screens)
	bodyH := math.Round(winW * 600 / 960)
	winH := chromeH + bodyH
	winY := 134.0
	ox, oy := winX+winW, winY
	pos := []string{
		"none",
		"translate(0px,-24px) scale(.94)",
		"translate(0px,-48px) scale(.88)",
		"translate(0px,-72px) scale(.82)",
	}
	op := []string{"1", ".72", ".4", "0"}
	move := 77.0
	css.WriteString(fmt.Sprintf(".sl{transform-box:view-box;transform-origin:%spx %spx;animation-duration:%ss;animation-iteration-count:infinite}\n", f2(ox), f2(oy), f2(period)))
	fmt.Fprintf(&css, "@keyframes s0{0%%,%s%%{transform:none;opacity:1;animation-timing-function:%s}100%%{transform:translate(-18px,30px) scale(1.02);opacity:0}}\n", f2(move), ease)
	for k := 1; k < 4; k++ {
		fmt.Fprintf(&css, "@keyframes s%d{0%%,%s%%{transform:%s;opacity:%s;animation-timing-function:%s}100%%{transform:%s;opacity:%s}}\n", k, f2(move), pos[k], op[k], ease, pos[k-1], op[k-1])
	}
	showTo := 100/float64(n) - 0.01
	fmt.Fprintf(&css, "@keyframes sh{0%%,%s%%{opacity:1}%s%%,100%%{opacity:0}}\n.sh{animation:sh %ss linear infinite}\n", f2(showTo), f2(showTo+0.01), f2(float64(n)*period))
	showDelay := func(m int) string {
		m = ((m % n) + n) % n
		if m == 0 {
			return "0s"
		}
		return f2(-float64(n-m)*period) + "s"
	}

	var stage strings.Builder
	// Achromatic glow behind the stage.
	fmt.Fprintf(&stage, `<ellipse cx="%s" cy="%s" rx="420" ry="300" fill="url(#glow)"/>`, f2(winX+winW/2), f2(winY+winH/2))
	images := make([]string, n)
	for j, s := range h.screens {
		raw, err := os.ReadFile(filepath.Join("screens", s.file))
		if err != nil {
			log.Fatal(err)
		}
		images[j] = base64.StdEncoding.EncodeToString(raw)
	}
	for k := 3; k >= 0; k-- {
		base := op[k]
		fmt.Fprintf(&stage, `<g class="sl" style="animation-name:s%d;transform:%s" opacity="%s">`, k, pos[k], base)
		fmt.Fprintf(&stage, `<g filter="url(#float)"><rect x="%s" y="%s" width="%s" height="%s" rx="16" fill="%s"/></g>`, f2(winX), f2(winY), f2(winW), f2(winH), t.chrome)
		fmt.Fprintf(&stage, `<g clip-path="url(#win)">`)
		for j, s := range h.screens {
			v := "0"
			if j == k%n {
				v = "1"
			}
			fmt.Fprintf(&stage, `<g class="sh" opacity="%s" style="animation-delay:%s">`, v, showDelay(j-k))
			fmt.Fprintf(&stage, `<use href="#img%d" x="%s" y="%s"/>`, j, f2(winX), f2(winY+chromeH))
			uw := measure(ft.mono, 12, 0, s.url) + 28
			fmt.Fprintf(&stage, `<rect x="%s" y="%s" width="%s" height="22" rx="11" fill="%s"/>`, f2(winX+winW/2-uw/2), f2(winY+7), f2(uw), t.line)
			fmt.Fprintf(&stage, `<text class="m" x="%s" y="%s" font-size="12" text-anchor="middle" fill="%s">%s</text>`, f2(winX+winW/2), f2(winY+22), t.text2, esc(s.url))
			stage.WriteString(`</g>`)
		}
		stage.WriteString(`</g>`)
		for d := 0; d < 3; d++ {
			fmt.Fprintf(&stage, `<circle cx="%s" cy="%s" r="5" fill="%s"/>`, f2(winX+20+float64(d)*16), f2(winY+18), t.lineStrong)
		}
		fmt.Fprintf(&stage, `<rect x="%s" y="%s" width="%s" height="%s" rx="16" fill="none" stroke="%s"/>`, f2(winX+.5), f2(winY+.5), f2(winW-1), f2(winH-1), t.lineStrong)
		fmt.Fprintf(&stage, `<line x1="%s" y1="%s" x2="%s" y2="%s" stroke="%s"/>`, f2(winX), f2(winY+chromeH+.5), f2(winX+winW), f2(winY+chromeH+.5), t.line)
		stage.WriteString(`</g>`)
	}

	// Caption card for the front window, with pill ticks.
	capY := winY + winH + 22
	capW, capH := 360.0, 66.0
	fmt.Fprintf(&stage, `<rect x="%s" y="%s" width="%s" height="%s" rx="18" fill="%s" stroke="%s"/>`, f2(winX+.5), f2(capY+.5), f2(capW-1), f2(capH-1), t.caption, t.lineStrong)
	for j, s := range h.screens {
		v := "0"
		if j == 0 {
			v = "1"
		}
		fmt.Fprintf(&stage, `<g class="sh" opacity="%s" style="animation-delay:%s">`, v, showDelay(j))
		fmt.Fprintf(&stage, `<text x="%s" y="%s" font-size="13" fill="%s">%s</text>`, f2(winX+20), f2(capY+26), t.text3, esc(s.client))
		fmt.Fprintf(&stage, `<text x="%s" y="%s" font-size="15" font-weight="500" letter-spacing="-0.2" fill="%s">%s</text>`, f2(winX+20), f2(capY+48), t.ink, esc(s.title))
		tickX := winX + float64(j)*34
		fmt.Fprintf(&stage, `<rect x="%s" y="%s" width="26" height="3" rx="1.5" fill="%s"/>`, f2(tickX), f2(capY+capH+20), t.ink)
		stage.WriteString(`</g>`)
	}
	fmt.Fprintf(&stage, `<path d="M%s %sh12m-5-5 5 5-5 5" fill="none" stroke="%s" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/>`, f2(winX+capW-38), f2(capY+capH/2), t.ink)
	for j := range h.screens {
		fmt.Fprintf(&stage, `<rect x="%s" y="%s" width="26" height="3" rx="1.5" fill="%s"/>`, f2(winX+float64(j)*34), f2(capY+capH+20), t.lineStrong)
	}
	// Active ticks must paint above the idle ones: move them last.
	stageSVG := stage.String()
	stageBottom := capY + capH + 23

	H := math.Max(leftBottom, stageBottom) + 56
	b.WriteString(svgOpen(W, H, h.title, h.desc, css.String()))
	fmt.Fprintf(&b, `<defs>
<radialGradient id="glow"><stop offset="0" stop-color="%s"/><stop offset="1" stop-color="%s" stop-opacity="0"/></radialGradient>
<filter id="float" x="-30%%" y="-30%%" width="160%%" height="180%%"><feDropShadow dx="0" dy="28" stdDeviation="26" flood-color="#000" flood-opacity="%s"/></filter>
<clipPath id="win"><rect x="%s" y="%s" width="%s" height="%s" rx="16"/></clipPath>
`, t.glow, t.glow, f2(t.shadow), f2(winX), f2(winY), f2(winW), f2(winH))
	for j, img := range images {
		fmt.Fprintf(&b, `<image id="img%d" width="%s" height="%s" preserveAspectRatio="xMidYMin slice" href="data:image/jpeg;base64,%s"/>`+"\n", j, f2(winW), f2(bodyH), img)
	}
	b.WriteString("</defs>\n")
	fmt.Fprintf(&b, `<rect x=".5" y=".5" width="%s" height="%s" rx="24" fill="%s" stroke="%s"/>`, f2(W-1), f2(H-1), t.surface, t.line)
	b.WriteString(left.String())
	b.WriteString(stageSVG)
	return b.String()
}

// ---------------------------------------------------------------- cards

type card struct {
	file, category, title, summary, url string
}

func renderCard(c card, t theme) string {
	const cw, ch = 640.0, 264.0
	var b strings.Builder
	b.WriteString(svgOpen(cw, ch, c.title, c.summary, ""))
	fmt.Fprintf(&b, `<rect x=".5" y=".5" width="%s" height="%s" rx="24" fill="%s" stroke="%s"/>`, f2(cw-1), f2(ch-1), t.surface, t.line)
	lw := measure(ft.medium, 13, 0, c.category) + 24
	fmt.Fprintf(&b, `<rect x="32.5" y="32.5" width="%s" height="27" rx="13.5" fill="none" stroke="%s"/>`, f2(lw), t.lineStrong)
	fmt.Fprintf(&b, `<text x="44" y="50.5" font-size="13" font-weight="500" fill="%s">%s</text>`, t.text2, esc(c.category))
	fmt.Fprintf(&b, `<circle cx="%s" cy="46" r="21.5" fill="none" stroke="%s"/>`, f2(cw-53.5), t.lineStrong)
	fmt.Fprintf(&b, `<path d="M%s 51l10-10m-7 0h7v7" fill="none" stroke="%s" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/>`, f2(cw-58.5), t.ink)
	fmt.Fprintf(&b, `<text x="32" y="122" font-size="30" font-weight="500" letter-spacing="-0.75" fill="%s">%s</text>`, t.ink, esc(c.title))
	lines := wrap(ft.regular, 17, 0, cw-64, c.summary)
	if len(lines) > 2 {
		log.Fatalf("card summary too long: %q", c.summary)
	}
	for i, l := range lines {
		fmt.Fprintf(&b, `<text x="32" y="%s" font-size="17" fill="%s">%s</text>`, f2(158+float64(i)*26), t.text2, esc(l))
	}
	uw := measure(ft.mono, 13, 0, c.url) + 28
	fmt.Fprintf(&b, `<rect x="32" y="206" width="%s" height="28" rx="14" fill="%s"/>`, f2(uw), t.tag)
	fmt.Fprintf(&b, `<text class="m" x="46" y="224.5" font-size="13" fill="%s">%s</text>`, t.text2, esc(c.url))
	return b.String()
}

// ---------------------------------------------------------------- stack

type chip struct{ icon, label string }
type group struct {
	label string
	chips []chip
}

func renderStack(groups []group, title string, t theme, marks map[string]mark) string {
	const sw = 1280.0
	const labelW = 168.0
	const chipH, gap, rowGap = 44.0, 10.0, 12.0
	var body strings.Builder
	y := 0.0
	for gi, g := range groups {
		if gi > 0 {
			y += 22
			fmt.Fprintf(&body, `<line x1="0" y1="%s" x2="%s" y2="%s" stroke="%s"/>`, f2(y-11+.5), f2(sw), f2(y-11+.5), t.line)
		}
		fmt.Fprintf(&body, `<text x="0" y="%s" font-size="16" font-weight="500" fill="%s">%s</text>`, f2(y+chipH/2+5.5), t.text3, esc(g.label))
		x := labelW
		for _, c := range g.chips {
			tw := measure(ft.medium, 15, 0, c.label)
			w := tw + 32
			if c.icon != "" {
				w += 26
			}
			if x+w > sw {
				x = labelW
				y += chipH + rowGap
			}
			fmt.Fprintf(&body, `<rect x="%s" y="%s" width="%s" height="%s" rx="%s" fill="%s" stroke="%s"/>`, f2(x+.5), f2(y+.5), f2(w-1), f2(chipH-1), f2(chipH/2), t.tag, t.line)
			lx := x + 16
			if c.icon != "" {
				m := marks[c.icon]
				ih := 18.0
				if m.vbW > m.vbH {
					ih = 18 * m.vbH / m.vbW
				}
				svg, iw := m.draw(lx, y+chipH/2-ih/2, ih, t.ink)
				body.WriteString(svg)
				_ = iw
				lx += 26
			}
			fmt.Fprintf(&body, `<text x="%s" y="%s" font-size="15" font-weight="500" fill="%s">%s</text>`, f2(lx), f2(y+chipH/2+5.2), t.ink, esc(c.label))
			x += w + gap
		}
		y += chipH
	}
	var names []string
	for _, g := range groups {
		for _, c := range g.chips {
			names = append(names, c.label)
		}
	}
	return svgOpen(sw, y+2, title, strings.Join(names, ", "), "") + body.String()
}

// ---------------------------------------------------------------- content

func main() {
	profile := flag.String("profile", "paulo", "paulo or hyzis")
	out := flag.String("out", "../assets", "output directory")
	flag.Parse()
	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}
	ft = faces{loadFont("Geist-Regular.ttf"), loadFont("Geist-Medium.ttf"), loadFont("Geist-Bold.ttf"), loadFont("GeistMono-Regular.ttf")}

	marks := map[string]mark{
		"arandu": loadMark("brand/arandu.svg"),
		"hyzis":  loadMark("brand/hyzis.svg"),
	}
	icons, _ := filepath.Glob("icons/*.svg")
	for _, f := range icons {
		marks[strings.TrimSuffix(filepath.Base(f), ".svg")] = loadMark(f)
	}

	var h hero
	var cards []card
	var groups []group
	switch *profile {
	case "paulo":
		h = hero{
			brand: "paulo", label: "Paulo R. Lima · Brasil",
			headline: []string{"Engenheiro de software,", "criador do Arandu,", "fundador da HYZIS."},
			lead:     "22 anos construindo software, IA e produtos digitais. Hoje escrevo em Go o framework, os módulos e os produtos que rodam sobre ele.",
			ask:      "Pergunte à Tayi: ",
			prompts:  []string{"como o Arandu isola cada tenant?", "quais módulos já são abertos?", "o que é a linguagem Yxy?"},
			screens: []screen{
				{"arandu.io.jpg", "arandu.io", "Arandu · Framework Go", "Construa em Go. Mantenha a estrutura."},
				{"tayi.ai.jpg", "tayi.ai", "Tayi · Pesquisa em IA", "Menos bits. Mais possibilidades."},
				{"perata.sh.jpg", "perata.sh", "Peráta · Monetização", "Pagamentos para produtos digitais"},
				{"yxy.dev.jpg", "yxy.dev", "Yxy · Linguagem de sistemas", "O código diz o que faz"},
				{"hyz.is.jpg", "hyz.is", "HYZIS · Engenharia de software", "Software, IA e produtos digitais"},
			},
			title: "Paulo R. Lima",
			desc:  "Engenheiro de software, criador do Arandu e fundador da HYZIS. Telas de arandu.io, tayi.ai, perata.sh, yxy.dev e hyz.is.",
		}
		cards = []card{
			{"card-arandu", "Framework Go", "Arandu", "Serviços, APIs e SaaS em Go, com a autorização chegando ao repositório como um tipo.", "github.com/arandu-io"},
			{"card-modules", "Open source", "Módulos HYZIS", "Mais de 20 módulos arandu-*: carteira, permissões, activity log, placas, rostos.", "github.com/hyz-is"},
			{"card-tayi", "Pesquisa em IA", "Tayi", "Modelos, agentes e infraestrutura abertos para engenharia de software.", "tayi.ai"},
			{"card-perata", "Monetização", "Peráta", "Checkout, assinaturas, licenças e repasse a vendedores para produtos digitais.", "perata.sh"},
			{"card-yxy", "Linguagem", "Yxy", "Linguagem de sistemas em que o código diz o que faz, para humanos e agentes.", "yxy.dev"},
			{"card-alpr", "Visão computacional", "alpr.sh", "Leitura de placas em tempo real, de câmeras IP ou pela API, sobre o arandu-alpr.", "alpr.sh"},
		}
		groups = []group{
			{"Linguagens", []chip{{"go", "Go"}, {"python", "Python"}, {"typescript", "TypeScript"}, {"javascript", "JavaScript"}, {"php", "PHP"}, {"dart", "Dart"}, {"yxy", "Yxy"}}},
			{"Web e apps", []chip{{"arandu", "Arandu"}, {"", "Kyse"}, {"htmx", "HTMX"}, {"laravel", "Laravel"}, {"react", "React"}, {"nextdotjs", "Next.js"}, {"vuedotjs", "Vue"}, {"tailwindcss", "Tailwind CSS"}, {"flutter", "Flutter"}}},
			{"IA e visão", []chip{{"pytorch", "PyTorch"}, {"onnx", "ONNX Runtime"}, {"nvidia", "TensorRT"}, {"huggingface", "Hugging Face"}, {"", "llama.cpp"}, {"", "YOLO"}, {"", "MCP"}}},
			{"Bancos de dados", []chip{{"postgresql", "PostgreSQL"}, {"mysql", "MySQL"}, {"mariadb", "MariaDB"}, {"redis", "Redis"}, {"", "Valkey"}, {"mongodb", "MongoDB"}, {"clickhouse", "ClickHouse"}, {"scylladb", "ScyllaDB"}, {"apachecassandra", "Cassandra"}, {"elasticsearch", "Elasticsearch"}, {"timescale", "TimescaleDB"}, {"cockroachlabs", "CockroachDB"}, {"duckdb", "DuckDB"}, {"sqlite", "SQLite"}}},
			{"Infra", []chip{{"dokploy", "Dokploy"}, {"docker", "Docker"}, {"docker", "Docker Swarm"}, {"traefikproxy", "Traefik"}, {"", "Nixpacks"}, {"linux", "Linux"}, {"ubuntu", "Ubuntu"}, {"debian", "Debian"}, {"cloudflare", "Cloudflare"}, {"git", "Git"}, {"githubactions", "GitHub Actions"}}},
		}
	case "hyzis":
		h = hero{
			brand: "hyzis", hyzisMark: marks["hyzis"],
			headline: []string{"Engenharia de software,", "IA e produtos digitais", "de alta performance."},
			lead:     "Plataformas web, aplicativos para iOS e Android, integrações e IA aplicada, do primeiro protótipo ao sistema em operação.",
			ask:      "Pergunte à Tayi: ",
			prompts:  []string{"vocês fazem apps?", "dá para ler placas em tempo real?", "como funciona um SaaS multi-tenant?"},
			screens: []screen{
				{"arandu.io.jpg", "arandu.io", "Arandu · Framework Go", "Construa em Go. Mantenha a estrutura."},
				{"alpr.sh.jpg", "alpr.sh", "alpr.sh · Visão computacional", "Leitura de placas em tempo real"},
				{"tayi.ai.jpg", "tayi.ai", "Tayi · Pesquisa em IA", "Menos bits. Mais possibilidades."},
				{"perata.sh.jpg", "perata.sh", "Peráta · Monetização", "Pagamentos para produtos digitais"},
				{"yxy.dev.jpg", "yxy.dev", "Yxy · Linguagem de sistemas", "O código diz o que faz"},
			},
			title: "HYZIS",
			desc:  "Engenharia de software, IA e produtos digitais de alta performance. Telas de arandu.io, alpr.sh, tayi.ai, perata.sh e yxy.dev.",
		}
		cards = []card{
			{"card-arandu", "Framework Go", "Arandu.io", "Construa em Go e mantenha a estrutura ao crescer. Código aberto, licença MIT.", "arandu.io"},
			{"card-tayi", "Pesquisa em IA", "Tayi.ai", "Pesquisa em adaptação e baixa precisão para modelos de linguagem.", "tayi.ai"},
			{"card-perata", "Monetização", "Perata.sh", "Produtos, preços e clientes de SaaS, APIs e produtos de IA em um só lugar.", "perata.sh"},
			{"card-modules", "Open source", "Módulos Arandu", "Mais de 20 módulos que a HYZIS desenvolve e publica em código aberto.", "github.com/hyz-is"},
		}
		groups = []group{
			{"Plataforma", []chip{{"go", "Go"}, {"arandu", "Arandu"}, {"", "Kyse"}, {"htmx", "HTMX"}, {"typescript", "TypeScript"}, {"flutter", "Flutter"}}},
			{"IA e visão", []chip{{"pytorch", "PyTorch"}, {"onnx", "ONNX Runtime"}, {"nvidia", "TensorRT"}, {"huggingface", "Hugging Face"}, {"", "llama.cpp"}, {"", "MCP"}}},
			{"Bancos de dados", []chip{{"postgresql", "PostgreSQL"}, {"mysql", "MySQL"}, {"mariadb", "MariaDB"}, {"redis", "Redis"}, {"", "Valkey"}, {"mongodb", "MongoDB"}, {"clickhouse", "ClickHouse"}, {"scylladb", "ScyllaDB"}, {"apachecassandra", "Cassandra"}, {"elasticsearch", "Elasticsearch"}, {"timescale", "TimescaleDB"}, {"cockroachlabs", "CockroachDB"}, {"duckdb", "DuckDB"}, {"sqlite", "SQLite"}}},
			{"Infra", []chip{{"dokploy", "Dokploy"}, {"docker", "Docker"}, {"docker", "Docker Swarm"}, {"traefikproxy", "Traefik"}, {"", "Nixpacks"}, {"linux", "Linux"}, {"ubuntu", "Ubuntu"}, {"debian", "Debian"}, {"cloudflare", "Cloudflare"}, {"git", "Git"}, {"githubactions", "GitHub Actions"}}},
		}
	default:
		log.Fatalf("unknown profile %q", *profile)
	}

	for _, t := range themes {
		write(*out, "hero-"+t.name+".svg", renderHero(h, t))
		write(*out, "stack-"+t.name+".svg", renderStack(groups, "Stack", t, marks))
		for _, c := range cards {
			write(*out, c.file+"-"+t.name+".svg", renderCard(c, t))
		}
	}
}
