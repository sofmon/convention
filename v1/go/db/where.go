package db

import (
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"
)

func Where() *where {
	return &where{}
}

type where struct {
	query  strings.Builder
	params []any
	err    error
}

func (w *where) statement() (string, []any, error) {
	if w == nil {
		return "", nil, errors.New("where statement is nil")
	}
	return w.query.String(), w.params, w.err
}

func (w *where) Begin() *where {
	if w.err != nil {
		return w
	}
	_, w.err = w.query.WriteRune('(')
	return w
}

func (w *where) End() *where {
	if w.err != nil {
		return w
	}
	_, w.err = w.query.WriteRune(')')
	return w
}

func (w *where) Key(key string) *where {
	if w.err != nil {
		return w
	}
	split := strings.Split(key, ".")
	switch len(split) {
	case 0:
		return w
	case 1:
		_, w.err = w.query.WriteString(`"object"->'` + split[0] + `'`)
	default:
		_, w.err = w.query.WriteString(`"object"`)
		for _, s := range split {
			_, w.err = w.query.WriteString(`->'` + s + `'`)

		}
	}
	return w
}

func (w *where) Equals() *where {
	if w.err != nil {
		return w
	}
	_, w.err = w.query.WriteString(`=`)
	return w
}

func (w *where) NotEquals() *where {
	if w.err != nil {
		return w
	}
	_, w.err = w.query.WriteString(`!=`)
	return w
}

func (w *where) GreaterThan() *where {
	if w.err != nil {
		return w
	}
	_, w.err = w.query.WriteString(`>`)
	return w
}

func (w *where) GreaterThanOrEquals() *where {
	if w.err != nil {
		return w
	}
	_, w.err = w.query.WriteString(`>=`)
	return w
}

func (w *where) LessThan() *where {
	if w.err != nil {
		return w
	}
	_, w.err = w.query.WriteString(`<`)
	return w
}

func (w *where) LessThanOrEquals() *where {
	if w.err != nil {
		return w
	}
	_, w.err = w.query.WriteString(`<=`)
	return w
}

func (w *where) Value(value any) *where {
	if w.err != nil {
		return w
	}
	_, w.err = w.query.WriteString(`$` + strconv.Itoa(len(w.params)+1))
	if w.err != nil {
		return w
	}
	var jsonValue []byte
	jsonValue, w.err = json.Marshal(value)
	w.params = append(w.params, string(jsonValue))
	return w
}

func (w *where) Or() *where {
	_, w.err = w.query.WriteString(` OR `)
	return w
}

func (w *where) And() *where {
	_, w.err = w.query.WriteString(` AND `)
	return w
}

func (w *where) Search(text string) *where {
	_, w.err = w.query.WriteString(`"text_search" @@ to_tsquery('english', $` + strconv.Itoa(len(w.params)+1) + `)`)
	w.params = append(w.params, toTSQuery(text))
	return w
}

func (w *where) Limit(limit int) *where {
	_, w.err = w.query.WriteString(` LIMIT ` + strconv.Itoa(limit))
	return w
}

func toTSQuery(input string) string {

	// Step 1: Remove non-alphanumeric characters (except spaces)
	re := regexp.MustCompile(`[^\w\s]`)
	cleaned := re.ReplaceAllString(input, "")

	// Step 2: Replace multiple spaces with a single space
	cleaned = regexp.MustCompile(`\s+`).ReplaceAllString(cleaned, " ")

	// Step 3: Trim leading and trailing spaces (if any)
	cleaned = strings.TrimSpace(cleaned)

	// Step 4: Replace spaces with the '&' operator
	formatted := strings.ReplaceAll(cleaned, " ", " & ")

	return formatted
}
