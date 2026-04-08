package controller

import (
	"encoding/json"
	"net/http"
	"strconv"

	"kpi-backend/dao"
	"kpi-backend/model"
	"kpi-backend/util"

	"github.com/gin-gonic/gin"
)

type GeoController struct{}

// --- Regiões ---

func (GeoController) ListRegioes(c *gin.Context) {
	geoDao := dao.GeoDao{}
	list, err := geoDao.GetAllRegioes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list, "total": len(list)})
}

type RegiaoInput struct {
	Name          string          `json:"name" binding:"required"`
	Code          string          `json:"code"`
	Polygon       json.RawMessage `json:"polygon"`
	ResponsibleID *uint           `json:"responsible_id"`
}

func (GeoController) CreateRegiao(c *gin.Context) {
	var input RegiaoInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	polygonStr := ""
	if len(input.Polygon) > 0 {
		polygonStr = string(input.Polygon)
	}

	regiao := model.Regiao{
		Name:          input.Name,
		Code:          input.Code,
		Polygon:       polygonStr,
		ResponsibleID: input.ResponsibleID,
	}

	geoDao := dao.GeoDao{}
	if err := geoDao.CreateRegiao(&regiao); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	auditDao := dao.AuditDao{}
	auditDao.Write("regiao", regiao.ID, util.ExtractUserID(c), "CREATE", nil, regiao, c.ClientIP())

	c.JSON(http.StatusCreated, regiao)
}

func (GeoController) SingleRegiao(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	geoDao := dao.GeoDao{}
	regiao, err := geoDao.GetRegiaoByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}
	c.JSON(http.StatusOK, regiao)
}

func (GeoController) UpdateRegiao(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	geoDao := dao.GeoDao{}
	regiao, err := geoDao.GetRegiaoByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	oldData := regiao

	var input RegiaoInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	polygonStr := ""
	if len(input.Polygon) > 0 {
		polygonStr = string(input.Polygon)
	}

	regiao.Name = input.Name
	regiao.Code = input.Code
	regiao.Polygon = polygonStr
	regiao.ResponsibleID = input.ResponsibleID

	if err := geoDao.UpdateRegiao(&regiao); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	auditDao := dao.AuditDao{}
	auditDao.Write("regiao", regiao.ID, util.ExtractUserID(c), "UPDATE", oldData, regiao, c.ClientIP())

	c.JSON(http.StatusOK, regiao)
}

func (GeoController) DeleteRegiao(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	geoDao := dao.GeoDao{}

	regiao, err := geoDao.GetRegiaoByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	if err := geoDao.DeleteRegiao(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	auditDao := dao.AuditDao{}
	auditDao.Write("regiao", uint(id), util.ExtractUserID(c), "DELETE", regiao, nil, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// --- ASCs ---

func (GeoController) ListASCs(c *gin.Context) {
	geoDao := dao.GeoDao{}
	list, err := geoDao.GetAllASCs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list, "total": len(list)})
}

type ASCInput struct {
	Name          string          `json:"name" binding:"required"`
	Code          string          `json:"code"`
	RegiaoID      *uint           `json:"regiao_id"`
	Polygon       json.RawMessage `json:"polygon"`
	ResponsibleID *uint           `json:"responsible_id"`
	DirectorID    *uint           `json:"director_id"`
}

func (GeoController) CreateASC(c *gin.Context) {
	var input ASCInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	ascPolygonStr := ""
	if len(input.Polygon) > 0 {
		ascPolygonStr = string(input.Polygon)
	}

	asc := model.ASC{
		Name:          input.Name,
		Code:          input.Code,
		RegiaoID:      input.RegiaoID,
		Polygon:       ascPolygonStr,
		ResponsibleID: input.ResponsibleID,
		DirectorID:    input.DirectorID,
	}

	geoDao := dao.GeoDao{}
	if err := geoDao.CreateASC(&asc); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	auditDao := dao.AuditDao{}
	auditDao.Write("asc", asc.ID, util.ExtractUserID(c), "CREATE", nil, asc, c.ClientIP())

	c.JSON(http.StatusCreated, asc)
}

func (GeoController) SingleASC(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	geoDao := dao.GeoDao{}
	asc, err := geoDao.GetASCByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}
	c.JSON(http.StatusOK, asc)
}

func (GeoController) UpdateASC(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	geoDao := dao.GeoDao{}
	asc, err := geoDao.GetASCByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	oldData := asc

	var input ASCInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	updatePolygonStr := ""
	if len(input.Polygon) > 0 {
		updatePolygonStr = string(input.Polygon)
	}

	asc.Name = input.Name
	asc.Code = input.Code
	asc.RegiaoID = input.RegiaoID
	asc.Polygon = updatePolygonStr
	asc.ResponsibleID = input.ResponsibleID
	asc.DirectorID = input.DirectorID

	if err := geoDao.UpdateASC(&asc); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	auditDao := dao.AuditDao{}
	auditDao.Write("asc", asc.ID, util.ExtractUserID(c), "UPDATE", oldData, asc, c.ClientIP())

	c.JSON(http.StatusOK, asc)
}

func (GeoController) DeleteASC(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	geoDao := dao.GeoDao{}

	asc, err := geoDao.GetASCByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	if err := geoDao.DeleteASC(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	auditDao := dao.AuditDao{}
	auditDao.Write("asc", uint(id), util.ExtractUserID(c), "DELETE", asc, nil, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
