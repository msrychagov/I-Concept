package usecase

import (
	"encoding/json"
	"vucdiagnostic/internal/domain/model"
)

func (a *Service) publicItems(s model.Session) []map[string]any {
	out := []map[string]any{}
	for _, i := range a.itemsFor(s) {
		m := map[string]any{"id": i.ID, "module": i.Module, "text": i.Text, "min": i.Min, "max": i.Max, "type": i.Type}
		if i.Hint != "" {
			m["hint"] = i.Hint
		}
		if i.Topic != "" {
			m["topic"] = i.Topic
		}
		if i.TimeWindow != "" {
			m["timeWindow"] = i.TimeWindow
		}
		if i.ResponseOnly {
			m["responseOnly"] = true
		}
		if i.Section != "" {
			m["module"] = i.Section
		}
		if i.Module == "B" {
			m["indicator"] = i.Indicator
		}
		if len(i.Options) > 0 {
			opts := []map[string]string{}
			for _, o := range i.Options {
				opts = append(opts, map[string]string{"id": o.ID, "text": o.Text})
			}
			m["options"] = opts
		}
		out = append(out, m)
	}
	return out
}
func (a *Service) groupReport(all []model.SessionRecord, group string) (any, error) {
	type g struct {
		Group, Wave, Profile, ProfileID string
		People                          map[string]model.Session
	}
	gs := map[string]*g{}
	for _, r := range all {
		s := r.Session
		if s.Kind != "student" || (group != "" && s.Group != group) {
			continue
		}
		snapshot, _ := json.Marshal(s.Profile)
		k := s.Group + "/" + s.Wave + "/" + s.Profile.ID + "/" + s.BankVersion + "/" + hash(string(snapshot))
		if gs[k] == nil {
			gs[k] = &g{s.Group, s.Wave, s.Profile.Name, s.Profile.ID, map[string]model.Session{}}
		}
		identity := s.StudentID
		if identity == "" {
			identity = "legacy:" + s.Participant
		}
		old, ok := gs[k].People[identity]
		if !ok || old.Created < s.Created {
			gs[k].People[identity] = s
		}
	}
	out := []map[string]any{}
	for _, x := range gs {
		rows := []map[string]any{}
		for _, ind := range a.bank.Indicators {
			if ind.Descriptive {
				continue
			}
			enabled := false
			for _, session := range x.People {
				enabled = a.indicatorEnabled(session.Profile, ind.ID)
				break
			}
			if !enabled {
				continue
			}
			sum := 0.0
			n := 0
			var target *float64
			for _, s := range x.People {
				target = s.Profile.Values[ind.ID]
				ids := ind.Items
				for _, effective := range a.engineFor(s).bank.Indicators {
					if effective.ID == ind.ID {
						ids = effective.Items
						break
					}
				}
				if v := a.mean(s, ids); v != nil {
					sum += *v
					n++
				}
			}
			var actual *float64
			if n > 0 {
				v := sum / float64(n)
				actual = &v
			}
			rows = append(rows, map[string]any{"id": ind.ID, "name": ind.Name, "block": ind.Block, "target": target, "actual": actual, "n": n})
		}
		out = append(out, map[string]any{"group": x.Group, "wave": x.Wave, "profile": x.Profile, "n": len(x.People), "rows": rows})
	}
	return out, nil
}
func filterRecords(a []model.SessionRecord, f func(model.Session) bool) []model.SessionRecord {
	o := []model.SessionRecord{}
	for _, r := range a {
		if f(r.Session) {
			o = append(o, r)
		}
	}
	return o
}
