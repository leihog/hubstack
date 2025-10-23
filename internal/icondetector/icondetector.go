package icondetector

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Detector is an interface for icon auto-detection
type Detector interface {
	DetectIcon(serviceURL string) (string, error)
}

// iconCandidate represents a potential icon URL with metadata
type iconCandidate struct {
	url      string
	iconType string // e.g., "image/png", "image/svg+xml"
	size     int    // width in pixels, 0 if unknown
}

// HTTPDetector implements icon detection using HTTP requests
type HTTPDetector struct {
	client *http.Client
}

// NewHTTPDetector creates a new HTTP-based icon detector
func NewHTTPDetector() *HTTPDetector {
	return &HTTPDetector{
		client: &http.Client{
			Timeout: 5 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 3 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
	}
}

// DetectIcon attempts to find the best icon for a service URL
func (d *HTTPDetector) DetectIcon(serviceURL string) (string, error) {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return "", fmt.Errorf("invalid service URL: %w", err)
	}

	baseURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)

	// Try common favicon paths first (fast path) to avoid fetching the HTML
	commonPaths := []string{
		"/favicon.ico",
		"/favicon.png",
		"/apple-touch-icon.png",
		"/apple-touch-icon-precomposed.png",
	}

	for _, path := range commonPaths {
		iconURL := baseURL + path
		if d.checkIconExists(iconURL) {
			return iconURL, nil
		}
	}

	// If common paths don't work, fetch and parse the HTML
	icons, err := d.parseHTMLForIcons(serviceURL, baseURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	if len(icons) == 0 {
		return "", fmt.Errorf("no icons found")
	}

	bestIcon := selectBestIcon(icons)
	return bestIcon.url, nil
}

// checkIconExists checks if an icon URL is accessible and returns valid image content
func (d *HTTPDetector) checkIconExists(iconURL string) bool {
	resp, err := d.client.Head(iconURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	// Check Content-Type header to ensure we're getting an actual image
	// Some sites return 200 for everything, even 404 pages
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))

	validTypes := []string{
		"image/",                   // Any image type
		"application/octet-stream", // Sometimes used for .ico files
	}

	for _, validType := range validTypes {
		if strings.HasPrefix(contentType, validType) {
			return true
		}
	}

	return false
}

// parseHTMLForIcons fetches the HTML and extracts icon links
func (d *HTTPDetector) parseHTMLForIcons(serviceURL, baseURL string) ([]iconCandidate, error) {
	resp, err := d.client.Get(serviceURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// Check if a redirect occurred and update baseURL accordingly
	// resp.Request.URL contains the final URL after all redirects
	finalURL := resp.Request.URL.String()
	if finalURL != serviceURL {
		baseURL = fmt.Sprintf("%s://%s", resp.Request.URL.Scheme, resp.Request.URL.Host)
		serviceURL = finalURL
	}

	// Read the HTML (limit to first 100KB to avoid reading huge pages)
	body, err := io.ReadAll(io.LimitReader(resp.Body, 100*1024))
	if err != nil {
		return nil, err
	}

	html := string(body)
	var icons []iconCandidate

	// Look for <link> tags with rel="icon" or rel="shortcut icon"
	// This regex matches link tags with rel containing "icon"
	linkRegex := regexp.MustCompile(`(?i)<link[^>]*rel=["']([^"']*icon[^"']*)["'][^>]*>`)
	matches := linkRegex.FindAllString(html, -1)

	for _, match := range matches {
		icon := parseLinkTag(match, baseURL, serviceURL)
		if icon.url != "" {
			icons = append(icons, icon)
		}
	}

	return icons, nil
}

// parseLinkTag extracts icon information from a link tag
func parseLinkTag(linkTag, baseURL, pageURL string) iconCandidate {
	icon := iconCandidate{}

	// Extract href
	hrefRegex := regexp.MustCompile(`(?i)href=["']([^"']+)["']`)
	if matches := hrefRegex.FindStringSubmatch(linkTag); len(matches) > 1 {
		href := matches[1]
		icon.url = resolveURL(href, baseURL, pageURL)
	}

	// Extract type
	typeRegex := regexp.MustCompile(`(?i)type=["']([^"']+)["']`)
	if matches := typeRegex.FindStringSubmatch(linkTag); len(matches) > 1 {
		icon.iconType = strings.ToLower(matches[1])
	}

	// Extract size (look for sizes="64x64" or similar)
	sizeRegex := regexp.MustCompile(`(?i)sizes=["'](\d+)x\d+["']`)
	if matches := sizeRegex.FindStringSubmatch(linkTag); len(matches) > 1 {
		if size, err := strconv.Atoi(matches[1]); err == nil {
			icon.size = size
		}
	}

	return icon
}

// resolveURL converts a relative URL to absolute
func resolveURL(href, baseURL, pageURL string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}

	// Protocol-relative URL
	if strings.HasPrefix(href, "//") {
		parsedPage, err := url.Parse(pageURL)
		if err == nil {
			return parsedPage.Scheme + ":" + href
		}

		// We default to http as most internal services are not https (yet)
		return "http:" + href
	}

	// Absolute path
	if strings.HasPrefix(href, "/") {
		return baseURL + href
	}

	// Relative path - resolve against page URL
	parsedPage, err := url.Parse(pageURL)
	if err != nil {
		return baseURL + "/" + href
	}

	resolved, err := parsedPage.Parse(href)
	if err != nil {
		return baseURL + "/" + href
	}

	return resolved.String()
}

// selectBestIcon chooses the best icon from candidates
func selectBestIcon(icons []iconCandidate) iconCandidate {
	if len(icons) == 0 {
		return iconCandidate{}
	}

	targetSize := 64
	bestIcon := icons[0]
	bestScore := scoreIcon(bestIcon, targetSize)

	for _, icon := range icons[1:] {
		score := scoreIcon(icon, targetSize)
		if score > bestScore {
			bestScore = score
			bestIcon = icon
		}
	}

	return bestIcon
}

// scoreIcon assigns a score to an icon based on type and size preference
func scoreIcon(icon iconCandidate, targetSize int) float64 {
	score := 0.0

	// Prefer PNG over other formats
	switch icon.iconType {
	case "image/png":
		score += 100
	case "image/x-icon", "image/vnd.microsoft.icon":
		score += 80
	case "image/jpeg", "image/jpg":
		score += 60
	case "image/webp":
		score += 70
	case "image/svg+xml":
		score += 50 // TODO: should SVG be higher priority?
	default:
		score += 40
	}

	// Prefer sizes close to target (64x64)
	if icon.size > 0 {
		sizeDiff := math.Abs(float64(icon.size - targetSize))
		// Smaller difference = better score
		sizeScore := math.Max(100-sizeDiff, 0)
		score += sizeScore
	} else {
		// No size specified, give a neutral score
		score += 30
	}

	return score
}
