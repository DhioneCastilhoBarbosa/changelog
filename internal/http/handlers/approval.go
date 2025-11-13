package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/models"
	"github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/service"
)

// ApprovalHandler lida com homologações de produto.
type ApprovalHandler struct {
	Svc *service.ApprovalService

	// Config do file-server (mesma ideia do ReleaseHandler)
	FilePublicBase string // ex: "https://files.seudominio.com/approvals"
	FileServerBase string // ex: "https://files.seudominio.com/approvals"
	FileServerUser string
	FileServerPass string

	HTTPTimeout time.Duration
}

/*
   DTOs
*/

type CreateApprovalDTO struct {
	Establishment string    `json:"establishment" binding:"required"`
	Date          time.Time `json:"date"` // se vier zero, setamos now no service
	ProductName   string    `json:"productName" binding:"required"`
	Category      string    `json:"category" binding:"required"`
	Description   string    `json:"description" binding:"required"`
	FileURL       string    `json:"fileUrl"` // para JSON puro (sem multipart)
}

type ApprovalResponse struct {
	ID            uint      `json:"id"`
	Establishment string    `json:"establishment"`
	Date          time.Time `json:"date"`
	ProductName   string    `json:"productName"`
	Category      string    `json:"category"`
	Description   string    `json:"description"`
	FileURL       string    `json:"fileUrl"`
	CreatedBy     *UserPublic `json:"createdBy,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

/*
   Mapeadores
*/

func toApprovalResponse(m *models.Approval) ApprovalResponse {
	return ApprovalResponse{
		ID:            m.ID,
		Establishment: m.Establishment,
		Date:          m.Date,
		ProductName:   m.ProductName,
		Category:      m.Category,
		Description:   m.Description,
		FileURL:       m.FileURL,
		CreatedBy:     toPublicUser(m.CreatedBy),
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}

/*
   Helpers de upload (versão simplificada reaproveitando tua lógica)
*/

// monta destino DAV: <FileServerBase>/<filename>
func (h ApprovalHandler) davDest(filename string) (string, error) {
	if h.FileServerBase == "" {
		return "", fmt.Errorf("file-server não configurado")
	}
	base := strings.TrimRight(h.FileServerBase, "/") + "/"
	return base + url.PathEscape(filepath.Base(filename)), nil
}





func (h ApprovalHandler) davPut(ctx context.Context, filename string, r io.Reader) (publicURL string, size int64, err error) {
	dest, err := h.davDest(filename)
	if err != nil {
		return "", 0, err
	}

	mt := mime.TypeByExtension(strings.ToLower(filepath.Ext(filename)))
	if mt == "" {
		mt = "application/octet-stream"
	}

	cr := &countReader{R: r}
	body := io.TeeReader(cr, io.Discard) // se quiser hash depois, adapta

	to := h.HTTPTimeout
	if to == 0 {
		to = 120 * time.Second
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, dest, body)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Content-Type", mt)
	if h.FileServerUser != "" {
		req.SetBasicAuth(h.FileServerUser, h.FileServerPass)
	}

	resp, err := (&http.Client{Timeout: to}).Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", 0, fmt.Errorf("file-server %d: %s", resp.StatusCode, string(b))
	}

	// URL pública final
	pub := strings.TrimRight(h.FilePublicBase, "/") + "/"
	pub += url.PathEscape(filepath.Base(filename))

	return pub, cr.N, nil
}



/*
   Handlers
*/

// POST /approvals
func (h ApprovalHandler) Create(c *gin.Context) {
	ct := c.ContentType()

	// Pega userID do contexto (mesmo padrão do Release)
	uidVal, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}
	userID, ok := uidVal.(uint)
	if !ok || userID == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user id"})
		return
	}

	// 1) JSON puro
	if strings.HasPrefix(ct, "application/json") {
		var in CreateApprovalDTO
		if err := c.ShouldBindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Valida URL se vier preenchida
		if strings.TrimSpace(in.FileURL) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "fileUrl é obrigatório em JSON"})
			return
		}
		if _, err := url.ParseRequestURI(in.FileURL); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "fileUrl inválido"})
			return
		}

		a := &models.Approval{
			Establishment:  in.Establishment,
			Date:           in.Date,
			ProductName:    in.ProductName,
			Category:       in.Category,
			Description:    in.Description,
			FileURL:        in.FileURL,
			CreatedByUserID: userID,
		}

		out, err := h.Svc.Create(a)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, toApprovalResponse(out))
		return
	}

	// 2) multipart/form-data (campos + arquivo)
	if strings.HasPrefix(ct, "multipart/form-data") {
		var in CreateApprovalDTO

		// Você pode mandar um JSON no campo "data" ou mandar tudo plano.
		raw := c.PostForm("data")
		if raw != "" {
			if err := json.Unmarshal([]byte(raw), &in); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "JSON inválido em 'data': " + err.Error()})
				return
			}
		} else {
			// Campos simples
			in.Establishment = c.PostForm("establishment")
			in.ProductName = c.PostForm("productName")
			in.Category = c.PostForm("category")
			in.Description = c.PostForm("description")

			if v := c.PostForm("date"); v != "" {
				// espera "2006-01-02" ou ISO; adapta conforme teu front
				if t, err := time.Parse("2006-01-02", v); err == nil {
					in.Date = t
				}
			}
		}

		if strings.TrimSpace(in.Establishment) == "" ||
			strings.TrimSpace(in.ProductName) == "" ||
			strings.TrimSpace(in.Category) == "" ||
			strings.TrimSpace(in.Description) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "campos obrigatórios: establishment, productName, category, description"})
			return
		}

		// Arquivo obrigatório em multipart
		fh, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "arquivo 'file' é obrigatório"})
			return
		}
		filename := filepath.Base(fh.Filename)
		f, oerr := fh.Open()
		if oerr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "falha ao abrir arquivo"})
			return
		}
		publicURL, _, perr := h.davPut(c.Request.Context(), filename, f)
		_ = f.Close()
		if perr != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "upload falhou: " + perr.Error()})
			return
		}

		a := &models.Approval{
			Establishment:  in.Establishment,
			Date:           in.Date,
			ProductName:    in.ProductName,
			Category:       in.Category,
			Description:    in.Description,
			FileURL:        publicURL,
			CreatedByUserID: userID,
		}

		out, err := h.Svc.Create(a)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, toApprovalResponse(out))
		return
	}

	c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "use application/json ou multipart/form-data"})
}

// GET /approvals/:id
func (h ApprovalHandler) Get(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	out, err := h.Svc.Get(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "approval não encontrado"})
		return
	}
	c.JSON(http.StatusOK, toApprovalResponse(out))
}

// GET /approvals
func (h ApprovalHandler) List(c *gin.Context) {
	var (
		q             = c.Query("q")
		establishment = c.Query("establishment")
		productName   = c.Query("productName")
		category      = c.Query("category")
		df, dt        *time.Time
	)
	if v := c.Query("date_from"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			df = &t
		}
	}
	if v := c.Query("date_to"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			dt = &t
		}
	}

	list, err := h.Svc.List(service.ApprovalQuery{
		Q:             q,
		Establishment: establishment,
		ProductName:   productName,
		Category:      category,
		DateFrom:      df,
		DateTo:        dt,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := make([]ApprovalResponse, 0, len(list))
	for _, it := range list {
		resp = append(resp, toApprovalResponse(&it))
	}
	c.JSON(http.StatusOK, resp)
}

// DELETE /approvals/:id
func (h ApprovalHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	// Primeiro busca para tentar remover o arquivo remoto
	approval, _ := h.Svc.Get(uint(id))
	if approval != nil {
		u := strings.TrimSpace(approval.FileURL)
		basePub := strings.TrimRight(h.FilePublicBase, "/") + "/"
		if u != "" && strings.HasPrefix(u, basePub) {
			// Apaga no file-server
			// Aproveita a mesma lógica de Delete do Release (poderia extrair pra helper compartilhado)
			req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodDelete, u, nil)
			if err == nil {
				if h.FileServerUser != "" {
					req.SetBasicAuth(h.FileServerUser, h.FileServerPass)
				}
				resp, err := (&http.Client{Timeout: h.HTTPTimeout}).Do(req)
				if err == nil {
					_ = resp.Body.Close()
				}
			}
		}
	}

	if err := h.Svc.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
