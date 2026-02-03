package validation

import (
	"testing"
)

func TestValidateSemanticVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		wantErr bool
	}{
		{"valid semver 1.0.0", "1.0.0", false},
		{"valid semver with v prefix", "v1.0.0", false},
		{"valid semver with prerelease", "1.0.0-alpha", false},
		{"valid semver with prerelease and build", "1.0.0-alpha+build", false},
		{"valid semver with build", "1.0.0+build.123", false},
		{"empty version", "", true},
		{"invalid format", "1.0", true},
		{"invalid characters", "1.0.0_invalid", true},
		{"just v prefix", "v", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSemanticVersion(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSemanticVersion(%q) error = %v, wantErr %v", tt.version, err, tt.wantErr)
			}
		})
	}
}

func TestIsSemanticVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{"valid semver", "1.0.0", true},
		{"valid semver with v", "v1.0.0", true},
		{"invalid semver", "1.0", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSemanticVersion(tt.version)
			if got != tt.want {
				t.Errorf("IsSemanticVersion(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid https URL", "https://example.com", false},
		{"valid http URL", "http://example.com", false},
		{"valid git URL", "git://github.com/user/repo", false},
		{"valid ssh URL", "ssh://git@github.com/user/repo", false},
		{"valid OCI URL", "oci://registry.io/repo/image", false},
		{"empty URL", "", true},
		{"invalid scheme", "ftp://example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestValidateRepositoryURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid https URL", "https://github.com/user/repo", false},
		{"valid git URL", "git://github.com/user/repo", false},
		{"GitHub shorthand", "user/repo", false},
		{"empty URL", "", true},
		{"invalid scheme", "ftp://example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRepositoryURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRepositoryURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid name", "my-resource", false},
		{"valid alphanumeric", "resource123", false},
		{"valid single char", "a", false},
		{"empty name", "", true},
		{"starts with dash", "-resource", true},
		{"ends with dash", "resource-", true},
		{"contains uppercase", "Resource", true},
		{"contains underscore", "resource_name", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"lowercase", "MyResource", "myresource"},
		{"with spaces", "my resource", "my-resource"},
		{"with underscores", "my_resource", "my-resource"},
		{"with slashes", "my/resource", "my-resource"},
		{"empty becomes resource", "", "resource"},
		{"multiple special chars", "a@@b##c", "a-b-c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeName(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeVersion(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"with slashes", "1.0.0/feature", "1.0.0-feature"},
		{"with colons", "1.0.0:build", "1.0.0-build"},
		{"with backslashes", "1.0.0\\build", "1.0.0-build"},
		{"multiple hyphens", "1.0.0--build", "1.0.0-build"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeVersion(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeVersion(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
