package conversion

import (
	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
)

// TransportJSON represents transport configuration in JSON format
type TransportJSON struct {
	Type    string         `json:"type"`
	URL     string         `json:"url,omitempty"`
	Headers []KeyValueJSON `json:"headers,omitempty"`
}

// KeyValueJSON represents a key-value pair in JSON format
type KeyValueJSON struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Value       string `json:"value,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// ArgumentJSON represents an argument in JSON format
type ArgumentJSON struct {
	Name        string `json:"name"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
	Value       string `json:"value,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Multiple    bool   `json:"multiple,omitempty"`
}

// PackageJSON represents a package in JSON format
type PackageJSON struct {
	RegistryType         string         `json:"registryType"`
	RegistryBaseURL      string         `json:"registryBaseUrl,omitempty"`
	Identifier           string         `json:"identifier"`
	Version              string         `json:"version,omitempty"`
	FileSHA256           string         `json:"fileSha256,omitempty"`
	RuntimeHint          string         `json:"runtimeHint,omitempty"`
	Transport            TransportJSON  `json:"transport"`
	RuntimeArguments     []ArgumentJSON `json:"runtimeArguments,omitempty"`
	PackageArguments     []ArgumentJSON `json:"packageArguments,omitempty"`
	EnvironmentVariables []KeyValueJSON `json:"environmentVariables,omitempty"`
}

// RepositoryJSON represents a repository in JSON format
type RepositoryJSON struct {
	URL       string `json:"url,omitempty"`
	Source    string `json:"source,omitempty"`
	ID        string `json:"id,omitempty"`
	Subfolder string `json:"subfolder,omitempty"`
}

// TransportFromCRD converts a CRD Transport to TransportJSON
func TransportFromCRD(t agentregistryv1alpha1.Transport) TransportJSON {
	transport := TransportJSON{
		Type: t.Type,
		URL:  t.URL,
	}
	for _, h := range t.Headers {
		transport.Headers = append(transport.Headers, KeyValueJSON{
			Name:        h.Name,
			Description: h.Description,
			Value:       h.Value,
			Required:    h.Required,
		})
	}
	return transport
}

// KeyValueFromCRD converts a CRD KeyValueInput to KeyValueJSON
func KeyValueFromCRD(kv agentregistryv1alpha1.KeyValueInput) KeyValueJSON {
	return KeyValueJSON{
		Name:        kv.Name,
		Description: kv.Description,
		Value:       kv.Value,
		Required:    kv.Required,
	}
}

// ArgumentFromCRD converts a CRD Argument to ArgumentJSON
func ArgumentFromCRD(a agentregistryv1alpha1.Argument) ArgumentJSON {
	return ArgumentJSON{
		Name:        a.Name,
		Type:        a.Type,
		Description: a.Description,
		Value:       a.Value,
		Required:    a.Required,
		Multiple:    a.Multiple,
	}
}

// PackageFromCRD converts a CRD Package to PackageJSON
func PackageFromCRD(p agentregistryv1alpha1.Package) PackageJSON {
	pkg := PackageJSON{
		RegistryType:    p.RegistryType,
		RegistryBaseURL: p.RegistryBaseURL,
		Identifier:      p.Identifier,
		Version:         p.Version,
		FileSHA256:      p.FileSHA256,
		RuntimeHint:     p.RuntimeHint,
		Transport:       TransportFromCRD(p.Transport),
	}

	for _, a := range p.RuntimeArguments {
		pkg.RuntimeArguments = append(pkg.RuntimeArguments, ArgumentFromCRD(a))
	}
	for _, a := range p.PackageArguments {
		pkg.PackageArguments = append(pkg.PackageArguments, ArgumentFromCRD(a))
	}
	for _, e := range p.EnvironmentVariables {
		pkg.EnvironmentVariables = append(pkg.EnvironmentVariables, KeyValueFromCRD(e))
	}

	return pkg
}

// RepositoryFromCRD converts a CRD Repository to RepositoryJSON
func RepositoryFromCRD(r *agentregistryv1alpha1.Repository) *RepositoryJSON {
	if r == nil {
		return nil
	}
	return &RepositoryJSON{
		URL:       r.URL,
		Source:    r.Source,
		ID:        r.ID,
		Subfolder: r.Subfolder,
	}
}
