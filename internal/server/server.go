package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"mcp-enterprise-registry/internal/db"
)

type Server struct {
	mux    *http.ServeMux
	db     *db.Pool
	logger *log.Logger
}

type Metadata struct {
	NextCursor *string `json:"nextCursor"`
	Count      int     `json:"count"`
}

type ServerResponse struct {
	Server map[string]any `json:"server"`
	Meta   map[string]any `json:"_meta,omitempty"`
}

type ServerListResponse struct {
	Servers  []ServerResponse `json:"servers"`
	Metadata Metadata         `json:"metadata"`
}

func New(ctx context.Context, pool *db.Pool, logger *log.Logger) *Server {
	s := &Server{mux: http.NewServeMux(), db: pool, logger: logger}
	// health and version will be set by caller as needed; register endpoints now
	s.mux.HandleFunc("/v0/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	})
	s.mux.HandleFunc("/v0/version", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"version": "0.1.0"})
	})
	// List servers
	s.mux.HandleFunc("/v0/servers", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v0/servers" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.listServersHandler(w, r)
	})
	// Versions routes
	s.mux.HandleFunc("/v0/servers/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.serversPathRouter(w, r)
	})
	return s
}

func (s *Server) Start(addr string) error {
	s.logger.Printf("registry-api listening on %s", addr)
	return http.ListenAndServe(addr, s.mux)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) listServersHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 30
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > 100 {
				n = 100
			}
			limit = n
		}
	}
	search := q.Get("search")
	updatedSince := q.Get("updated_since")
	version := q.Get("version")

	servers, err := queryListServers(r.Context(), s.db, limit, search, updatedSince, version)
	if err != nil {
		s.logger.Printf("list servers error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to get registry list"})
		return
	}
	resp := ServerListResponse{
		Servers:  servers,
		Metadata: Metadata{NextCursor: nil, Count: len(servers)},
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) serversPathRouter(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/v0/servers/")
	parts := strings.Split(rest, "/")
	if len(parts) >= 2 && parts[1] == "versions" {
		if len(parts) == 2 {
			// versions list
			s.versionsListHandler(w, r, parts[0])
			return
		}
		if len(parts) == 3 {
			// version detail
			s.versionDetailHandler(w, r, parts[0], parts[2])
			return
		}
	}
	http.NotFound(w, r)
}

func (s *Server) versionsListHandler(w http.ResponseWriter, r *http.Request, serverNameEnc string) {
	serverName, err := url.PathUnescape(serverNameEnc)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid server name encoding"})
		return
	}
	servers, err := queryServerVersions(r.Context(), s.db, serverName)
	if err != nil {
		s.logger.Printf("versions list error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to get server versions"})
		return
	}
	resp := ServerListResponse{Servers: servers, Metadata: Metadata{NextCursor: nil, Count: len(servers)}}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) versionDetailHandler(w http.ResponseWriter, r *http.Request, serverNameEnc, versionEnc string) {
	serverName, err := url.PathUnescape(serverNameEnc)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid server name encoding"})
		return
	}
	version, err := url.PathUnescape(versionEnc)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid version encoding"})
		return
	}
	server, err := queryServerVersion(r.Context(), s.db, serverName, version)
	if err != nil {
		s.logger.Printf("version detail error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to get server details"})
		return
	}
	if server == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Server not found"})
		return
	}
	writeJSON(w, http.StatusOK, server)
}

// data access helpers shared with server

