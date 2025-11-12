package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/models"
	"github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/service"
)

/*
=========================

	DTOs de entrada
	=========================
*/

type ReleaseHandler struct {
	Svc *service.ReleaseService

	// URLs públicas (para devolver ao cliente)
	FilePublicBase string // ex: "https://files.seudominio.com/firmware"

	// WebDAV (Nginx do servidor de arquivos)
	FileServerBase string // ex: "https://files.seudominio.com/firmware"  (sem barra final)
	FileServerUser string // ex: "uploader"
	FileServerPass string // ex: "vmx30032"

	// Local file system root for firmware uploads
	FileLocalRoot string // ex: "/var/www/files/firmware"

	HTTPTimeout time.Duration
}


type FirmwareLinkDTO struct {
	Module      string `json:"module" binding:"required"`
	Description string `json:"description" binding:"required"`
	URL         string `json:"url" binding:"required"`
}

type ModuleDTO struct {
	Module  string `json:"module"`
	Version string `json:"version"`
	Updated bool   `json:"updated"`
}
type EntryDTO struct {
	ItemOrder      int    `json:"itemOrder"`
	Classification string `json:"classification"`
	Observation    string `json:"observation"`
}
type CreateReleaseDTO struct {
	Version         string      `json:"version"`
	PreviousVersion string      `json:"previousVersion"`
	OTA             bool        `json:"ota"`
	OTAObs          string      `json:"otaObs"`
	ReleaseDate     time.Time   `json:"releaseDate"`
	ImportantNote   string      `json:"importantNote"`
	Status          string      `json:"status"` // <- NOVO: "revisao" | "producao" | "descontinuado"
	Modules         []ModuleDTO `json:"modules"`
	Entries         []EntryDTO  `json:"entries"`
	Links           []FirmwareLinkDTO `json:"links"` // <- NOVO
	ProductCategory string      `json:"productCategory"`
	ProductName     string      `json:"productName"`
}

type deleteFileDTO struct{ URL string `json:"url"`; Path string `json:"path"` }

/*
=========================

	DTOs de saída (seguros)
	=========================
*/

