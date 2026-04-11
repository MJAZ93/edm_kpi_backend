package dao

import (
	"kpi-backend/model"

	"gorm.io/gorm"
)

// UserScope holds the resolved org entity IDs visible to a user.
type UserScope struct {
	IsGlobal        bool // ADMIN/CA — no filtering
	Role            string
	UserID          uint
	PelouroIDs      []uint
	DirecaoIDs      []uint
	DepartamentoIDs []uint
	// Regional director fields — set when DIRECAO user has no direcão but is
	// responsible for a Região. RegiaoID != 0 indicates regional-director mode.
	RegiaoID uint
	ASCIDs   []uint
}

// ResolveScope determines which org entities a user can see based on role.
//
// Visibility model (branch visibility):
//   - ADMIN / CA: everything
//   - PELOURO: their pelouro + all child direcoes + all child departamentos
//   - DIRECAO: their direcao + all child departamentos (+ parent pelouro for context)
//   - DEPARTAMENTO: their departamento(s) (+ parent direcao/pelouro for context)
func ResolveScope(userID uint, role string) UserScope {
	scope := UserScope{UserID: userID, Role: role}

	if role == "ADMIN" || role == "CA" {
		scope.IsGlobal = true
		return scope
	}

	switch role {
	case "PELOURO":
		scope.resolvePelouro(userID)
	case "DIRECAO":
		scope.resolveDirecao(userID)
	case "DEPARTAMENTO":
		scope.resolveDepartamento(userID)
	}

	return scope
}

func (s *UserScope) resolvePelouro(userID uint) {
	// Find pelouro where user is responsible
	var pelouros []model.Pelouro
	Database.Where("responsible_id = ?", userID).Find(&pelouros)
	for _, p := range pelouros {
		s.PelouroIDs = append(s.PelouroIDs, p.ID)
	}

	// All direcoes under those pelouros
	if len(s.PelouroIDs) > 0 {
		var direcoes []model.Direcao
		Database.Where("pelouro_id IN ?", s.PelouroIDs).Find(&direcoes)
		for _, d := range direcoes {
			s.DirecaoIDs = append(s.DirecaoIDs, d.ID)
		}
	}

	// All departamentos under those direcoes
	if len(s.DirecaoIDs) > 0 {
		var depts []model.Departamento
		Database.Where("direcao_id IN ?", s.DirecaoIDs).Find(&depts)
		for _, d := range depts {
			s.DepartamentoIDs = append(s.DepartamentoIDs, d.ID)
		}
	}
}

func (s *UserScope) resolveDirecao(userID uint) {
	// Find direcao where user is responsible
	var direcoes []model.Direcao
	Database.Where("responsible_id = ?", userID).Find(&direcoes)
	for _, d := range direcoes {
		s.DirecaoIDs = append(s.DirecaoIDs, d.ID)
		// Include parent pelouro for context
		s.PelouroIDs = appendUnique(s.PelouroIDs, d.PelouroID)
	}

	// All departamentos under those direcoes
	if len(s.DirecaoIDs) > 0 {
		var depts []model.Departamento
		Database.Where("direcao_id IN ?", s.DirecaoIDs).Find(&depts)
		for _, d := range depts {
			s.DepartamentoIDs = append(s.DepartamentoIDs, d.ID)
		}
	}

	// If no direcões found, check if user is responsible for a Região
	// (regional director: read-only access scoped via milestone geography)
	if len(s.DirecaoIDs) == 0 {
		s.resolveRegiao(userID)
	}
}

// resolveRegiao populates RegiaoID and ASCIDs for a regional director.
func (s *UserScope) resolveRegiao(userID uint) {
	var r model.Regiao
	if err := Database.Where("responsible_id = ?", userID).First(&r).Error; err != nil {
		return
	}
	s.RegiaoID = r.ID
	var ascs []model.ASC
	Database.Where("regiao_id = ?", r.ID).Find(&ascs)
	for _, a := range ascs {
		s.ASCIDs = append(s.ASCIDs, a.ID)
	}
}

