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

// TransportToCRD converts a TransportJSON to a CRD Transport
func TransportToCRD(t TransportJSON) agentregistryv1alpha1.Transport {
	transport := agentregistryv1alpha1.Transport{
		Type: t.Type,
		URL:  t.URL,
	}
	for _, h := range t.Headers {
		transport.Headers = append(transport.Headers, agentregistryv1alpha1.KeyValueInput{
			Name:        h.Name,
			Description: h.Description,
			Value:       h.Value,
			Required:    h.Required,
		})
	}
	return transport
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

// KeyValueToCRD converts a KeyValueJSON to a CRD KeyValueInput
func KeyValueToCRD(kv KeyValueJSON) agentregistryv1alpha1.KeyValueInput {
	return agentregistryv1alpha1.KeyValueInput{
		Name:        kv.Name,
		Description: kv.Description,
		Value:       kv.Value,
		Required:    kv.Required,
	}
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

// ArgumentToCRD converts an ArgumentJSON to a CRD Argument
func ArgumentToCRD(a ArgumentJSON) agentregistryv1alpha1.Argument {
	return agentregistryv1alpha1.Argument{
		Name:        a.Name,
		Type:        a.Type,
		Description: a.Description,
		Value:       a.Value,
		Required:    a.Required,
		Multiple:    a.Multiple,
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

// PackageToCRD converts a PackageJSON to a CRD Package
func PackageToCRD(p PackageJSON) agentregistryv1alpha1.Package {
	pkg := agentregistryv1alpha1.Package{
		RegistryType:    p.RegistryType,
		RegistryBaseURL: p.RegistryBaseURL,
		Identifier:      p.Identifier,
		Version:         p.Version,
		FileSHA256:      p.FileSHA256,
		RuntimeHint:     p.RuntimeHint,
		Transport:       TransportToCRD(p.Transport),
	}

	for _, a := range p.RuntimeArguments {
		pkg.RuntimeArguments = append(pkg.RuntimeArguments, ArgumentToCRD(a))
	}
	for _, a := range p.PackageArguments {
		pkg.PackageArguments = append(pkg.PackageArguments, ArgumentToCRD(a))
	}
	for _, e := range p.EnvironmentVariables {
		pkg.EnvironmentVariables = append(pkg.EnvironmentVariables, KeyValueToCRD(e))
	}

	return pkg
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

// RepositoryToCRD converts a RepositoryJSON to a CRD Repository
func RepositoryToCRD(r *RepositoryJSON) *agentregistryv1alpha1.Repository {
	if r == nil {
		return nil
	}
	return &agentregistryv1alpha1.Repository{
		URL:       r.URL,
		Source:    r.Source,
		ID:        r.ID,
		Subfolder: r.Subfolder,
	}
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

// TransportListToCRD converts a slice of TransportJSON to CRD Transport slice
func TransportListToCRD(transports []TransportJSON) []agentregistryv1alpha1.Transport {
	result := make([]agentregistryv1alpha1.Transport, len(transports))
	for i, t := range transports {
		result[i] = TransportToCRD(t)
	}
	return result
}

// TransportListFromCRD converts a CRD Transport slice to TransportJSON slice
func TransportListFromCRD(transports []agentregistryv1alpha1.Transport) []TransportJSON {
	result := make([]TransportJSON, len(transports))
	for i, t := range transports {
		result[i] = TransportFromCRD(t)
	}
	return result
}

// PackageListToCRD converts a slice of PackageJSON to CRD Package slice
func PackageListToCRD(packages []PackageJSON) []agentregistryv1alpha1.Package {
	result := make([]agentregistryv1alpha1.Package, len(packages))
	for i, p := range packages {
		result[i] = PackageToCRD(p)
	}
	return result
}

// PackageListFromCRD converts a CRD Package slice to PackageJSON slice
func PackageListFromCRD(packages []agentregistryv1alpha1.Package) []PackageJSON {
	result := make([]PackageJSON, len(packages))
	for i, p := range packages {
		result[i] = PackageFromCRD(p)
	}
	return result
}
