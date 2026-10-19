package metadata

import (
	"context"
	"encoding/json"
	"errors"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// NetflixTitle contém apenas material editorial público da página oficial da obra.
// Não representa uma correspondência TMDB nem usa a sessão do assinante.
type NetflixTitle struct {
	NetflixID, Name, Overview, PosterURL, MediaType, Language, SourceURL string
}

type NetflixClient struct {
	baseURL string
	http    *http.Client
}

func NewNetflix() *NetflixClient {
	return &NetflixClient{baseURL: "https://www.netflix.com", http: &http.Client{Timeout: 20 * time.Second}}
}

const netflixBodyLimit = 4 << 20

var (
	netflixIDPattern = regexp.MustCompile(`^[0-9]{1,20}$`)
	netflixScripts   = regexp.MustCompile(`(?is)<script\b([^>]*)>(.*?)</script\s*>`)
	netflixMetaTags  = regexp.MustCompile(`(?is)<(?:meta|link)\b([^>]*)>`)
	netflixAttrs     = regexp.MustCompile(`(?is)([a-z][a-z0-9:_-]*)\s*=\s*(?:"([^"]*)"|'([^']*)')`)
	netflixJSBytes   = regexp.MustCompile(`\\x([0-9a-fA-F]{2})`)
)

// Lookup prefere pt-BR. Uma página sem nome localizado tenta o catálogo GB;
// o idioma da fonte fica explícito, sem inventar/traduzir sinopses.
func (c *NetflixClient) Lookup(ctx context.Context, id string) (NetflixTitle, error) {
	if !netflixIDPattern.MatchString(id) {
		return NetflixTitle{}, ErrNotFound
	}
	var best NetflixTitle
	unavailable := false
	for _, region := range []struct{ path, language string }{{"br", "pt-BR"}, {"gb", "en-GB"}} {
		if ctx.Err() != nil {
			return NetflixTitle{}, errors.Join(ErrUnavailable, ctx.Err())
		}
		body, err := c.page(ctx, "/"+region.path+"/title/"+id, region.language)
		if err != nil {
			unavailable = unavailable || errors.Is(err, ErrUnavailable)
			continue
		}
		title, err := parseNetflixPage(body, id)
		if err != nil {
			unavailable = unavailable || errors.Is(err, ErrUnavailable)
			continue
		}
		title.Language = region.language
		title.SourceURL = "https://www.netflix.com/" + region.path + "/title/" + id
		if netflixCompleteness(title) > netflixCompleteness(best) {
			best = title
		}
		if title.PosterURL != "" && title.Overview != "" {
			return title, nil
		}
	}
	if ctx.Err() != nil {
		return NetflixTitle{}, errors.Join(ErrUnavailable, ctx.Err())
	}
	if best.Name != "" {
		return best, nil
	}
	if unavailable {
		return NetflixTitle{}, ErrUnavailable
	}
	return NetflixTitle{}, ErrNotFound
}

func netflixCompleteness(t NetflixTitle) int {
	n := 0
	for _, value := range []string{t.Name, t.Overview, t.PosterURL} {
		if value != "" {
			n++
		}
	}
	return n
}

func (c *NetflixClient) page(ctx context.Context, path, language string) ([]byte, error) {
	base, err := url.Parse(c.baseURL)
	if err != nil || base.User != nil || base.RawQuery != "" || base.Fragment != "" || (base.Path != "" && base.Path != "/") {
		return nil, ErrUnavailable
	}
	// A origem não é configurável pela API. Loopback existe somente para testes
	// internos do pacote, sem permitir uma origem remota alternativa.
	ip := net.ParseIP(base.Hostname())
	official := base.Scheme == "https" && base.Host == "www.netflix.com"
	localTest := base.Scheme == "http" && ip != nil && ip.IsLoopback()
	if !official && !localTest {
		return nil, ErrUnavailable
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.baseURL, "/")+path, nil)
	if err != nil {
		return nil, ErrUnavailable
	}
	req.Header.Set("Accept", "text/html")
	req.Header.Set("Accept-Language", language)
	client := *c.http
	client.Jar = nil
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	if client.Timeout == 0 || client.Timeout > 20*time.Second {
		client.Timeout = 20 * time.Second
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return nil, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, ErrUnavailable
	}
	if contentType := resp.Header.Get("Content-Type"); contentType != "" && !strings.Contains(strings.ToLower(contentType), "text/html") && !strings.Contains(strings.ToLower(contentType), "application/xhtml+xml") {
		return nil, ErrUnavailable
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, netflixBodyLimit+1))
	if err != nil || len(body) > netflixBodyLimit {
		return nil, ErrUnavailable
	}
	return body, nil
}

func netflixAttributes(tag string) map[string]string {
	attrs := map[string]string{}
	for _, m := range netflixAttrs.FindAllStringSubmatch(tag, -1) {
		attrs[strings.ToLower(m[1])] = html.UnescapeString(m[2] + m[3])
	}
	return attrs
}