func (s *UserScope) resolveDepartamento(userID uint) {
	// Find departamentos via departamento_users join table
	var duList []model.DepartamentoUser
	Database.Where("user_id = ?", userID).Find(&duList)
	for _, du := range duList {
		s.DepartamentoIDs = appendUnique(s.DepartamentoIDs, du.DepartamentoID)
	}

	// Also check if user is responsible for a departamento
	var depts []model.Departamento
	Database.Where("responsible_id = ?", userID).Find(&depts)
	for _, d := range depts {
		s.DepartamentoIDs = appendUnique(s.DepartamentoIDs, d.ID)
	}

	// Resolve parent direcoes and pelouros for context
	if len(s.DepartamentoIDs) > 0 {
		var allDepts []model.Departamento
		Database.Where("id IN ?", s.DepartamentoIDs).Find(&allDepts)
		for _, d := range allDepts {
			s.DirecaoIDs = appendUnique(s.DirecaoIDs, d.DirecaoID)
		}
	}
	if len(s.DirecaoIDs) > 0 {
		var dirs []model.Direcao
		Database.Where("id IN ?", s.DirecaoIDs).Find(&dirs)
		for _, d := range dirs {
			s.PelouroIDs = appendUnique(s.PelouroIDs, d.PelouroID)
		}
	}
}

// ApplyToProjects filters a project query by scope.
// Projects use creator_type (CA/PELOURO/DIRECAO/DEPARTAMENTO) + creator_org_id.
func (s *UserScope) ApplyToProjects(q *gorm.DB) *gorm.DB {
	if s.IsGlobal {
		return q
	}

	// Regional director: no org hierarchy — find projects via milestone geography
	if s.RegiaoID != 0 {
		// Always include CA/ADMIN-level projects (globally visible)
		cond := Database.Where("creator_type IN ('CA','ADMIN')")

		// Projects where any milestone is scoped to this Região
		regiaoSubq := Database.Model(&model.Project{}).
			Select("DISTINCT projects.id").
			Joins("JOIN tasks ON tasks.project_id = projects.id AND tasks.deleted_at IS NULL").
			Joins("JOIN milestones ON milestones.task_id = tasks.id AND milestones.deleted_at IS NULL").
			Where("milestones.scope_type = 'REGIAO' AND milestones.scope_id = ?", s.RegiaoID)
		cond = cond.Or("id IN (?)", regiaoSubq)

		// Projects where any milestone is scoped to an ASC inside this Região
		if len(s.ASCIDs) > 0 {
			ascSubq := Database.Model(&model.Project{}).
				Select("DISTINCT projects.id").
				Joins("JOIN tasks ON tasks.project_id = projects.id AND tasks.deleted_at IS NULL").
				Joins("JOIN milestones ON milestones.task_id = tasks.id AND milestones.deleted_at IS NULL").
				Where("milestones.scope_type = 'ASC' AND milestones.scope_id IN ?", s.ASCIDs)
			cond = cond.Or("id IN (?)", ascSubq)
		}

		return q.Where(cond)
	}

	// Department users should also see any project that has tasks owned by
	// their department(s), even when the project itself was created higher up.
	if s.Role == "DEPARTAMENTO" {
		taskOwnedProjectSubq := Database.Model(&model.Task{}).
			Select("DISTINCT project_id").
			Where("owner_type = 'DEPARTAMENTO' AND owner_id IN ? AND deleted_at IS NULL", safeIDs(s.DepartamentoIDs))

		return q.Where(
			Database.Where("creator_type IN ('CA','ADMIN')").
				Or("creator_type = 'PELOURO' AND creator_org_id IN ?", safeIDs(s.PelouroIDs)).
				Or("creator_type = 'DIRECAO' AND creator_org_id IN ?", safeIDs(s.DirecaoIDs)).
				Or("creator_type = 'DEPARTAMENTO' AND creator_org_id IN ?", safeIDs(s.DepartamentoIDs)).
				Or("id IN (?)", taskOwnedProjectSubq),
		)
	}

	// For DIRECAO/PELOURO users, also check project_direcoes join table and task ownership
	taskOwnedProjectSubq := Database.Model(&model.Task{}).
		Select("DISTINCT project_id").
		Where("deleted_at IS NULL AND ((owner_type = 'DIRECAO' AND owner_id IN ?) OR (owner_type = 'DEPARTAMENTO' AND owner_id IN ?))",
			safeIDs(s.DirecaoIDs), safeIDs(s.DepartamentoIDs))

	direcaoLinkedSubq := Database.Table("project_direcoes").
		Select("project_id").
		Where("direcao_id IN ?", safeIDs(s.DirecaoIDs))

	return q.Where(
		Database.Where("creator_type IN ('CA','ADMIN')").
			Or("creator_type = 'PELOURO' AND creator_org_id IN ?", safeIDs(s.PelouroIDs)).
			Or("creator_type = 'DIRECAO' AND creator_org_id IN ?", safeIDs(s.DirecaoIDs)).
			Or("creator_type = 'DEPARTAMENTO' AND creator_org_id IN ?", safeIDs(s.DepartamentoIDs)).
			Or("id IN (?)", taskOwnedProjectSubq).
			Or("id IN (?)", direcaoLinkedSubq),
	)
}

