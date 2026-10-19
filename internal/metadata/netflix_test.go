package metadata

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Estrutura mínima do nmTitleGQL público, sem identificadores anônimos/sessão
// ou HTML real capturado. Título, sinopse e URL são dados sintéticos.
func netflixFixture(id, name, overview, kind, image string) string {
	return fmt.Sprintf(`<html><meta property="og:description" content="Assista a séries e filmes online."><script>window.netflix = window.netflix || {};
netflix.reactContext = {"models":{"nmTitleGQL":{"data":{"copy":{"title":%q,"synopsis":%q},"artwork":{"heroImage":{"url":%q}},"metaData":{"videoId":%s,"topLevelVideoId":%s,"nextPage":"https://www.netflix.com/br/title/%s","type":%q}}}}};</script></html>`, name, overview, image, id, id, id, kind)
}

func netflixTestClient(t *testing.T, handler http.HandlerFunc) *NetflixClient {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	c := NewNetflix()
	c.baseURL, c.http = srv.URL, srv.Client()
	return c
}

func TestNetflixOfficialReactContextBR(t *testing.T) {
	requests := 0
	c := netflixTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/br/title/70306300" || r.Header.Get("Accept-Language") != "pt-BR" || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
			t.Errorf("requisição não pública/localizada: %s %v", r.URL.Path, r.Header)
		}
		// O formato da Netflix usa escapes JavaScript \xHH além de JSON.
		fixture := netflixFixture("70306300", "Filme &amp; Amigos", "Uma história oficial.", "Show", "https://occ-0-541-1740.1.nflxso.net/dnm/art.jpg?r=synthetic")
		fmt.Fprint(w, strings.ReplaceAll(fixture, "Uma história", `Uma\x20história`))
	})
	// Mesmo um cookie jar acidentalmente anexado ao cliente é descartado.
	jar, _ := cookiejar.New(nil)
	u, _ := url.Parse(c.baseURL)
	jar.SetCookies(u, []*http.Cookie{{Name: "NetflixId", Value: "synthetic-never-sent"}})
	c.http.Jar = jar
	got, err := c.Lookup(context.Background(), "70306300")
	if err != nil || got.NetflixID != "70306300" || got.Name != "Filme & Amigos" || got.Overview != "Uma história oficial." || got.MediaType != "tv" || got.Language != "pt-BR" || got.SourceURL != "https://www.netflix.com/br/title/70306300" || got.PosterURL != "https://occ-0-541-1740.1.nflxso.net/dnm/art.jpg?r=synthetic" || requests != 1 {
		t.Fatalf("obra pública incorreta: %+v, %v; requests=%d", got, err, requests)
	}
}

func TestNetflixFallbackGBWithoutLocalizedName(t *testing.T) {
	var paths []string
	c := netflixTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if strings.HasPrefix(r.URL.Path, "/br/") {
			fmt.Fprint(w, netflixFixture("81087734", " ", " ", "Movie", "https://occ.example.nflxso.net/other.jpg"))
			return
		}
		if r.Header.Get("Accept-Language") != "en-GB" {
			t.Error("GB sem idioma explícito")
		}
		fmt.Fprint(w, netflixFixture("81087734", "A Synthetic Movie", "Official English synopsis.", "Movie", "https://occ.example.nflxso.net/real.jpg"))
	})
	got, err := c.Lookup(context.Background(), "81087734")
	if err != nil || got.Name != "A Synthetic Movie" || got.Overview != "Official English synopsis." || got.MediaType != "movie" || got.Language != "en-GB" || got.SourceURL != "https://www.netflix.com/gb/title/81087734" || strings.Join(paths, ",") != "/br/title/81087734,/gb/title/81087734" {
		t.Fatalf("fallback GB incorreto: %+v, %v, %v", got, err, paths)
	}
}