type ReleaseLinkPublic struct {
	ID          uint   `json:"id"`
	Module      string `json:"module"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

type UserPublic struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

type ReleaseModulePublic struct {
	ID      uint   `json:"id"`
	Module  string `json:"module"`
	Version string `json:"version"`
	Updated bool   `json:"updated"`
}

type ChangelogEntryPublic struct {
	ID             uint   `json:"id"`
	ItemOrder      int    `json:"itemOrder"`
	Classification string `json:"classification"`
	Observation    string `json:"observation"`
}

type ReleaseResponse struct {
	ID              uint                   `json:"id"`
	Version         string                 `json:"version"`
	PreviousVersion string                 `json:"previousVersion"`
	OTA             bool                   `json:"ota"`
	OTAObs          string                 `json:"otaObs,omitempty"`
	ReleaseDate     time.Time              `json:"releaseDate"`
	ImportantNote   string                 `json:"importantNote,omitempty"`
	ProductCategory string                 `json:"productCategory"`
	ProductName     string                 `json:"productName"`
	Status          string                 `json:"status"`
	CreatedBy       *UserPublic            `json:"createdBy,omitempty"`
	Modules         []ReleaseModulePublic  `json:"modules,omitempty"`
	Entries         []ChangelogEntryPublic `json:"entries,omitempty"`
	Links          []ReleaseLinkPublic     `json:"links,omitempty"` // <- NOVO
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
}

/*
=========================

	Mapeadores
	=========================
*/

func toModelLinks(ls []FirmwareLinkDTO) []models.FirmwareLink {
	out := make([]models.FirmwareLink, 0, len(ls))
	for _, l := range ls {
		out = append(out, models.FirmwareLink{
			Module:      l.Module,
			Description: l.Description,
			URL:         l.URL,
		})
	}
	return out
}

func toPublicLinks(ls []models.FirmwareLink) []ReleaseLinkPublic {
	out := make([]ReleaseLinkPublic, 0, len(ls))
	for _, l := range ls {
		out = append(out, ReleaseLinkPublic{
			ID: l.ID, Module: l.Module, Description: l.Description, URL: l.URL,
		})
	}
	return out
}

func toModelModules(ms []ModuleDTO) []models.ReleaseModule {
	out := make([]models.ReleaseModule, 0, len(ms))
	for _, m := range ms {
		out = append(out, models.ReleaseModule{
			Module:  m.Module,
			Version: m.Version,
			Updated: m.Updated,
		})
	}
	return out
}
func toModelEntries(es []EntryDTO) []models.ChangelogEntry {
	out := make([]models.ChangelogEntry, 0, len(es))
	for _, e := range es {
		out = append(out, models.ChangelogEntry{
			ItemOrder:      e.ItemOrder,
			Classification: models.EntryClassification(e.Classification),
			Observation:    e.Observation,
		})
	}
	return out
}

func toPublicUser(u *models.User) *UserPublic {
	if u == nil || u.ID == 0 {
		return nil
	}
	return &UserPublic{ID: u.ID, Name: u.Name, Role: string(u.Role)}
}

func toPublicModules(ms []models.ReleaseModule) []ReleaseModulePublic {
	out := make([]ReleaseModulePublic, 0, len(ms))
	for _, m := range ms {
		out = append(out, ReleaseModulePublic{
			ID: m.ID, Module: m.Module, Version: m.Version, Updated: m.Updated,
		})
	}
	return out
}

func toPublicEntries(es []models.ChangelogEntry) []ChangelogEntryPublic {
	out := make([]ChangelogEntryPublic, 0, len(es))
	for _, e := range es {
		out = append(out, ChangelogEntryPublic{
			ID: e.ID, ItemOrder: e.ItemOrder,
			Classification: string(e.Classification),
			Observation:    e.Observation,
		})
	}
	return out
}

func toReleaseResponse(m *models.Release) ReleaseResponse {
	return ReleaseResponse{
		ID: m.ID, Version: m.Version, PreviousVersion: m.PreviousVersion,
		OTA: m.OTA, OTAObs: m.OTAObs, ReleaseDate: m.ReleaseDate,
		ImportantNote:   m.ImportantNote,
		ProductCategory: m.ProductCategory,
		ProductName:     m.ProductName,
		Status:          string(m.Status), // <- NOVO
		CreatedBy:       toPublicUser(m.CreatedBy),
		Modules:         toPublicModules(m.Modules),
		Entries:         toPublicEntries(m.Entries),
		Links:           toPublicLinks(m.Links), // <- NOVO
		CreatedAt:       m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

/* ===== Helpers de upload ===== */


// helper: carrega 1..N arquivos do multipart e devolve []FirmwareLink
func (h ReleaseHandler) collectUploadedLinks(c *gin.Context) ([]models.FirmwareLink, error) {
    form, err := c.MultipartForm()
    if err != nil || form == nil {
        // fallback: aceitar um único arquivo no campo "file"
        fh, ferr := c.FormFile("file")
        if ferr != nil {
            return nil, nil // sem arquivos
        }
        f, oerr := fh.Open()
        if oerr != nil { return nil, fmt.Errorf("falha ao abrir arquivo: %w", oerr) }
        defer f.Close()

        dir := strings.TrimSpace(c.PostForm("dir"))
        publicURL, _, perr := h.davPut(c.Request.Context(), filepath.Base(fh.Filename), dir, f)
        if perr != nil { return nil, fmt.Errorf("upload falhou: %w", perr) }

        m := strings.TrimSpace(c.PostForm("linkModule"))
        if m == "" { m = "default" }
        d := strings.TrimSpace(c.PostForm("linkDescription"))
        if d == "" { d = "Firmware" }

        return []models.FirmwareLink{{Module: m, Description: d, URL: publicURL}}, nil
    }

    // múltiplos arquivos: files[]
    files := form.File["files[]"]
    if len(files) == 0 {
        return nil, nil
    }
    if len(files) > 20 {
        return nil, fmt.Errorf("máximo de 20 arquivos em files[]")
    }

    dir  := strings.TrimSpace(c.PostForm("dir"))
    mods := form.Value["linkModule[]"]
    desc := form.Value["linkDescription[]"]

    out := make([]models.FirmwareLink, 0, len(files))
    for i, fh := range files {
        if fh == nil { continue }
        f, err := fh.Open()
        if err != nil { return nil, fmt.Errorf("falha ao abrir arquivo: %w", err) }

        // opcional: nome seguro
        filename := filepath.Base(fh.Filename)

        publicURL, _, upErr := h.davPut(c.Request.Context(), filename, dir, f)
        _ = f.Close()
        if upErr != nil { return nil, fmt.Errorf("upload falhou: %w", upErr) }

        m := "default"
        if i < len(mods) && strings.TrimSpace(mods[i]) != "" { m = strings.TrimSpace(mods[i]) }
        d := "Firmware"
        if i < len(desc) && strings.TrimSpace(desc[i]) != "" { d = strings.TrimSpace(desc[i]) }

        out = append(out, models.FirmwareLink{Module: m, Description: d, URL: publicURL})
    }
    return out, nil
}


// monta destino DAV: <FileServerBase>/<dir>/<filename>
func (h ReleaseHandler) davDest(dir, filename string) (string, error) {
    if h.FileServerBase == "" { return "", fmt.Errorf("file-server não configurado") }
    base := strings.TrimRight(h.FileServerBase, "/") + "/"
    if dir != "" {
        s, err := sanitizeRel(dir)
        if err != nil { return "", err }
        if s != "" { base += url.PathEscape(s) + "/" }
    }
    return base + url.PathEscape(filepath.Base(filename)), nil
}

// PUT para o servidor de arquivos via WebDAV
func (h ReleaseHandler) davPut(ctx context.Context, filename, dir string, r io.Reader) (publicURL string, size int64, err error) {
    dest, err := h.davDest(dir, filename)
    if err != nil { return "", 0, err }

    mt := mime.TypeByExtension(strings.ToLower(filepath.Ext(filename)))
    if mt == "" { mt = "application/octet-stream" }

    hh := sha256.New()
    cr := &countReader{R: r}
    body := io.TeeReader(cr, hh)

    to := h.HTTPTimeout
    if to == 0 { to = 120 * time.Second }

    req, err := http.NewRequestWithContext(ctx, http.MethodPut, dest, body)
    if err != nil { return "", 0, err }
    req.Header.Set("Content-Type", mt)
    if h.FileServerUser != "" {
        req.SetBasicAuth(h.FileServerUser, h.FileServerPass)
    }

    resp, err := (&http.Client{Timeout: to}).Do(req)
    if err != nil { return "", 0, err }
    defer resp.Body.Close()
    if resp.StatusCode >= 300 {
        b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
        return "", 0, fmt.Errorf("file-server %d: %s", resp.StatusCode, string(b))
    }

    // URL pública final
    pub := strings.TrimRight(h.FilePublicBase, "/") + "/"
    if dir != "" {
        s, _ := sanitizeRel(dir)
        if s != "" { pub += url.PathEscape(s) + "/" }
    }
    pub += url.PathEscape(filepath.Base(filename))
    _ = hex.EncodeToString(hh.Sum(nil)) // disponível p/ log
    return pub, cr.N, nil
}

// DELETE no servidor de arquivos. Aceita URL pública completa OU caminho "AC/arquivo.bin".
func (h ReleaseHandler) davDelete(ctx context.Context, publicOrPath string) error {
    to := h.HTTPTimeout
    if to == 0 { to = 120 * time.Second }

    target := publicOrPath
    if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
        if h.FileServerBase == "" { return fmt.Errorf("file-server não configurado") }
        base := strings.TrimRight(h.FileServerBase, "/")
        p := strings.ReplaceAll(publicOrPath, "\\", "/")
        p2, err := sanitizeRel(p)
        if err != nil { return err }
        // OBS: p2 já é “AC/arquivo.bin”; não re-escape diretórios separadamente aqui
        target = base + "/" + strings.ReplaceAll(url.PathEscape(p2), "%2F", "/")
    }

    req, err := http.NewRequestWithContext(ctx, http.MethodDelete, target, nil)
    if err != nil { return err }
    if h.FileServerUser != "" {
        req.SetBasicAuth(h.FileServerUser, h.FileServerPass)
    }
    resp, err := (&http.Client{Timeout: to}).Do(req)
    if err != nil { return err }
    defer resp.Body.Close()
    if resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotFound {
        b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
        return fmt.Errorf("file-server %d: %s", resp.StatusCode, string(b))
    }
    return nil
}


func sanitizeRel(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" { return "", nil }
	p = strings.ReplaceAll(p, "\\", "/")
	p = path.Clean("/" + p)
	if strings.Contains(p, "..") {
		return "", fmt.Errorf("caminho inválido")
	}
	return strings.Trim(p, "/"), nil
}

type countReader struct{ R io.Reader; N int64 }
func (c *countReader) Read(p []byte) (int, error) {
	n, err := c.R.Read(p); c.N += int64(n); return n, err
}

// constrói URL pública com base + dir + filename
func (h ReleaseHandler) makePublicURL(dir, filename string) string {
    base := strings.TrimRight(h.FilePublicBase, "/") + "/"
    if dir != "" {
        return base + url.PathEscape(dir) + "/" + url.PathEscape(filename)
    }
    return base + url.PathEscape(filename)
}

// salva no FS local: <FileLocalRoot>/firmware[/dir]/filename
func (h ReleaseHandler) saveToLocal(filename, dir string, r io.Reader) (publicURL string, size int64, err error) {
	if h.FileLocalRoot == "" { return "", 0, fmt.Errorf("FileLocalRoot não configurado") }

	root := strings.TrimRight(h.FileLocalRoot, "/")

	if dir != "" {
		if dir, err = sanitizeRel(dir); err != nil { return "", 0, err }
	}

	destDir := root
	if dir != "" {
		destDir = filepath.Join(destDir, filepath.FromSlash(dir))
	}
	if err := os.MkdirAll(destDir, 0o775); err != nil {
		return "", 0, fmt.Errorf("falha ao criar diretório: %w", err)
	}

	destPath := filepath.Join(destDir, filepath.Base(filename))
	f, err := os.Create(destPath)
	if err != nil { return "", 0, fmt.Errorf("falha ao criar arquivo: %w", err) }
	defer func() { _ = f.Sync(); _ = f.Close() }()

	n, err := io.Copy(f, r)
	if err != nil { return "", 0, fmt.Errorf("falha ao gravar arquivo: %w", err) }
	if n == 0 { return "", 0, fmt.Errorf("arquivo vazio: %s", destPath) }

	return h.makePublicURL(dir, filepath.Base(filename)), n, nil
}




/* =========================
   Handlers
   ========================= */

func (h ReleaseHandler) Create(c *gin.Context) {
	ct := c.ContentType()

	// === JSON puro ===
	if strings.HasPrefix(ct, "application/json") {
		var in CreateReleaseDTO
		if err := c.ShouldBindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		for i, l := range in.Links {
			if _, err := url.ParseRequestURI(l.URL); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "link inválido na posição " + strconv.Itoa(i)})
				return
			}
		}

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

		st := models.FirmwareStatus(in.Status)
		if st == "" {
			st = models.FirmwareStatusProducao
		}
		if !st.Valid() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "status inválido: use revisao|producao|descontinuado"})
			return
		}

		rel := &models.Release{
			Version:         in.Version,
			PreviousVersion: in.PreviousVersion,
			OTA:             in.OTA,
			OTAObs:          in.OTAObs,
			ReleaseDate:     in.ReleaseDate,
			ImportantNote:   in.ImportantNote,
			ProductCategory: in.ProductCategory,
			ProductName:     in.ProductName,
			Status:          st,
			Modules:         toModelModules(in.Modules),
			Entries:         toModelEntries(in.Entries),
			Links:           toModelLinks(in.Links),
			CreatedByUserID: userID,
		}
		out, err := h.Svc.Create(rel)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, toReleaseResponse(out))
		return
	}

	// === multipart/form-data ===
	if strings.HasPrefix(ct, "multipart/form-data") {
		var in CreateReleaseDTO

		raw := c.PostForm("data")
		if raw == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "campo 'data' (JSON) obrigatório no multipart"})
			return
		}
		if err := json.Unmarshal([]byte(raw), &in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "JSON inválido em 'data': " + err.Error()})
			return
		}

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

		st := models.FirmwareStatus(in.Status)
		if st == "" {
			st = models.FirmwareStatusProducao
		}
		if !st.Valid() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "status inválido: use revisao|producao|descontinuado"})
			return
		}

		// valida links já presentes no JSON
		for i, l := range in.Links {
			if _, err := url.ParseRequestURI(l.URL); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "link inválido na posição " + strconv.Itoa(i)})
				return
			}
		}
		links := toModelLinks(in.Links)

		// tenta obter o formulário completo
		form, _ := c.MultipartForm()
		dir := strings.TrimSpace(c.PostForm("dir"))

		// caminho A: múltiplos arquivos via files[]
		if form != nil && len(form.File["files[]"]) > 0 {
			files := form.File["files[]"]
			mods := form.Value["linkModule[]"]
			descs := form.Value["linkDescription[]"]

			const maxFiles = 20
			if len(files) > maxFiles {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("máximo de %d arquivos em files[]", maxFiles)})
				return
			}

			for i, fh := range files {
				if fh == nil {
					continue
				}
				// opcional: validar tamanho antes de abrir
				// if fh.Size > (100<<20) { ... } // 100 MB

				filename := filepath.Base(fh.Filename)
				f, err := fh.Open()
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "falha ao abrir arquivo"})
					return
				}
				publicURL, _, err := h.davPut(c.Request.Context(), filename, dir, f)
				_ = f.Close()
				if err != nil {
					c.JSON(http.StatusBadGateway, gin.H{"error": "upload falhou: " + err.Error()})
					return
				}

				m := "default"
				if i < len(mods) && strings.TrimSpace(mods[i]) != "" {
					m = strings.TrimSpace(mods[i])
				}
				d := "Firmware"
				if i < len(descs) && strings.TrimSpace(descs[i]) != "" {
					d = strings.TrimSpace(descs[i])
				}

				links = append(links, models.FirmwareLink{
					Module:      m,
					Description: d,
					URL:         publicURL,
				})
			}
		} else {
			// caminho B: único arquivo via file (retrocompatível)
			fh, err := c.FormFile("file")
			if err == nil && fh != nil {
				filename := filepath.Base(fh.Filename)
				f, oerr := fh.Open()
				if oerr != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "falha ao abrir arquivo"})
					return
				}
				publicURL, _, perr := h.davPut(c.Request.Context(), filename, dir, f)
				_ = f.Close()
				if perr != nil {
					c.JSON(http.StatusBadGateway, gin.H{"error": "upload falhou: " + perr.Error()})
					return
				}

				m := strings.TrimSpace(c.PostForm("linkModule"))
				if m == "" {
					m = "default"
				}
				d := strings.TrimSpace(c.PostForm("linkDescription"))
				if d == "" {
					d = "Firmware"
				}

				links = append(links, models.FirmwareLink{
					Module:      m,
					Description: d,
					URL:         publicURL,
				})
			}
		}

		rel := &models.Release{
			Version:         in.Version,
			PreviousVersion: in.PreviousVersion,
			OTA:             in.OTA,
			OTAObs:          in.OTAObs,
			ReleaseDate:     in.ReleaseDate,
			ImportantNote:   in.ImportantNote,
			ProductCategory: in.ProductCategory,
			ProductName:     in.ProductName,
			Status:          st,
			Modules:         toModelModules(in.Modules),
			Entries:         toModelEntries(in.Entries),
			Links:           links,
			CreatedByUserID: userID,
		}

		out, err := h.Svc.Create(rel)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, toReleaseResponse(out))
		return
	}

	c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "use application/json ou multipart/form-data"})
}



func (h ReleaseHandler) Get(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	out, err := h.Svc.Get(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "release não encontrado"})
		return
	}
	c.JSON(http.StatusOK, toReleaseResponse(out))
}

func (h ReleaseHandler) List(c *gin.Context) {
	var (
		q       = c.Query("q")
		version = c.Query("version")
		df, dt  *time.Time
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
	list, err := h.Svc.List(service.ReleaseQuery{Q: q, Version: version, DateFrom: df, DateTo: dt})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := make([]ReleaseResponse, 0, len(list))
	for _, it := range list {
		resp = append(resp, toReleaseResponse(&it))
	}
	c.JSON(http.StatusOK, resp)
}

func (h ReleaseHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	cur, err := h.Svc.Get(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "release não encontrado"})
		return
	}

	ct := c.ContentType()

	// =======================
	// 1) application/json
	// =======================
	if strings.HasPrefix(ct, "application/json") {
		var in CreateReleaseDTO
		if err := c.ShouldBindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		for i, l := range in.Links {
			if _, err := url.ParseRequestURI(l.URL); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "link inválido na posição " + strconv.Itoa(i)})
				return
			}
		}

		st := models.FirmwareStatus(in.Status)
		if st == "" { st = models.FirmwareStatusProducao }
		if !st.Valid() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "status inválido: use revisao|producao|descontinuado"})
			return
		}

		base := models.Release{
			ID:              cur.ID,
			Version:         in.Version,
			PreviousVersion: in.PreviousVersion,
			OTA:             in.OTA,
			OTAObs:          in.OTAObs,
			ReleaseDate:     in.ReleaseDate,
			ImportantNote:   in.ImportantNote,
			Status:          st,
			ProductCategory: in.ProductCategory,
			ProductName:     in.ProductName,
			CreatedByUserID: cur.CreatedByUserID,
			CreatedAt:       cur.CreatedAt,
		}

		out, err := h.Svc.UpdateFull(
			uint(id),
			base,
			toModelModules(in.Modules),
			toModelEntries(in.Entries),
			toModelLinks(in.Links),
		)
		if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
		c.JSON(http.StatusOK, toReleaseResponse(out))
		return
	}

	// =======================
	// 2) multipart/form-data
	// =======================
	if strings.HasPrefix(ct, "multipart/form-data") {
		var in CreateReleaseDTO

		raw := c.PostForm("data")
		if raw == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "campo 'data' (JSON) obrigatório no multipart"})
			return
		}
		if err := json.Unmarshal([]byte(raw), &in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "JSON inválido em 'data': " + err.Error()})
			return
		}

		// valida links vindos no JSON
		for i, l := range in.Links {
			if _, err := url.ParseRequestURI(l.URL); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "link inválido na posição " + strconv.Itoa(i)})
				return
			}
		}
		links := toModelLinks(in.Links)

		st := models.FirmwareStatus(in.Status)
		if st == "" { st = models.FirmwareStatusProducao }
		if !st.Valid() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "status inválido: use revisao|producao|descontinuado"})
			return
		}

		// uploads opcionais
		form, _ := c.MultipartForm()
		dir := strings.TrimSpace(c.PostForm("dir"))

		// junta múltiplos "files[]" e também múltiplos "file" (tolerante ao Postman)
		var files []*multipart.FileHeader
		if form != nil {
			files = append(files, form.File["files[]"]...)
			files = append(files, form.File["file"]...)
		}

		// módulos/descrições: aceita [] e repetidos simples
		mods := []string{}
		descs := []string{}
		if form != nil {
			mods = form.Value["linkModule[]"]
			if len(mods) == 0 { mods = form.Value["linkModule"] }
			descs = form.Value["linkDescription[]"]
			if len(descs) == 0 { descs = form.Value["linkDescription"] }
		}

		if len(files) > 0 {
			const maxFiles = 20
			if len(files) > maxFiles {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("máximo de %d arquivos", maxFiles)})
				return
			}

			for i, fh := range files {
				if fh == nil { continue }
				// opcional: validar tamanho (ex.: 100 MB)
				// if fh.Size > (100<<20) { c.JSON(413, gin.H{"error":"arquivo grande"}); return }

				filename := filepath.Base(fh.Filename)
				f, err := fh.Open()
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "falha ao abrir arquivo"})
					return
				}
				publicURL, _, err := h.davPut(c.Request.Context(), filename, dir, f)
				_ = f.Close()
				if err != nil {
					c.JSON(http.StatusBadGateway, gin.H{"error": "upload falhou: " + err.Error()})
					return
				}

				m := "default"
				if i < len(mods) && strings.TrimSpace(mods[i]) != "" { m = strings.TrimSpace(mods[i]) }
				d := "Firmware"
				if i < len(descs) && strings.TrimSpace(descs[i]) != "" { d = strings.TrimSpace(descs[i]) }

				links = append(links, models.FirmwareLink{
					Module:      m,
					Description: d,
					URL:         publicURL,
				})
			}
		}

		base := models.Release{
			ID:              cur.ID,
			Version:         in.Version,
			PreviousVersion: in.PreviousVersion,
			OTA:             in.OTA,
			OTAObs:          in.OTAObs,
			ReleaseDate:     in.ReleaseDate,
			ImportantNote:   in.ImportantNote,
			Status:          st,
			ProductCategory: in.ProductCategory,
			ProductName:     in.ProductName,
			CreatedByUserID: cur.CreatedByUserID,
			CreatedAt:       cur.CreatedAt,
		}

		// IMPORTANTE: UpdateFull substitui módulos/entries/links.
		// Portanto: os links finais = (JSON in.Links) + (uploads acima).
		out, err := h.Svc.UpdateFull(
			uint(id),
			base,
			toModelModules(in.Modules),
			toModelEntries(in.Entries),
			links,
		)
		if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
		c.JSON(http.StatusOK, toReleaseResponse(out))
		return
	}

	c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "use application/json ou multipart/form-data"})
}



func (h ReleaseHandler) Delete(c *gin.Context) {
    id, _ := strconv.Atoi(c.Param("id"))

    // 4.1 buscar release para obter os links (se existir)
    rel, _ := h.Svc.Get(uint(id)) // se errar aqui, seguimos com a deleção do banco

    // 4.2 tentar apagar os arquivos remotos ligados a este release
    if rel != nil && len(rel.Links) > 0 {
        for _, lk := range rel.Links {
            u := strings.TrimSpace(lk.URL)
            if u == "" { continue }

            // regra: só apagar o que estiver sob a sua base pública (evita deletar URLs de terceiros)
            basePub := strings.TrimRight(h.FilePublicBase, "/") + "/"
            if strings.HasPrefix(u, basePub) {
                // Delete pela URL pública direta
                _ = h.davDelete(c.Request.Context(), u)
            }
        }
    }

    // 4.3 deletar o release no serviço
    if err := h.Svc.Delete(uint(id)); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.Status(http.StatusNoContent)
}



func (h ReleaseHandler) DeleteFile(c *gin.Context) {
    var in deleteFileDTO
    if err := c.ShouldBindJSON(&in); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
    }
    v := strings.TrimSpace(in.URL)
    if v == "" { v = strings.TrimSpace(in.Path) }
    if v == "" { c.JSON(http.StatusBadRequest, gin.H{"error": "informe url ou path"}); return }

    if err := h.davDelete(c.Request.Context(), v); err != nil {
        c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()}); return
    }
    c.Status(http.StatusNoContent)
}

