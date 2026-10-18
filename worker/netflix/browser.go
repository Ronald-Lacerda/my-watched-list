package netflix

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"

	pw "github.com/mxschmitt/playwright-go"
	"mywatchedlist/worker/collector"
)

var ErrBrowser = errors.New("navegador indisponível")

type Profile struct {
	ID   string `json:"guid"`
	Name string `json:"name"`
}
type Batch struct {
	Events []collector.Event
	State  []byte
}
type Browser struct {
	Zone     *time.Location
	MaxPages int
	entryURL string // sobrescrita somente pelos testes no mesmo pacote
}

func (b Browser) entry() string {
	if b.entryURL != "" {
		return b.entryURL
	}
	return "https://www.netflix.com/viewingactivity"
}

func openBrowser(ctx context.Context, state []byte, headed bool) (pw.BrowserContext, pw.Page, func(), error) {
	driver, err := pw.Run()
	if err != nil {
		return nil, nil, nil, ErrBrowser
	}
	browser, err := driver.Chromium.Launch(pw.BrowserTypeLaunchOptions{Headless: pw.Bool(!headed)})
	if err != nil {
		_ = driver.Stop()
		return nil, nil, nil, ErrBrowser
	}
	options := pw.BrowserNewContextOptions{Locale: pw.String("pt-BR"), TimezoneId: pw.String("America/Sao_Paulo"), AcceptDownloads: pw.Bool(false)}
	if len(state) > 0 {
		var s pw.StorageState
		if json.Unmarshal(state, &s) != nil {
			_ = browser.Close()
			_ = driver.Stop()
			return nil, nil, nil, collector.ErrLoginRequired
		}
		options.StorageState = s.ToOptionalStorageState()
	}
	bc, err := browser.NewContext(options)
	if err != nil {
		_ = browser.Close()
		_ = driver.Stop()
		return nil, nil, nil, ErrBrowser
	}
	page, err := bc.NewPage()
	if err != nil {
		_ = bc.Close()
		_ = browser.Close()
		_ = driver.Stop()
		return nil, nil, nil, ErrBrowser
	}
	bc.SetDefaultTimeout(15000)
	closed := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = browser.Close()
		case <-closed:
		}
	}()
	cleanup := func() { close(closed); _ = bc.Close(); _ = browser.Close(); _ = driver.Stop() }
	return bc, page, cleanup, nil
}

func currentProfile(page pw.Page) (Profile, error) {
	value, err := page.Evaluate(`() => { const u=window.netflix?.reactContext?.models?.memberContext?.data?.userInfo; return u ? {guid:u.guid,name:u.name} : null; }`)
	if err != nil {
		return Profile{}, collector.ErrParserChanged
	}
	raw, _ := json.Marshal(value)
	var p Profile
	if json.Unmarshal(raw, &p) != nil || p.ID == "" || p.Name == "" {
		return Profile{}, collector.ErrParserChanged
	}
	return p, nil
}

// Login abre uma janela isolada. O operador autentica e escolhe seu perfil;
// confirm apenas aguarda a confirmação local, nunca recebe senha/cookies.
func (b Browser) Login(ctx context.Context, expectedName string, confirm func() error) (Profile, []byte, error) {
	bc, page, close, err := openBrowser(ctx, nil, true)
	if err != nil {
		return Profile{}, nil, err
	}
	defer close()
	if _, err = page.Goto("https://www.netflix.com/browse"); err != nil {
		return Profile{}, nil, ErrBrowser
	}
	if err = confirm(); err != nil {
		return Profile{}, nil, err
	}
	p, err := currentProfile(page)
	if err != nil || p.Name != expectedName {
		return Profile{}, nil, collector.ErrLoginRequired
	}
	state, err := stateJSON(bc)
	return p, state, err
}

