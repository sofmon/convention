package api

type Values map[string]string

func (v Values) Get(key string) string {
	return v[key]
}

func (v Values) Set(key, value string) {
	v[key] = value
}

func (v Values) Del(key string) {
	delete(v, key)
}

func (v Values) Has(key string) bool {
	_, ok := v[key]
	return ok
}
