package shuttle

import "net/url"

// extractCredentials extracts and scrubs basic auth credentials from a URL to
// ensure that they never get logged.
func extractCredentials(uri string) (cleanURL *url.URL, username string, password string, err error) {
	cleanURL, err = url.Parse(uri)
	if err != nil {
		return
	}

	if cleanURL.User != nil {
		username = cleanURL.User.Username()
		password, _ = cleanURL.User.Password()
	}
	cleanURL.User = nil
	return cleanURL, username, password, nil
}
