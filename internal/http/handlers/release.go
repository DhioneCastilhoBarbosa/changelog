package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/models"
	"github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/service"
)

type ReleaseHandler struct {
	Svc *service.ReleaseService
}

/*
=========================

	DTOs de entrada (como já tinha)
	=========================
*/
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
	ProductCategory string      `json:"productCategory"`
	ProductName     string      `json:"productName"`
}

/*
=========================

	DTOs de saída (seguros)
	=========================
*/
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
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
}

/*
=========================

	Mapeadores
	=========================
*/
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
		CreatedAt:       m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

/* =========================
   Handlers
   ========================= */

func (h ReleaseHandler) Create(c *gin.Context) {
	var in CreateReleaseDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	// validar status
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
		ProductCategory: in.ProductCategory, // novo
		ProductName:     in.ProductName,     // novo
		Status:          st,                 // <- NOVO
		Modules:         toModelModules(in.Modules),
		Entries:         toModelEntries(in.Entries),
		CreatedByUserID: userID,
	}

	out, err := h.Svc.Create(rel)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, toReleaseResponse(out))
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

	// 1) Carrega o atual para preservar campos imutáveis
	cur, err := h.Svc.Get(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "release não encontrado"})
		return
	}

	// 2) Bind do payload (sem timestamps)
	var in CreateReleaseDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 3) Valida status
	st := models.FirmwareStatus(in.Status)
	if st == "" {
		st = models.FirmwareStatusProducao
	}
	if !st.Valid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status inválido: use revisao|producao|descontinuado"})
		return
	}

	// 4) Monta base preservando quem não deve mudar
	base := models.Release{
		ID:               cur.ID,              // garante PK correta
		Version:          in.Version,
		PreviousVersion:  in.PreviousVersion,
		OTA:              in.OTA,
		OTAObs:           in.OTAObs,
		ReleaseDate:      in.ReleaseDate,
		ImportantNote:    in.ImportantNote,
		Status:           st,
		ProductCategory:  in.ProductCategory,
		ProductName:      in.ProductName,

		CreatedByUserID:  cur.CreatedByUserID, // PRESERVA criador
		CreatedAt:        cur.CreatedAt,       // PRESERVA CreatedAt
		// UpdatedAt: deixe o GORM atualizar sozinho
	}

	// 5) Atualiza
	out, err := h.Svc.UpdateFull(
		uint(id),
		base,
		toModelModules(in.Modules),
		toModelEntries(in.Entries),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, toReleaseResponse(out))
}


func (h ReleaseHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.Svc.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