func TestNetflixLinkedDataAndCanonicalOpenGraph(t *testing.T) {
	for _, page := range []string{
		`<script type="application/ld+json">{"@graph":[{"@type":"Movie","url":"https://www.netflix.com/br/title/7","name":"Obra A &amp; B","description":"Sinopse oficial.","image":[{"url":"https://occ.a.nflxso.net/poster.jpg"}]}]}</script>`,
		`<meta content='Obra A &amp; B' property='og:title'><meta property='og:type' content='video.movie'><meta content='Sinopse oficial.' property='og:description'><meta property='og:image' content='https://occ.a.nflxso.net/poster.jpg'><link href='https://www.netflix.com/br/title/7' rel='canonical'>`,
	} {
		t.Run(page[:20], func(t *testing.T) {
			c := netflixTestClient(t, func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, page) })
			got, err := c.Lookup(context.Background(), "7")
			if err != nil || got.Name != "Obra A & B" || got.MediaType != "movie" || got.Overview != "Sinopse oficial." || got.PosterURL != "https://occ.a.nflxso.net/poster.jpg" {
				t.Fatalf("metadados oficiais perdidos: %+v, %v", got, err)
			}
		})
	}
}

func TestNetflixRejectsUnrelatedOrGenericPages(t *testing.T) {
	good := netflixFixture("7", "Obra correta", "Sinopse própria.", "Movie", "https://occ.a.nflxso.net/poster.jpg")
	for name, page := range map[string]string{
		"blank_name":       netflixFixture("7", "  ", "Sinopse", "Movie", "https://occ.a.nflxso.net/poster.jpg"),
		"landing":          `<meta property="og:title" content="Netflix"><meta property="og:description" content="Watch films online"><meta property="og:image" content="https://assets.nflxext.com/logo.jpg">`,
		"generic_with_id":  `<meta property="og:title" content="Netflix"><meta property="og:type" content="video.movie"><meta property="og:url" content="https://www.netflix.com/br/title/7">`,
		"wrong_video":      strings.Replace(good, `"videoId":7`, `"videoId":8`, 1),
		"wrong_parent":     strings.Replace(good, `"topLevelVideoId":7`, `"topLevelVideoId":8`, 1),
		"wrong_source":     strings.Replace(good, "netflix.com/br/title/7", "netflix.com/br/title/8", 1),
		"unknown_type":     strings.Replace(good, `"type":"Movie"`, `"type":"Episode"`, 1),
		"recommendations":  `<script>netflix.reactContext = {"models":{"recommendations":{"videoId":7,"title":"Obra errada"}}};</script>`,
		"wrong_jsonld_id":  `<script type="application/ld+json">{"@type":"Movie","url":"https://www.netflix.com/br/title/8","name":"Outra obra","image":"https://occ.a.nflxso.net/poster.jpg"}</script>`,
		"og_no_canonical":  `<meta property="og:title" content="Outra obra"><meta property="og:type" content="video.movie"><meta property="og:image" content="https://occ.a.nflxso.net/poster.jpg">`,
		"og_wrong_source":  `<link rel="canonical" href="https://www.netflix.com/br/title/8"><meta property="og:title" content="Outra obra"><meta property="og:type" content="video.movie">`,
		"og_external_site": `<link rel="canonical" href="https://www.netflix.com.evil.test/br/title/7"><meta property="og:title" content="Outra obra"><meta property="og:type" content="video.movie">`,
	} {
		t.Run(name, func(t *testing.T) {
			c := netflixTestClient(t, func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, page) })
			got, err := c.Lookup(context.Background(), "7")
			if !errors.Is(err, ErrNotFound) || got.Name != "" || got.PosterURL != "" {
				t.Fatalf("material não confirmado aceito: %+v %v", got, err)
			}
		})
	}
}