// ApplyToTasks filters a task query by scope.
// Tasks use owner_type (DIRECAO/DEPARTAMENTO) + owner_id.
func (s *UserScope) ApplyToTasks(q *gorm.DB) *gorm.DB {
	if s.IsGlobal {
		return q
	}
	// Regional directors are read-only viewers; they can see all tasks within any
	// project they have access to (project-level scope already applied upstream).
	if s.RegiaoID != 0 {
		return q
	}
	return q.Where(
		Database.Where("owner_type = 'DIRECAO' AND owner_id IN ?", safeIDs(s.DirecaoIDs)).
			Or("owner_type = 'DEPARTAMENTO' AND owner_id IN ?", safeIDs(s.DepartamentoIDs)),
	)
}

// ApplyToMilestones filters milestones by joining through the task's owner scope.
func (s *UserScope) ApplyToMilestones(q *gorm.DB) *gorm.DB {
	if s.IsGlobal {
		return q
	}
	// Regional directors see all milestones within accessible tasks (read-only).
	if s.RegiaoID != 0 {
		return q
	}
	return q.Where("task_id IN (?)",
		Database.Model(&model.Task{}).Select("id").Where(
			Database.Where("owner_type = 'DIRECAO' AND owner_id IN ?", safeIDs(s.DirecaoIDs)).
				Or("owner_type = 'DEPARTAMENTO' AND owner_id IN ?", safeIDs(s.DepartamentoIDs)),
		),
	)
}

// TaskIDsSubquery returns a subquery of task IDs visible to this scope.
func (s *UserScope) TaskIDsSubquery() *gorm.DB {
	q := Database.Model(&model.Task{}).Select("id")
	if s.IsGlobal {
		return q
	}
	return q.Where(
		Database.Where("owner_type = 'DIRECAO' AND owner_id IN ?", safeIDs(s.DirecaoIDs)).
			Or("owner_type = 'DEPARTAMENTO' AND owner_id IN ?", safeIDs(s.DepartamentoIDs)),
	)
}

// CanSeeTask checks if a task falls within this user's scope.
func (s *UserScope) CanSeeTask(task model.Task) bool {
	if s.IsGlobal {
		return true
	}
	// Regional directors have read-only access to all tasks in visible projects.
	if s.RegiaoID != 0 {
		return true
	}
	switch task.OwnerType {
	case "DIRECAO":
		return containsID(s.DirecaoIDs, task.OwnerID)
	case "DEPARTAMENTO":
		return containsID(s.DepartamentoIDs, task.OwnerID)
	}
	return false
}

// CanSeeProject checks if a project falls within this user's scope.
func (s *UserScope) CanSeeProject(p model.Project) bool {
	if s.IsGlobal {
		return true
	}
	if p.CreatorType == "CA" || p.CreatorType == "ADMIN" {
		return true
	}
	// Regional directors: access determined by milestone geography, not org hierarchy.
	// Treat as visible (caller should rely on ApplyToProjects for list queries).
	if s.RegiaoID != 0 {
		return true
	}
	if p.CreatorOrgID == nil {
		return true
	}
	switch p.CreatorType {
	case "PELOURO":
		return containsID(s.PelouroIDs, *p.CreatorOrgID)
	case "DIRECAO":
		return containsID(s.DirecaoIDs, *p.CreatorOrgID)
	case "DEPARTAMENTO":
		return containsID(s.DepartamentoIDs, *p.CreatorOrgID)
	}
	return false
}

func containsID(ids []uint, id uint) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

func appendUnique(slice []uint, val uint) []uint {
	for _, v := range slice {
		if v == val {
			return slice
		}
	}
	return append(slice, val)
}

// safeIDs ensures we never pass an empty slice to IN clause (would match nothing).
func safeIDs(ids []uint) []uint {
	if len(ids) == 0 {
		return []uint{0} // impossible ID, ensures no match
	}
	return ids
}