func (b Browser) Fetch(ctx context.Context, state []byte, expectedProfile string) (Batch, error) {
	bc, page, close, err := openBrowser(ctx, state, false)
	if err != nil {
		return Batch{}, err
	}
	defer close()
	target, _ := url.Parse(b.entry())
	predicate := func(r pw.Response) bool {
		u, e := url.Parse(r.URL())
		return e == nil && u.Host == target.Host && strings.Contains(strings.ToLower(u.Path), "viewingactivity") && (strings.Contains(r.Headers()["content-type"], "json") || r.Status() == 401 || r.Status() == 403)
	}
	response, err := page.ExpectResponse(predicate, func() error { _, e := page.Goto(b.entry()); return e })
	if strings.Contains(page.URL(), "/login") {
		return Batch{}, collector.ErrLoginRequired
	}
	if err != nil {
		return Batch{}, collector.ErrParserChanged
	}
	profile, err := currentProfile(page)
	if err != nil {
		return Batch{}, err
	}
	if profile.ID != expectedProfile {
		return Batch{}, collector.ErrLoginRequired
	}
	limit := b.MaxPages
	if limit == 0 {
		limit = 1000
	}
	zone := b.Zone
	if zone == nil {
		zone = time.UTC
	}
	all := make([]collector.Event, 0)
	fingerprints := map[[32]byte]bool{}
	for n := 0; n < limit; n++ {
		if ctx.Err() != nil {
			return Batch{}, ctx.Err()
		}
		if response.Status() == 401 || response.Status() == 403 {
			return Batch{}, collector.ErrLoginRequired
		}
		if response.Status() != 200 {
			return Batch{}, ErrBrowser
		}
		body, e := response.Body()
		if e != nil || len(body) > 16<<20 {
			return Batch{}, collector.ErrParserChanged
		}
		events, e := Normalize(body, zone)
		if e != nil {
			return Batch{}, e
		}
		// A identidade da página ignora campos voláteis do JSON.
		keys := make([]string, len(events))
		for i, event := range events {
			keys[i] = event.SourceKey
		}
		fp := sha256.Sum256([]byte(strings.Join(keys, "\n")))
		if fingerprints[fp] && len(events) > 0 {
			return Batch{}, collector.ErrParserChanged
		}
		fingerprints[fp] = true
		all = append(all, events...)
		more := page.GetByRole("button", pw.PageGetByRoleOptions{Name: regexp.MustCompile(`(?i)^(show more|mostrar mais|exibir mais|ver mais)$`)})
		count, e := more.Count()
		if e != nil {
			return Batch{}, collector.ErrParserChanged
		}
		if count == 0 {
			p, e := currentProfile(page)
			if e != nil || p.ID != expectedProfile {
				return Batch{}, collector.ErrLoginRequired
			}
			fresh, e := stateJSON(bc)
			if e != nil {
				return Batch{}, e
			}
			return Batch{Events: all, State: fresh}, nil
		}
		if count != 1 {
			return Batch{}, collector.ErrParserChanged
		}
		response, err = page.ExpectResponse(predicate, func() error { return more.Click() })
		if strings.Contains(page.URL(), "/login") {
			return Batch{}, collector.ErrLoginRequired
		}
		if err != nil {
			return Batch{}, collector.ErrParserChanged
		}
	}
	return Batch{}, collector.ErrParserChanged
}

func stateJSON(bc pw.BrowserContext) ([]byte, error) {
	s, err := bc.StorageState()
	if err != nil {
		return nil, ErrBrowser
	}
	s.Credentials = nil
	// Só são persistidos cookies e armazenamento da Netflix. Nenhum trace/HAR.
	cookies := s.Cookies[:0]
	for _, c := range s.Cookies {
		if c.Domain == "netflix.com" || strings.HasSuffix(c.Domain, ".netflix.com") {
			cookies = append(cookies, c)
		}
	}
	s.Cookies = cookies
	origins := s.Origins[:0]
	for _, o := range s.Origins {
		u, e := url.Parse(o.Origin)
		if e == nil && u.Scheme == "https" && (u.Hostname() == "netflix.com" || strings.HasSuffix(u.Hostname(), ".netflix.com")) {
			origins = append(origins, o)
		}
	}
	s.Origins = origins
	raw, err := json.Marshal(s)
	if err != nil {
		return nil, ErrBrowser
	}
	return raw, nil
}
