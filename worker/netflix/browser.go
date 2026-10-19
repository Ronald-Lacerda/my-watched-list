package netflix

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"sort"
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
	Zone         *time.Location
	MaxPages     int
	LoginChannel string
	entryURL     string // sobrescrita somente pelos testes no mesmo pacote
}

func (b Browser) entry() string {
	if b.entryURL != "" {
		return b.entryURL
	}
	return "https://www.netflix.com/viewingactivity"
}

func openBrowser(ctx context.Context, state []byte, headed bool, channel string) (pw.BrowserContext, pw.Page, func(), error) {
	driver, err := pw.Run()
	if err != nil {
		return nil, nil, nil, ErrBrowser
	}
	opts := pw.BrowserTypeLaunchOptions{Headless: pw.Bool(!headed)}
	if channel != "" {
		opts.Channel = pw.String(channel)
	}
	browser, err := driver.Chromium.Launch(opts)
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
	value, err := page.Evaluate(`() => { const m=window.netflix?.reactContext?.models; const v=m?.vaModel?.data?.profileInfo; if(v) return {guid:v.guid,name:v.profileName}; const u=m?.memberContext?.data?.userInfo ?? m?.userInfo?.data ?? m?.auiUserContext?.data; return u ? {guid:u.guid,name:u.name} : null; }`)
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
	return b.LoginWithState(ctx, expectedName, nil, confirm)
}

// LoginWithState reaproveita a autenticação cifrada de outra conexão local;
// o perfil escolhido ainda precisa ser confirmado antes de salvar a nova sessão.
func (b Browser) LoginWithState(ctx context.Context, expectedName string, state []byte, confirm func() error) (Profile, []byte, error) {
	bc, page, close, err := openBrowser(ctx, state, true, b.LoginChannel)
	if err != nil {
		return Profile{}, nil, err
	}
	defer close()
	if _, err = page.Goto("https://www.netflix.com/browse"); err != nil {
		if code := regexp.MustCompile(`net::ERR_[A-Z_]+`).FindString(err.Error()); code != "" {
			return Profile{}, nil, errors.New("navegação Netflix: " + code)
		}
		return Profile{}, nil, ErrBrowser
	}
	_ = page.BringToFront()
	if err = confirm(); err != nil {
		return Profile{}, nil, err
	}
	for _, candidate := range bc.Pages() {
		if !candidate.IsClosed() && strings.HasPrefix(candidate.URL(), "https://www.netflix.com/") {
			page = candidate
		}
	}
	// A troca de perfil pode acontecer via SPA sem atualizar reactContext.
	// Recarregar confirma o GUID e nome atuais a partir da sessão do servidor.
	if _, err = page.Goto(b.entry()); err != nil {
		return Profile{}, nil, navigationError(err)
	}
	p, err := currentProfile(page)
	if err != nil {
		return Profile{}, nil, err
	}
	if !strings.EqualFold(strings.TrimSpace(p.Name), strings.TrimSpace(expectedName)) {
		return Profile{}, nil, errors.New("perfil selecionado não corresponde ao solicitado: " + p.Name)
	}
	fresh, err := stateJSON(bc)
	return p, fresh, err
}

func navigationError(err error) error {
	if code := regexp.MustCompile(`net::ERR_[A-Z_]+`).FindString(err.Error()); code != "" {
		return errors.New("navegação Netflix: " + code)
	}
	if strings.Contains(strings.ToLower(err.Error()), "closed") {
		return errors.New("a janela ou aba do coletor foi fechada")
	}
	if strings.Contains(strings.ToLower(err.Error()), "timeout") {
		return errors.New("tempo esgotado ao carregar a Netflix")
	}
	return ErrBrowser
}

