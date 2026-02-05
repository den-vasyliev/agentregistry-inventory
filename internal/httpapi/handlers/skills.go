package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
	"github.com/agentregistry-dev/agentregistry/internal/controller"
)

// SkillHandler handles skill catalog operations
type SkillHandler struct {
	client client.Client
	cache  cache.Cache
	logger zerolog.Logger
}

// NewSkillHandler creates a new skill handler
func NewSkillHandler(c client.Client, cache cache.Cache, logger zerolog.Logger) *SkillHandler {
	return &SkillHandler{
		client: c,
		cache:  cache,
		logger: logger.With().Str("handler", "skills").Logger(),
	}
}

// Skill response types
type SkillJSON struct {
	Name        string               `json:"name"`
	Version     string               `json:"version"`
	Title       string               `json:"title,omitempty"`
	Category    string               `json:"category,omitempty"`
	Description string               `json:"description,omitempty"`
	WebsiteURL  string               `json:"websiteUrl,omitempty"`
	Repository  *SkillRepositoryJSON `json:"repository,omitempty"`
	Packages    []SkillPackageJSON   `json:"packages,omitempty"`
	Remotes     []SkillRemoteJSON    `json:"remotes,omitempty"`
}

type SkillRepositoryJSON struct {
	URL    string `json:"url,omitempty"`
	Source string `json:"source,omitempty"`
}

type SkillPackageJSON struct {
	RegistryType string                     `json:"registryType"`
	Identifier   string                     `json:"identifier"`
	Version      string                     `json:"version,omitempty"`
	Transport    *SkillPackageTransportJSON `json:"transport,omitempty"`
}

type SkillPackageTransportJSON struct {
	Type string `json:"type"`
}

type SkillRemoteJSON struct {
	URL string `json:"url"`
}

type SkillUsageRefJSON struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Kind      string `json:"kind,omitempty"`
}

type SkillMeta struct {
	Official          *OfficialMeta          `json:"io.modelcontextprotocol.registry/official,omitempty"`
	PublisherProvided map[string]interface{} `json:"io.modelcontextprotocol.registry/publisher-provided,omitempty"`
	UsedBy            []SkillUsageRefJSON    `json:"usedBy,omitempty"`
}

type SkillResponse struct {
	Skill SkillJSON `json:"skill"`
	Meta  SkillMeta `json:"_meta"`
}

type SkillListResponse struct {
	Skills   []SkillResponse `json:"skills"`
	Metadata ListMetadata    `json:"metadata"`
}

// Input types
type ListSkillsInput struct {
	Cursor   string `query:"cursor" json:"cursor,omitempty"`
	Limit    int    `query:"limit" json:"limit,omitempty" default:"30" minimum:"1" maximum:"100"`
	Search   string `query:"search" json:"search,omitempty"`
	Category string `query:"category" json:"category,omitempty"`
	Version  string `query:"version" json:"version,omitempty"`
}

type SkillDetailInput struct {
	SkillName string `path:"skillName" json:"skillName"`
}

type SkillVersionDetailInput struct {
	SkillName string `path:"skillName" json:"skillName"`
	Version   string `path:"version" json:"version"`
}

type CreateSkillInput struct {
	Body SkillJSON
}

// RegisterRoutes registers skill endpoints
func (h *SkillHandler) RegisterRoutes(api huma.API, pathPrefix string, isAdmin bool) {
	tags := []string{"skills"}
	if isAdmin {
		tags = append(tags, "admin")
	}

	// List skills
	huma.Register(api, huma.Operation{
		OperationID: "list-skills" + strings.ReplaceAll(pathPrefix, "/", "-"),
		Method:      http.MethodGet,
		Path:        pathPrefix + "/skills",
		Summary:     "List skills",
		Tags:        tags,
	}, func(ctx context.Context, input *ListSkillsInput) (*Response[SkillListResponse], error) {
		return h.listSkills(ctx, input, isAdmin)
	})

	// Get skill by name
	huma.Register(api, huma.Operation{
		OperationID: "get-skill" + strings.ReplaceAll(pathPrefix, "/", "-"),
		Method:      http.MethodGet,
		Path:        pathPrefix + "/skills/{skillName}",
		Summary:     "Get skill details",
		Tags:        tags,
	}, func(ctx context.Context, input *SkillDetailInput) (*Response[SkillResponse], error) {
		return h.getSkill(ctx, input, isAdmin)
	})

	// Get specific version
	huma.Register(api, huma.Operation{
		OperationID: "get-skill-version" + strings.ReplaceAll(pathPrefix, "/", "-"),
		Method:      http.MethodGet,
		Path:        pathPrefix + "/skills/{skillName}/versions/{version}",
		Summary:     "Get specific skill version",
		Tags:        tags,
	}, func(ctx context.Context, input *SkillVersionDetailInput) (*Response[SkillResponse], error) {
		return h.getSkillVersion(ctx, input, isAdmin)
	})

	// Create skill
	huma.Register(api, huma.Operation{
		OperationID: "push-skill" + strings.ReplaceAll(pathPrefix, "/", "-"),
		Method:      http.MethodPost,
		Path:        pathPrefix + "/skills/push",
		Summary:     "Push skill",
		Tags:        tags,
	}, func(ctx context.Context, input *CreateSkillInput) (*Response[SkillResponse], error) {
		return h.createSkill(ctx, input)
	})

	// Admin-only endpoints
	if isAdmin {
		// Create skill (POST /admin/v0/skills) - same as push but different path for UI compatibility
		huma.Register(api, huma.Operation{
			OperationID: "create-skill" + strings.ReplaceAll(pathPrefix, "/", "-"),
			Method:      http.MethodPost,
			Path:        pathPrefix + "/skills",
			Summary:     "Create skill",
			Tags:        tags,
		}, func(ctx context.Context, input *CreateSkillInput) (*Response[SkillResponse], error) {
			return h.createSkill(ctx, input)
		})

		// List all versions of a skill
		huma.Register(api, huma.Operation{
			OperationID: "list-skill-versions" + strings.ReplaceAll(pathPrefix, "/", "-"),
			Method:      http.MethodGet,
			Path:        pathPrefix + "/skills/{skillName}/versions",
			Summary:     "List all versions of a skill",
			Tags:        tags,
		}, func(ctx context.Context, input *SkillDetailInput) (*Response[SkillListResponse], error) {
			return h.listSkillVersions(ctx, input)
		})
	}
}