func parseNetflixPage(body []byte, id string) (NetflixTitle, error) {
	malformed := false
	scripts := netflixScripts.FindAllSubmatch(body, -1)
	for _, script := range scripts {
		if strings.EqualFold(netflixAttributes(string(script[1]))["type"], "application/ld+json") {
			var data any
			if json.Unmarshal(script[2], &data) != nil {
				malformed = true
				continue
			}
			if title, ok := netflixLinkedData(data, id, 0); ok {
				return title, nil
			}
		}
	}
	// A página pública atual usa nmTitleGQL, cujo videoId e topLevelVideoId
	// vinculam título/sinopse/arte à obra, sem aproveitar cards recomendados.
	for _, script := range scripts {
		text := string(script[2])
		const assignment = "netflix.reactContext ="
		pos := strings.Index(text, assignment)
		if pos < 0 {
			continue
		}
		payload := strings.TrimSpace(text[pos+len(assignment):])
		payload = netflixJSBytes.ReplaceAllString(payload, `\u00${1}`)
		var data struct {
			Models struct {
				Title struct {
					Data struct {
						Copy    struct{ Title, Synopsis string }
						Artwork struct {
							Billboard struct{ Small string }
							HeroImage struct{ URL string }
						}
						MetaData struct {
							VideoID, TopLevelVideoID json.Number
							Type, NextPage           string
						}
					} `json:"data"`
				} `json:"nmTitleGQL"`
			} `json:"models"`
		}
		// Decode um único objeto JSON; o JS não é executado. O HTML pode conter
		// instruções adicionais depois do ';', que não fazem parte do objeto.
		if json.NewDecoder(strings.NewReader(payload)).Decode(&data) != nil {
			malformed = true
			continue
		}
		value := data.Models.Title.Data
		if value.MetaData.VideoID.String() != id || value.MetaData.TopLevelVideoID.String() != id || !netflixTitleURL(value.MetaData.NextPage, id) {
			continue
		}
		title := NetflixTitle{NetflixID: id, Name: netflixText(value.Copy.Title), Overview: netflixText(value.Copy.Synopsis), MediaType: netflixMediaType(value.MetaData.Type)}
		for _, image := range []string{value.Artwork.HeroImage.URL, value.Artwork.Billboard.Small} {
			if validNetflixImageURL(image) {
				title.PosterURL = image
				break
			}
		}
		if validNetflixTitle(title) {
			return title, nil
		}
	}
	if title, ok := netflixOpenGraph(body, id); ok {
		return title, nil
	}
	if malformed {
		return NetflixTitle{}, ErrUnavailable
	}
	return NetflixTitle{}, ErrNotFound
}

func netflixLinkedData(data any, id string, depth int) (NetflixTitle, bool) {
	if depth > 8 {
		return NetflixTitle{}, false
	}
	switch value := data.(type) {
	case []any:
		for _, item := range value {
			if title, ok := netflixLinkedData(item, id, depth+1); ok {
				return title, true
			}
		}
	case map[string]any:
		kind, _ := value["@type"].(string)
		name, _ := value["name"].(string)
		overview, _ := value["description"].(string)
		source, _ := value["url"].(string)
		if source == "" {
			source, _ = value["@id"].(string)
		}
		title := NetflixTitle{NetflixID: id, Name: netflixText(name), Overview: netflixText(overview), MediaType: netflixMediaType(kind), PosterURL: netflixLinkedImage(value["image"])}
		if netflixTitleURL(source, id) && validNetflixTitle(title) {
			return title, true
		}
		if graph, ok := value["@graph"]; ok {
			return netflixLinkedData(graph, id, depth+1)
		}
	}
	return NetflixTitle{}, false
}

func netflixLinkedImage(value any) string {
	switch image := value.(type) {
	case string:
		if validNetflixImageURL(image) {
			return image
		}
	case map[string]any:
		return netflixLinkedImage(image["url"])
	case []any:
		for _, entry := range image {
			if result := netflixLinkedImage(entry); result != "" {
				return result
			}
		}
	}
	return ""
}

func netflixOpenGraph(body []byte, id string) (NetflixTitle, bool) {
	values := map[string]string{}
	canonical := false
	for _, tag := range netflixMetaTags.FindAllSubmatch(body, -1) {
		attrs := netflixAttributes(string(tag[1]))
		if strings.EqualFold(attrs["rel"], "canonical") && netflixTitleURL(attrs["href"], id) {
			canonical = true
		}
		if key := strings.ToLower(attrs["property"]); strings.HasPrefix(key, "og:") {
			values[key] = attrs["content"]
		}
	}
	canonical = canonical || netflixTitleURL(values["og:url"], id)
	title := NetflixTitle{NetflixID: id, Name: netflixText(values["og:title"]), Overview: netflixText(values["og:description"]), MediaType: netflixMediaType(values["og:type"])}
	if validNetflixImageURL(values["og:image"]) {
		title.PosterURL = values["og:image"]
	}
	return title, canonical && validNetflixTitle(title)
}

func netflixText(text string) string {
	return strings.Join(strings.Fields(html.UnescapeString(text)), " ")
}

func validNetflixTitle(title NetflixTitle) bool {
	if title.Name == "" || len(title.Name) > 1000 || len(title.Overview) > 30000 || title.MediaType == "" {
		return false
	}
	switch strings.ToLower(title.Name) {
	case "netflix", "watch netflix", "assista à netflix", "netflix brasil", "netflix united kingdom":
		return false
	}
	return true
}

func netflixMediaType(kind string) string {
	switch strings.ToLower(kind) {
	case "movie", "video.movie":
		return "movie"
	case "show", "tvseries", "video.tv_show":
		return "tv"
	}
	return ""
}

func netflixTitleURL(raw, id string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Host != "www.netflix.com" || u.Fragment != "" {
		return false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	return (len(parts) == 2 && parts[0] == "title" && parts[1] == id) || (len(parts) == 3 && (parts[0] == "br" || parts[0] == "gb") && parts[1] == "title" && parts[2] == id)
}

func validNetflixImageURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" || u.Fragment != "" || u.Path == "" || u.Path == "/" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	for _, domain := range []string{"nflxso.net", "nflxext.com", "nflximg.net"} {
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return true
		}
	}
	return false
}