func TestNetflixImageOriginRestriction(t *testing.T) {
	for _, image := range []string{
		"http://occ.a.nflxso.net/poster.jpg", "//occ.a.nflxso.net/poster.jpg", "/poster.jpg", "data:image/png;base64,AA", "https://evil.test/poster.jpg", "https://nflxso.net.evil.test/poster.jpg", "https://evilnflxso.net/poster.jpg", "https://user:pass@occ.a.nflxso.net/poster.jpg", "https://occ.a.nflxso.net:8443/poster.jpg", "https://occ.a.nflxso.net/poster.jpg#fragment",
	} {
		t.Run(image, func(t *testing.T) {
			c := netflixTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, netflixFixture("7", "Obra", "Sinopse oficial.", "Movie", image))
			})
			got, err := c.Lookup(context.Background(), "7")
			if err != nil || got.Name != "Obra" || got.Overview != "Sinopse oficial." || got.PosterURL != "" {
				t.Fatalf("imagem insegura não removida: %+v %v", got, err)
			}
		})
	}
	for _, host := range []string{"occ-0-541-1740.1.nflxso.net", "assets.nflxext.com", "art.nflximg.net"} {
		c := netflixTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, netflixFixture("7", "Obra", "Sinopse.", "Movie", "https://"+host+"/poster.jpg"))
		})
		got, err := c.Lookup(context.Background(), "7")
		if err != nil || got.PosterURL == "" {
			t.Fatalf("CDN oficial recusada %s: %+v %v", host, got, err)
		}
	}
}

func TestNetflixHTTPFailureLimitsAndNoRedirect(t *testing.T) {
	var redirected atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { redirected.Add(1) }))
	defer target.Close()
	for _, mode := range []string{"redirect", "too_large", "malformed", "rate_limit", "wrong_content_type", "not_found"} {
		t.Run(mode, func(t *testing.T) {
			c := netflixTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				switch mode {
				case "redirect":
					http.Redirect(w, r, target.URL+"/private", http.StatusFound)
				case "too_large":
					fmt.Fprint(w, strings.Repeat("x", netflixBodyLimit+1))
				case "malformed":
					fmt.Fprint(w, `<script>netflix.reactContext = {bad secret-not-for-errors</script>`)
				case "rate_limit":
					w.WriteHeader(http.StatusTooManyRequests)
					fmt.Fprint(w, "secret-not-for-errors")
				case "wrong_content_type":
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprint(w, netflixFixture("7", "Obra", "Sinopse", "Movie", "https://occ.a.nflxso.net/poster.jpg"))
				case "not_found":
					w.WriteHeader(http.StatusNotFound)
				}
			})
			got, err := c.Lookup(context.Background(), "7")
			expected := ErrUnavailable
			if mode == "not_found" {
				expected = ErrNotFound
			}
			if !errors.Is(err, expected) || got.Name != "" || strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), c.baseURL) {
				t.Fatalf("falha HTTP/parser não sanitizada: %+v %v", got, err)
			}
		})
	}
	if redirected.Load() != 0 {
		t.Fatal("redirect executou requisição fora da origem")
	}
}

func TestNetflixInvalidIDAndCancellation(t *testing.T) {
	var requests atomic.Int32
	c := netflixTestClient(t, func(w http.ResponseWriter, r *http.Request) { requests.Add(1) })
	for _, id := range []string{"", "../7", "7?x=1", "https://evil.test/7", "7/8", " 7", "１２", strings.Repeat("1", 21)} {
		if _, err := c.Lookup(context.Background(), id); !errors.Is(err, ErrNotFound) {
			t.Fatalf("ID inválido aceito: %q: %v", id, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.Lookup(ctx, "7"); !errors.Is(err, context.Canceled) || !errors.Is(err, ErrUnavailable) {
		t.Fatalf("cancelamento perdido: %v", err)
	}
	if requests.Load() != 0 {
		t.Fatal("entrada inválida/cancelada executou HTTP")
	}
	started := make(chan struct{})
	c = netflixTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	})
	ctx, cancel = context.WithCancel(context.Background())
	go func() { <-started; cancel() }()
	begin := time.Now()
	if _, err := c.Lookup(ctx, "7"); !errors.Is(err, context.Canceled) || time.Since(begin) > 2*time.Second {
		t.Fatalf("HTTP não respeitou cancelamento: %v", err)
	}
}

func TestNetflixRejectsAlternativeRemoteOrigin(t *testing.T) {
	c := NewNetflix()
	c.baseURL = "https://example.invalid"
	if _, err := c.Lookup(context.Background(), "7"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("origem remota configurada aceita: %v", err)
	}
}