func (h *SkillHandler) listSkills(ctx context.Context, input *ListSkillsInput, isAdmin bool) (*Response[SkillListResponse], error) {
	var skillList agentregistryv1alpha1.SkillCatalogList

	listOpts := []client.ListOption{}

	if input.Version == "latest" {
		listOpts = append(listOpts, client.MatchingFields{
			controller.IndexSkillIsLatest: "true",
		})
	}

	if err := h.cache.List(ctx, &skillList, listOpts...); err != nil {
		return nil, huma.Error500InternalServerError("Failed to list skills", err)
	}

	skills := make([]SkillResponse, 0, len(skillList.Items))
	for _, s := range skillList.Items {
		if input.Search != "" && !strings.Contains(strings.ToLower(s.Spec.Name), strings.ToLower(input.Search)) {
			continue
		}

		if input.Category != "" && s.Spec.Category != input.Category {
			continue
		}

		if input.Version != "" && input.Version != "latest" && s.Spec.Version != input.Version {
			continue
		}

		skills = append(skills, h.convertToSkillResponse(&s))
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 30
	}
	if len(skills) > limit {
		skills = skills[:limit]
	}

	return &Response[SkillListResponse]{
		Body: SkillListResponse{
			Skills: skills,
			Metadata: ListMetadata{
				Count: len(skills),
			},
		},
	}, nil
}

func (h *SkillHandler) getSkill(ctx context.Context, input *SkillDetailInput, isAdmin bool) (*Response[SkillResponse], error) {
	skillName, err := url.PathUnescape(input.SkillName)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid skill name encoding", err)
	}

	var skillList agentregistryv1alpha1.SkillCatalogList
	listOpts := []client.ListOption{
		client.MatchingFields{
			controller.IndexSkillName:     skillName,
			controller.IndexSkillIsLatest: "true",
		},
	}

	if err := h.cache.List(ctx, &skillList, listOpts...); err != nil {
		return nil, huma.Error500InternalServerError("Failed to get skill", err)
	}

	if len(skillList.Items) == 0 {
		return nil, huma.Error404NotFound("Skill not found")
	}

	return &Response[SkillResponse]{
		Body: h.convertToSkillResponse(&skillList.Items[0]),
	}, nil
}

func (h *SkillHandler) getSkillVersion(ctx context.Context, input *SkillVersionDetailInput, isAdmin bool) (*Response[SkillResponse], error) {
	skillName, err := url.PathUnescape(input.SkillName)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid skill name encoding", err)
	}
	version, err := url.PathUnescape(input.Version)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid version encoding", err)
	}

	var skillList agentregistryv1alpha1.SkillCatalogList
	if err := h.cache.List(ctx, &skillList, client.MatchingFields{
		controller.IndexSkillName: skillName,
	}); err != nil {
		return nil, huma.Error500InternalServerError("Failed to get skill", err)
	}

	for _, s := range skillList.Items {
		if s.Spec.Version == version {
			return &Response[SkillResponse]{
				Body: h.convertToSkillResponse(&s),
			}, nil
		}
	}

	return nil, huma.Error404NotFound("Skill version not found")
}

