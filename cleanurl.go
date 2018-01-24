package shuttle

import "net/url"

// extractCredentials extracts and scrubs basic auth credentials from a URL to
// ensure that they never get logged.
func extractCredentials(uri *url.URL) (cleanURL *url.URL, username string, password string) {
	cleanURL, _ = url.Parse(uri.String())
	if cleanURL.User != nil {
		username = uri.User.Username()
		password, _ = uri.User.Password()
	}
	cleanURL.User = nil
	return cleanURL, username, password
}
