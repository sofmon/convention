package db

import (
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func Where() whereExpectingFirstStatement {
	return &where{}
}

type whereExpectingFirstStatement interface {
	Noop() whereExpectingLogicalOperator
	Key(key string) whereExpectingOperators
	Search(text string) whereExpectingLogicalOperator
	CreatedBetween(a, b time.Time) whereExpectingLogicalOperator
	CreatedBy(user string) whereExpectingLogicalOperator
	UpdatedBetween(a, b time.Time) whereExpectingLogicalOperator
	UpdatedBy(user string) whereExpectingLogicalOperator
	Expression(where whereExpectingLogicalOperator) whereExpectingLogicalOperator
}

type whereClosed interface {
	statement() (string, []any, error)
}

type whereOrdered interface {
	statement() (string, []any, error)

	LimitPerShard(limit int) whereLimited
}

type whereLimited interface {
	statement() (string, []any, error)

	Offset(offset int) whereClosed
}

type whereReady interface {
	statement() (string, []any, error)
}

type whereExpectingOperators interface {
	Equals() whereExpectingValue
	NotEquals() whereExpectingValue
	GreaterThan() whereExpectingValue
	GreaterThanOrEquals() whereExpectingValue
	LessThan() whereExpectingValue
	LessThanOrEquals() whereExpectingValue
	In() whereExpectingValues
	NotIn() whereExpectingValues
	Like() whereExpectingValue
}

type whereExpectingLogicalOperator interface {
	Or() whereExpectingFirstStatement
	And() whereExpectingFirstStatement
	OrderByAsc(key string) whereOrdered
	OrderByDesc(key string) whereOrdered

	statement() (string, []any, error)
}

type whereExpectingValue interface {
	Value(value any) whereExpectingLogicalOperator
}

type whereExpectingValues interface {
	Values(values ...any) whereExpectingLogicalOperator
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

func (w *where) Noop() whereExpectingLogicalOperator {
	_, w.err = w.query.WriteString("1=1")
	return w
}

func (w *where) Expression(where whereExpectingLogicalOperator) whereExpectingLogicalOperator {
	if w.err != nil {
		return w
	}
	_, w.err = w.query.WriteRune('(')
	if w.err != nil {
		return w
	}
	query, params, err := where.statement()
	if err != nil {
		w.err = err
		return w
	}

	for i, param := range params {
		query = strings.ReplaceAll(query, "$"+strconv.Itoa(i+1), "$"+strconv.Itoa(len(w.params)+1))
		w.params = append(w.params, param)
	}

	_, w.err = w.query.WriteString(query)
	if w.err != nil {
		return w
	}

	_, w.err = w.query.WriteRune(')')
	return w
}

func keyToJsonColumn(key string) string {
	split := strings.Split(key, ".")
	if len(split) == 0 {
		return ""
	}
	if len(split) == 1 {
		return `"object"->'` + split[0] + `'`
	}
	var sb strings.Builder
	sb.WriteString(`"object"`)
	for _, s := range split {
		sb.WriteString(`->'` + s + `'`)
	}
	return sb.String()
}

func (w *where) Key(key string) whereExpectingOperators {
	if w.err != nil {
		return w
	}
	if key == "" {
		w.err = errors.New("key cannot be empty")
	}
	_, w.err = w.query.WriteString(keyToJsonColumn(key))
	return w
}

func (w *where) Equals() whereExpectingValue {
	if w.err != nil {
		return w
	}
	_, w.err = w.query.WriteRune('=')
	return w
}

func (w *where) NotEquals() whereExpectingValue {
	if w.err != nil {
		return w
	}
	_, w.err = w.query.WriteString(`!=`)
	return w
}

func (w *where) GreaterThan() whereExpectingValue {
	if w.err != nil {
		return w
	}
	_, w.err = w.query.WriteRune('>')
	return w
}

func (w *where) GreaterThanOrEquals() whereExpectingValue {
	if w.err != nil {
		return w
	}
	_, w.err = w.query.WriteString(`>=`)
	return w
}

func (w *where) LessThan() whereExpectingValue {
	if w.err != nil {
		return w
	}
	_, w.err = w.query.WriteRune('<')
	return w
}

func (w *where) LessThanOrEquals() whereExpectingValue {
	if w.err != nil {
		return w
	}
	_, w.err = w.query.WriteString(`<=`)
	return w
}

func (w *where) Like() whereExpectingValue {
	_, w.err = w.query.WriteString(` LIKE `)
	return w
}

func (w *where) In() whereExpectingValues {
	_, w.err = w.query.WriteString(` IN `)
	return w
}

func (w *where) NotIn() whereExpectingValues {
	_, w.err = w.query.WriteString(` NOT IN `)
	return w
}

func (w *where) Value(value any) whereExpectingLogicalOperator {
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

func (w *where) Values(values ...any) whereExpectingLogicalOperator {
	if w.err != nil {
		return w
	}
	_, w.err = w.query.WriteRune('(')
	if w.err != nil {
		return w
	}
	for i, value := range values {
		if i > 0 {
			_, w.err = w.query.WriteString(`,`)
			if w.err != nil {
				return w
			}
		}
		_, w.err = w.query.WriteString(`$` + strconv.Itoa(len(w.params)+1))
		if w.err != nil {
			return w
		}
		var jsonValue []byte
		jsonValue, w.err = json.Marshal(value)
		if w.err != nil {
			return w
		}
		w.params = append(w.params, string(jsonValue))
	}
	_, w.err = w.query.WriteRune(')')
	return w
}

func (w *where) Or() whereExpectingFirstStatement {
	_, w.err = w.query.WriteString(` OR `)
	return w
}

func (w *where) And() whereExpectingFirstStatement {
	_, w.err = w.query.WriteString(` AND `)
	return w
}

func (w *where) Search(text string) whereExpectingLogicalOperator {
	_, w.err = w.query.WriteString(`"text_search" @@ to_tsquery('english', $` + strconv.Itoa(len(w.params)+1) + `)`)
	w.params = append(w.params, toTSQuery(text))
	return w
}

func (w *where) CreatedBetween(a, b time.Time) whereExpectingLogicalOperator {
	_, w.err = w.query.WriteString(`"created_at" BETWEEN $` + strconv.Itoa(len(w.params)+1) + ` AND $` + strconv.Itoa(len(w.params)+2))
	w.params = append(w.params, a, b)
	return w
}

func (w *where) CreatedBy(user string) whereExpectingLogicalOperator {
	_, w.err = w.query.WriteString(`"created_by" = $` + strconv.Itoa(len(w.params)+1))
	w.params = append(w.params, user)
	return w
}

func (w *where) UpdatedBetween(a, b time.Time) whereExpectingLogicalOperator {
	_, w.err = w.query.WriteString(`"updated_at" BETWEEN $` + strconv.Itoa(len(w.params)+1) + ` AND $` + strconv.Itoa(len(w.params)+2))
	w.params = append(w.params, a, b)
	return w
}

func (w *where) UpdatedBy(user string) whereExpectingLogicalOperator {
	_, w.err = w.query.WriteString(`"updated_by" = $` + strconv.Itoa(len(w.params)+1))
	w.params = append(w.params, user)
	return w
}

func (w *where) OrderByAsc(key string) whereOrdered {
	if w.err != nil {
		return w
	}
	_, w.err = w.query.WriteString(` ORDER BY ` + keyToJsonColumn(key) + ` ASC`)
	return w
}

func (w *where) OrderByDesc(key string) whereOrdered {
	if w.err != nil {
		return w
	}
	_, w.err = w.query.WriteString(` ORDER BY ` + keyToJsonColumn(key) + ` DESC`)
	return w
}

func (w *where) LimitPerShard(limit int) whereLimited {
	if w.err != nil {
		return w
	}
	_, w.err = w.query.WriteString(` LIMIT ` + strconv.Itoa(limit))
	return w
}

func (w *where) Offset(offset int) whereClosed {
	if w.err != nil {
		return w
	}
	_, w.err = w.query.WriteString(` OFFSET ` + strconv.Itoa(offset))

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