func (h *SkillHandler) createSkill(ctx context.Context, input *CreateSkillInput) (*Response[SkillResponse], error) {
	crName := GenerateCRName(input.Body.Name, input.Body.Version)

	skill := &agentregistryv1alpha1.SkillCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name: crName,
			Labels: map[string]string{
				"agentregistry.dev/name":    SanitizeK8sName(input.Body.Name),
				"agentregistry.dev/version": SanitizeK8sName(input.Body.Version),
			},
		},
		Spec: agentregistryv1alpha1.SkillCatalogSpec{
			Name:        input.Body.Name,
			Version:     input.Body.Version,
			Title:       input.Body.Title,
			Category:    input.Body.Category,
			Description: input.Body.Description,
			WebsiteURL:  input.Body.WebsiteURL,
		},
	}

	if input.Body.Repository != nil {
		skill.Spec.Repository = &agentregistryv1alpha1.SkillRepository{
			URL:    input.Body.Repository.URL,
			Source: input.Body.Repository.Source,
		}
	}

	for _, p := range input.Body.Packages {
		pkg := agentregistryv1alpha1.SkillPackage{
			RegistryType: p.RegistryType,
			Identifier:   p.Identifier,
			Version:      p.Version,
		}
		if p.Transport != nil {
			pkg.Transport = &agentregistryv1alpha1.SkillPackageTransport{
				Type: p.Transport.Type,
			}
		}
		skill.Spec.Packages = append(skill.Spec.Packages, pkg)
	}

	for _, r := range input.Body.Remotes {
		skill.Spec.Remotes = append(skill.Spec.Remotes, agentregistryv1alpha1.SkillRemote{
			URL: r.URL,
		})
	}

	if err := h.client.Create(ctx, skill); err != nil {
		return nil, huma.Error500InternalServerError("Failed to create skill", err)
	}

	return &Response[SkillResponse]{
		Body: h.convertToSkillResponse(skill),
	}, nil
}

func (h *SkillHandler) listSkillVersions(ctx context.Context, input *SkillDetailInput) (*Response[SkillListResponse], error) {
	skillName, err := url.PathUnescape(input.SkillName)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid skill name encoding", err)
	}

	var skillList agentregistryv1alpha1.SkillCatalogList
	if err := h.cache.List(ctx, &skillList, client.MatchingFields{
		controller.IndexSkillName: skillName,
	}); err != nil {
		return nil, huma.Error500InternalServerError("Failed to list skill versions", err)
	}

	skills := make([]SkillResponse, 0, len(skillList.Items))
	for _, s := range skillList.Items {
		skills = append(skills, h.convertToSkillResponse(&s))
	}

	return &Response[SkillListResponse]{
		Body: SkillListResponse{
			Skills: skills,
			Metadata: ListMetadata{
				Count: len(skills),
			},
		},
	}, nil
}

func (h *SkillHandler) convertToSkillResponse(s *agentregistryv1alpha1.SkillCatalog) SkillResponse {
	skill := SkillJSON{
		Name:        s.Spec.Name,
		Version:     s.Spec.Version,
		Title:       s.Spec.Title,
		Category:    s.Spec.Category,
		Description: s.Spec.Description,
		WebsiteURL:  s.Spec.WebsiteURL,
	}

	if s.Spec.Repository != nil {
		skill.Repository = &SkillRepositoryJSON{
			URL:    s.Spec.Repository.URL,
			Source: s.Spec.Repository.Source,
		}
	}

	for _, p := range s.Spec.Packages {
		pkg := SkillPackageJSON{
			RegistryType: p.RegistryType,
			Identifier:   p.Identifier,
			Version:      p.Version,
		}
		if p.Transport != nil {
			pkg.Transport = &SkillPackageTransportJSON{
				Type: p.Transport.Type,
			}
		}
		skill.Packages = append(skill.Packages, pkg)
	}

	for _, r := range s.Spec.Remotes {
		skill.Remotes = append(skill.Remotes, SkillRemoteJSON{
			URL: r.URL,
		})
	}

	var publishedAt *time.Time
	if s.Status.PublishedAt != nil {
		t := s.Status.PublishedAt.Time
		publishedAt = &t
	}

	// Convert usedBy references
	var usedBy []SkillUsageRefJSON
	for _, ref := range s.Status.UsedBy {
		usedBy = append(usedBy, SkillUsageRefJSON{
			Namespace: ref.Namespace,
			Name:      ref.Name,
			Kind:      ref.Kind,
		})
	}

	resp := SkillResponse{
		Skill: skill,
		Meta: SkillMeta{
			Official: &OfficialMeta{
				Status:      string(s.Status.Status),
				PublishedAt: publishedAt,
				UpdatedAt:   s.CreationTimestamp.Time,
				IsLatest:    s.Status.IsLatest,
				Published:   true,
			},
			UsedBy: usedBy,
		},
	}

	// Include publisher-provided metadata if available
	if s.Spec.Metadata != nil && len(s.Spec.Metadata.Raw) > 0 {
		var metadata map[string]interface{}
		if err := json.Unmarshal(s.Spec.Metadata.Raw, &metadata); err == nil {
			if publisherProvided, ok := metadata["io.modelcontextprotocol.registry/publisher-provided"].(map[string]interface{}); ok {
				resp.Meta.PublisherProvided = publisherProvided
			}
		}
	}

	return resp
}
