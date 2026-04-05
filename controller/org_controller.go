package controller

import (
	"net/http"

	"kpi-backend/dao"

	"github.com/gin-gonic/gin"
)

type OrgController struct{}

func (OrgController) Tree(c *gin.Context) {
	pelouroDao := dao.PelouroDao{}
	direcaoDao := dao.DirecaoDao{}
	deptDao := dao.DepartamentoDao{}

	pelouros, err := pelouroDao.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	type DeptNode struct {
		ID            uint        `json:"id"`
		Name          string      `json:"name"`
		ResponsibleID *uint       `json:"responsible_id,omitempty"`
		Responsible   interface{} `json:"responsible,omitempty"`
		UsersCount    int         `json:"users_count"`
	}

	type DirecaoNode struct {
		ID            uint        `json:"id"`
		Name          string      `json:"name"`
		ResponsibleID *uint       `json:"responsible_id,omitempty"`
		Responsible   interface{} `json:"responsible,omitempty"`
		Departamentos []DeptNode  `json:"departamentos"`
	}

	type PelouroNode struct {
		ID            uint          `json:"id"`
		Name          string        `json:"name"`
		ResponsibleID *uint         `json:"responsible_id,omitempty"`
		Responsible   interface{}   `json:"responsible,omitempty"`
		Direcoes      []DirecaoNode `json:"direcoes"`
	}

	var tree []PelouroNode

	for _, p := range pelouros {
		pNode := PelouroNode{
			ID:            p.ID,
			Name:          p.Name,
			ResponsibleID: p.ResponsibleID,
			Responsible:   p.Responsible,
		}

		direcoes, _ := direcaoDao.ListByPelouro(p.ID)
		for _, d := range direcoes {
			dNode := DirecaoNode{
				ID:            d.ID,
				Name:          d.Name,
				ResponsibleID: d.ResponsibleID,
				Responsible:   d.Responsible,
			}

			depts, _ := deptDao.ListByDirecao(d.ID)
			for _, dept := range depts {
				users, _ := deptDao.GetUsers(dept.ID)
				dNode.Departamentos = append(dNode.Departamentos, DeptNode{
					ID:            dept.ID,
					Name:          dept.Name,
					ResponsibleID: dept.ResponsibleID,
					Responsible:   dept.Responsible,
					UsersCount:    len(users),
				})
			}

			pNode.Direcoes = append(pNode.Direcoes, dNode)
		}

		tree = append(tree, pNode)
	}

	c.JSON(http.StatusOK, gin.H{"pelouros": tree})
}
