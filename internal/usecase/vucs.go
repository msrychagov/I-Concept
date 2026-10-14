package usecase

import (
	"strings"
	"vucdiagnostic/internal/domain/model"
)

func vucIn(db model.Database, name string) (model.VUC, bool) {
	name = strings.Join(strings.Fields(name), " ")
	for _, vuc := range db.VUCs {
		if sameVUC(vuc.Name, name) {
			return vuc, true
		}
	}
	return model.VUC{}, false
}

// CreateVUC is exposed only by administrator HTTP and maintenance CLI adapters.
func (a *Service) CreateVUC(name string) (any, error) {
	name = strings.Join(strings.Fields(name), " ")
	if name == "" || len(name) > 200 {
		return nil, problem(Invalid, "Укажите название ВУЦ (до 200 символов).")
	}
	return a.transaction(func(db *model.Database) (any, error) {
		if _, exists := vucIn(*db, name); exists {
			return nil, problem(Conflict, "ВУЦ с таким названием уже существует.")
		}
		vuc := model.VUC{ID: uuid(), Name: name}
		db.VUCs = append(db.VUCs, vuc)
		return vuc, nil
	})
}

func (a *Service) ListVUCs() (any, error) {
	db, err := a.repo.ReadSnapshot()
	if err != nil {
		return nil, err
	}
	list := db.VUCs
	if list == nil {
		list = []model.VUC{}
	}
	return map[string]any{"vucs": list}, nil
}
