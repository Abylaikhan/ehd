package http

import (
	"time"

	"ehd-api/internal/modules/reporter/domain"
)

// --- запросы ---

// sourceReq — тело создания/обновления источника и теста по параметрам.
// Пароль принимается только на вход; в ответах он отсутствует (REP-FR-013).
type sourceReq struct {
	Code          string `json:"code"`
	Name          string `json:"name"`
	Host          string `json:"host"`
	Port          int    `json:"port"`
	Protocol      string `json:"protocol"`
	TLSEnabled    bool   `json:"tls_enabled"`
	TLSSkipVerify bool   `json:"tls_skip_verify"`
	Username      string `json:"username"`
	Password      string `json:"password"`
}

// --- ответы ---

type sourceResp struct {
	ID            string    `json:"id"`
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	Host          string    `json:"host"`
	Port          int       `json:"port"`
	Protocol      string    `json:"protocol"`
	TLSEnabled    bool      `json:"tls_enabled"`
	TLSSkipVerify bool      `json:"tls_skip_verify"`
	Username      string    `json:"username"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func toSourceResp(ds domain.DataSource) sourceResp {
	return sourceResp{
		ID:            ds.ID,
		Code:          ds.Code,
		Name:          ds.Name,
		Host:          ds.Host,
		Port:          ds.Port,
		Protocol:      ds.Protocol,
		TLSEnabled:    ds.TLSEnabled,
		TLSSkipVerify: ds.TLSSkipVerify,
		Username:      ds.Username,
		Status:        ds.Status,
		CreatedAt:     ds.CreatedAt,
		UpdatedAt:     ds.UpdatedAt,
	}
}

type databaseResp struct {
	Name string `json:"name"`
}

type tableResp struct {
	Name   string `json:"name"`
	Engine string `json:"engine"`
	Kind   string `json:"kind"`
}

type columnResp struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Position     uint64 `json:"position"`
	Nullable     bool   `json:"nullable"`
	Comment      string `json:"comment"`
	InPrimaryKey bool   `json:"in_primary_key"`
	InSortingKey bool   `json:"in_sorting_key"`
}
