package gp

type Relationship struct {
	Table string
	// One of "ParentHasMany", "ChildHasMany", "HasOne", "BelongsTo"
	Type                  string
	IsArray               bool
	Alias                 string
	Parent                Table
	ParentName, ChildName string
	OperativeColumn       string
}

func (r Relationship) IsHasMany() bool {
	return r.Type == "ParentHasMany"
}

func (r Relationship) IsChildHasMany() bool {
	return r.Type == "ChildHasMany"
}

func (r Relationship) IsHasOne() bool {
	return r.Type == "HasOne"
}

func (r Relationship) IsBelongsTo() bool {
	return r.Type == "BelongsTo"
}

func (r Relationship) ColumnName() string {
	if r.Alias != "" {
		return r.Alias + "ID"
	}
	switch r.Type {
	case "ParentHasMany":
		return r.Parent.name + "ID"
	case "ChildHasMany":
		return r.Table + "ID"
	case "HasOne":
		return r.Parent.name + "ID"
	case "BelongsTo":
		return r.Table + "ID"
	}
	return ""
}

func (r Relationship) Name() string {
	if r.Alias != "" {
		return r.Alias
	}
	return r.Table
}
