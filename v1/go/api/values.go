package api

type value struct {
	Name  string
	Value string
}

type values []value

func (v values) GetByKey(key string) string {
	for _, p := range v {
		if p.Name == key {
			return p.Value
		}
	}
	return ""
}

func (v values) GetByIndex(i int) string {
	if i < len(v) {
		return v[i].Value
	}
	return ""
}

func (v *values) Add(key, val string) {
	*v = append(*v, value{Name: key, Value: val})
}

func (v *values) Del(key string) {
	for i, p := range *v {
		if p.Name == key {
			*v = append((*v)[:i], (*v)[i+1:]...)
			return
		}
	}
}

func (v values) Has(key string) bool {
	for _, p := range v {
		if p.Name == key {
			return true
		}
	}
	return false
}
