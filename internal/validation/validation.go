package validation

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/mod/semver"
)

var (
	// semanticVersionRegex matches semantic versioning format (simplified)
	// Allows: 1.0.0, 1.0.0-alpha, 1.0.0-alpha.1, 1.0.0+build, 1.0.0-alpha+build
	semanticVersionRegex = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)

	// nameRegex matches valid Kubernetes resource names
	nameRegex = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

	// ErrInvalidVersion is returned when a version string is invalid
	ErrInvalidVersion = fmt.Errorf("invalid version format")

	// ErrInvalidURL is returned when a URL is invalid
	ErrInvalidURL = fmt.Errorf("invalid URL format")

	// ErrInvalidName is returned when a name is invalid
	ErrInvalidName = fmt.Errorf("invalid name format")
)

// ValidateSemanticVersion checks if a version string follows semantic versioning.
// It accepts both with and without 'v' prefix.
func ValidateSemanticVersion(version string) error {
	if version == "" {
		return fmt.Errorf("%w: version cannot be empty", ErrInvalidVersion)
	}

	// Remove 'v' prefix if present for validation
	v := version
	if strings.HasPrefix(v, "v") {
		v = v[1:]
	}

	if !semanticVersionRegex.MatchString(v) {
		return fmt.Errorf("%w: %q does not follow semantic versioning (expected format: 1.0.0, 1.0.0-alpha, etc.)", ErrInvalidVersion, version)
	}

	return nil
}

// IsSemanticVersion checks if a version string follows semantic versioning.
// This is a lighter check that doesn't return detailed errors.
func IsSemanticVersion(version string) bool {
	if version == "" {
		return false
	}
	v := version
	if strings.HasPrefix(v, "v") {
		v = v[1:]
	}
	return semanticVersionRegex.MatchString(v) && semver.IsValid("v"+v)
}

// ValidateURL checks if a string is a valid URL.
// It accepts both absolute URLs (with scheme) and relative URLs.
func ValidateURL(urlStr string) error {
	if urlStr == "" {
		return fmt.Errorf("%w: URL cannot be empty", ErrInvalidURL)
	}

	u, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}

	// If it has a scheme, ensure it's a valid one
	if u.Scheme != "" {
		validSchemes := map[string]bool{
			"http":   true,
			"https":  true,
			"git":    true,
			"ssh":    true,
			"file":   true,
			"oci":    true,
			"docker": true,
		}
		if !validSchemes[u.Scheme] {
			return fmt.Errorf("%w: unsupported scheme %q", ErrInvalidURL, u.Scheme)
		}
	}

	return nil
}

// ValidateRepositoryURL checks if a string is a valid repository URL.
// It requires either a valid URL with http/https/git scheme or a GitHub/GitLab shorthand (owner/repo).
func ValidateRepositoryURL(repoURL string) error {
	if repoURL == "" {
		return fmt.Errorf("%w: repository URL cannot be empty", ErrInvalidURL)
	}

	// Check for GitHub/GitLab shorthand (owner/repo)
	parts := strings.Split(repoURL, "/")
	if len(parts) == 2 && !strings.Contains(parts[0], ":") {
		// Looks like owner/repo format
		return nil
	}

	// Otherwise validate as URL
	if err := ValidateURL(repoURL); err != nil {
		return err
	}

	u, _ := url.Parse(repoURL)
	if u.Scheme != "" {
		validSchemes := map[string]bool{
			"http":  true,
			"https": true,
			"git":   true,
			"ssh":   true,
		}
		if !validSchemes[u.Scheme] {
			return fmt.Errorf("%w: repository URL must use http, https, git, or ssh scheme", ErrInvalidURL)
		}
	}

	return nil
}

// ValidateName checks if a name is valid for use as a Kubernetes resource name.
// It must consist of lowercase alphanumeric characters or '-', and must start and end with an alphanumeric character.
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: name cannot be empty", ErrInvalidName)
	}

	if len(name) > 253 {
		return fmt.Errorf("%w: name must be no more than 253 characters", ErrInvalidName)
	}

	if !nameRegex.MatchString(name) {
		return fmt.Errorf("%w: name must consist of lowercase alphanumeric characters or '-', and must start and end with an alphanumeric character", ErrInvalidName)
	}

	return nil
}

// ValidateServerName validates a server name for the catalog.
// Server names can contain slashes for namespacing (e.g., "github/modelcontextprotocol/filesystem").
func ValidateServerName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: server name cannot be empty", ErrInvalidName)
	}

	// Split by slash and validate each part
	parts := strings.Split(name, "/")
	for _, part := range parts {
		if part == "" {
			return fmt.Errorf("%w: server name parts cannot be empty", ErrInvalidName)
		}
		if err := ValidateName(part); err != nil {
			return fmt.Errorf("%w: invalid name part %q", ErrInvalidName, part)
		}
	}

	return nil
}

// SanitizeVersion removes invalid characters from a version string.
// It replaces invalid filesystem characters with hyphens.
func SanitizeVersion(version string) string {
	if version == "" {
		return ""
	}

	// Replace common invalid filesystem characters with hyphens
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "-",
		"?", "-",
		"\"", "-",
		"<", "-",
		">", "-",
		"|", "-",
	)
	sanitized := replacer.Replace(version)

	// Remove leading/trailing dots and spaces
	sanitized = strings.Trim(sanitized, ". ")

	// Replace multiple consecutive hyphens with a single hyphen
	for strings.Contains(sanitized, "--") {
		sanitized = strings.ReplaceAll(sanitized, "--", "-")
	}

	return sanitized
}

// SanitizeName converts a string to a valid Kubernetes resource name.
func SanitizeName(name string) string {
	name = strings.ToLower(name)
	var b strings.Builder
	prevDash := false
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if !prevDash {
			b.WriteRune('-')
			prevDash = true
		}
	}
	result := strings.Trim(b.String(), "-")
	if result == "" {
		return "resource"
	}
	return result
}
