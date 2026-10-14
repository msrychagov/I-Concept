package model

type Profile struct {
	Questions   []Item              `json:"questions,omitempty"`
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Group       string              `json:"group"`
	Wave        string              `json:"wave"`
	Author      string              `json:"author"`
	Basis       string              `json:"basis"`
	Approved    bool                `json:"approved"`
	PartialGap  *float64            `json:"partialGap"`
	BandBasis   string              `json:"bandBasis"`
	Values      map[string]*float64 `json:"values"`
	Created     string              `json:"created,omitempty"`
	BankVersion string              `json:"bankVersion,omitempty"`
	OwnerID     string              `json:"ownerId,omitempty"`
}