func queryListServers(ctx context.Context, pool *db.Pool, limit int, search, updatedSince, version string) ([]ServerResponse, error) {
	if pool == nil || pool.P() == nil {
		return []ServerResponse{}, nil
	}
	var (
		args  []any
		where []string
	)
	if search != "" {
		where = append(where, "server_json->>'name' ILIKE '%'||$"+itoa(len(args)+1)+"||'%'")
		args = append(args, search)
	}
	if updatedSince != "" {
		where = append(where, "last_seen_at >= $"+itoa(len(args)+1))
		args = append(args, updatedSince)
	}
	baseSelect := "server_json, meta_official, meta_enterprise, last_seen_at"
	var sql string
	if version == "latest" {
		cond := ""
		if len(where) > 0 {
			cond = "WHERE " + strings.Join(where, " AND ")
		}
		sql = "SELECT " + baseSelect + " FROM (SELECT DISTINCT ON (server_name) server_name, " + baseSelect + " FROM servers " + cond + " ORDER BY server_name, last_seen_at DESC) t ORDER BY last_seen_at DESC LIMIT $" + itoa(len(args)+1)
		args = append(args, limit)
	} else if version != "" {
		where = append(where, "version = $"+itoa(len(args)+1))
		args = append(args, version)
		cond := ""
		if len(where) > 0 {
			cond = "WHERE " + strings.Join(where, " AND ")
		}
		sql = "SELECT " + baseSelect + " FROM servers " + cond + " ORDER BY last_seen_at DESC LIMIT $" + itoa(len(args)+1)
		args = append(args, limit)
	} else {
		cond := ""
		if len(where) > 0 {
			cond = "WHERE " + strings.Join(where, " AND ")
		}
		sql = "SELECT " + baseSelect + " FROM servers " + cond + " ORDER BY last_seen_at DESC LIMIT $" + itoa(len(args)+1)
		args = append(args, limit)
	}
	rows, err := pool.P().Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ServerResponse
	for rows.Next() {
		var serverJSON, metaOfficialJSON, metaEnterpriseJSON []byte
		var _lastSeen any
		if err := rows.Scan(&serverJSON, &metaOfficialJSON, &metaEnterpriseJSON, &_lastSeen); err != nil {
			return nil, err
		}
		sr, err := buildServerResponse(serverJSON, metaOfficialJSON, metaEnterpriseJSON)
		if err != nil {
			return nil, err
		}
		out = append(out, *sr)
	}
	return out, nil
}

func queryServerVersions(ctx context.Context, pool *db.Pool, serverName string) ([]ServerResponse, error) {
	if pool == nil || pool.P() == nil {
		return []ServerResponse{}, nil
	}
	sql := "SELECT server_json, meta_official, meta_enterprise FROM servers WHERE server_name = $1 ORDER BY last_seen_at DESC"
	rows, err := pool.P().Query(ctx, sql, serverName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ServerResponse
	for rows.Next() {
		var serverJSON, metaOfficialJSON, metaEnterpriseJSON []byte
		if err := rows.Scan(&serverJSON, &metaOfficialJSON, &metaEnterpriseJSON); err != nil {
			return nil, err
		}
		sr, err := buildServerResponse(serverJSON, metaOfficialJSON, metaEnterpriseJSON)
		if err != nil {
			return nil, err
		}
		out = append(out, *sr)
	}
	return out, nil
}

func queryServerVersion(ctx context.Context, pool *db.Pool, serverName, version string) (*ServerResponse, error) {
	if pool == nil || pool.P() == nil {
		return nil, nil
	}
	var sql string
	var args []any
	if version == "latest" {
		sql = "SELECT server_json, meta_official, meta_enterprise FROM servers WHERE server_name = $1 ORDER BY last_seen_at DESC LIMIT 1"
		args = []any{serverName}
	} else {
		sql = "SELECT server_json, meta_official, meta_enterprise FROM servers WHERE server_name = $1 AND version = $2 LIMIT 1"
		args = []any{serverName, version}
	}
	row := pool.P().QueryRow(ctx, sql, args...)
	var serverJSON, metaOfficialJSON, metaEnterpriseJSON []byte
	if err := row.Scan(&serverJSON, &metaOfficialJSON, &metaEnterpriseJSON); err != nil {
		return nil, nil
	}
	sr, err := buildServerResponse(serverJSON, metaOfficialJSON, metaEnterpriseJSON)
	if err != nil {
		return nil, err
	}
	return sr, nil
}

func buildServerResponse(serverJSON, metaOfficialJSON, metaEnterpriseJSON []byte) (*ServerResponse, error) {
	var server map[string]any
	if err := json.Unmarshal(serverJSON, &server); err != nil {
		return nil, err
	}
	meta := map[string]any{}
	if len(metaOfficialJSON) > 0 {
		var m map[string]any
		if err := json.Unmarshal(metaOfficialJSON, &m); err == nil {
			meta["io.modelcontextprotocol.registry/official"] = m
		}
	}
	if len(metaEnterpriseJSON) > 0 {
		var e map[string]any
		if err := json.Unmarshal(metaEnterpriseJSON, &e); err == nil {
			meta["io.modelcontextprotocol.registry/enterprise-mcp-registry"] = e
		}
	}
	return &ServerResponse{Server: server, Meta: meta}, nil
}

func itoa(i int) string { return strconv.Itoa(i) }