func (b Browser) Fetch(ctx context.Context, state []byte, expectedProfile string) (Batch, error) {
	bc, page, close, err := openBrowser(ctx, state, false, "")
	if err != nil {
		return Batch{}, err
	}
	defer close()
	target, _ := url.Parse(b.entry())
	predicate := func(r pw.Response) bool {
		u, e := url.Parse(r.URL())
		if e != nil || u.Host != target.Host {
			return false
		}
		path := strings.ToLower(u.Path)
		activity := strings.Contains(path, "viewingactivity")
		if strings.Contains(path, "/pathevaluator/") {
			post, _ := r.Request().PostData()
			query, _ := url.QueryUnescape(u.RawQuery)
			activity = strings.Contains(strings.ToLower(post+query), "viewingactivity")
		}
		return activity && (strings.Contains(r.Headers()["content-type"], "json") || r.Status() == 401 || r.Status() == 403)
	}
	initial := make(chan pw.Response, 1)
	page.OnResponse(func(r pw.Response) {
		if predicate(r) {
			select {
			case initial <- r:
			default:
			}
		}
	})
	_, err = page.Goto(b.entry())
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
	var body []byte
	bootstrap, e := page.Evaluate(`() => window.netflix?.reactContext?.models?.vaModel?.data ?? null`)
	if e != nil {
		return Batch{}, collector.ErrParserChanged
	}
	if bootstrap != nil {
		body, err = json.Marshal(bootstrap)
	} else {
		select {
		case r := <-initial:
			body, err = responseBody(r)
		case <-ctx.Done():
			return Batch{}, ctx.Err()
		case <-time.After(15 * time.Second):
			return Batch{}, collector.ErrParserChanged
		}
	}
	if err != nil {
		return Batch{}, err
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
	unique := map[string]bool{}
	total := -1
	fingerprints := map[[32]byte]bool{}
	for n := 0; n < limit; n++ {
		if ctx.Err() != nil {
			return Batch{}, ctx.Err()
		}
		if len(body) > 16<<20 {
			return Batch{}, collector.ErrParserChanged
		}
		payload, guid, e := activityData(body)
		if e != nil {
			return Batch{}, e
		}
		if guid != "" && guid != expectedProfile {
			return Batch{}, collector.ErrLoginRequired
		}
		var progress struct {
			Total *int `json:"vhSize"`
		}
		if json.Unmarshal(payload, &progress) != nil || (progress.Total != nil && (*progress.Total < 0 || (total >= 0 && total != *progress.Total))) {
			return Batch{}, collector.ErrParserChanged
		}
		if progress.Total != nil {
			total = *progress.Total
		}
		events, e := Normalize(body, zone)
		if e != nil {
			return Batch{}, e
		}
		// A identidade da página ignora campos voláteis do JSON.
		keys := make([]string, len(events))
		for i, event := range events {
			keys[i] = event.SourceKey
			unique[event.SourceKey] = true
		}
		sort.Strings(keys)
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
		finished := count == 0
		if count == 1 {
			enabled, e := more.IsEnabled()
			if e != nil {
				return Batch{}, collector.ErrParserChanged
			}
			finished = !enabled
		}
		if finished {
			if total >= 0 && len(unique) != total {
				return Batch{}, collector.ErrParserChanged
			}
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
		response, err := page.ExpectResponse(predicate, func() error { return more.Click() })
		if strings.Contains(page.URL(), "/login") {
			return Batch{}, collector.ErrLoginRequired
		}
		if err != nil {
			return Batch{}, collector.ErrParserChanged
		}
		body, err = responseBody(response)
		if err != nil {
			return Batch{}, err
		}
		// Esperar a renderização após o JSON evita ler o botão antigo da página.
		if _, err = page.Evaluate(`() => new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)))`); err != nil {
			return Batch{}, collector.ErrParserChanged
		}
	}
	return Batch{}, collector.ErrParserChanged
}

func responseBody(response pw.Response) ([]byte, error) {
	if response.Status() == 401 || response.Status() == 403 {
		return nil, collector.ErrLoginRequired
	}
	if response.Status() != 200 {
		return nil, ErrBrowser
	}
	body, err := response.Body()
	if err != nil {
		return nil, ErrBrowser
	}
	return body, nil
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
