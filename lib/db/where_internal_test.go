package db

import (
	"regexp"
	"strconv"
	"testing"
)

var testPlaceholderPattern = regexp.MustCompile(`\$(\d+)`)

// assertPlaceholdersConsistent verifies the contract between a generated SQL
// statement and its parameter slice: every parameter is referenced by at
// least one placeholder, and no placeholder references a missing parameter.
// A violation surfaces in Postgres as 42P18 ("could not determine data type
// of parameter $n") or a bind-count mismatch.
func assertPlaceholdersConsistent(t *testing.T, sql string, params []any) {
	t.Helper()
	referenced := make(map[int]bool)
	for _, m := range testPlaceholderPattern.FindAllStringSubmatch(sql, -1) {
		n, err := strconv.Atoi(m[1])
		if err != nil {
			t.Fatalf("unparsable placeholder %q in SQL: %s", m[0], sql)
		}
		if n < 1 || n > len(params) {
			t.Errorf("placeholder $%d out of range (have %d params) in SQL: %s", n, len(params), sql)
		}
		referenced[n] = true
	}
	for i := 1; i <= len(params); i++ {
		if !referenced[i] {
			t.Errorf("parameter $%d is never referenced (Postgres 42P18) in SQL: %s", i, sql)
		}
	}
}

func Test_where_expression_placeholder_renumbering(t *testing.T) {

	manyValues := func(n int) []any {
		vs := make([]any, n)
		for i := range vs {
			vs[i] = "v" + strconv.Itoa(i)
		}
		return vs
	}

	tests := []struct {
		name        string
		where       whereExpectingLogicalOperator
		expectSQL   string
		expectCount int
	}{
		{
			// Search binds $1, then a 2-param OR expression is embedded.
			name: "expression_after_one_param",
			where: Where().Noop().
				And().Search("niralee").
				And().Expression(
				Where().
					Key("grants.allow_private_investments").Equals().Value(true).
					Or().
					Key("requested_products.private_investments").Equals().Value(true),
			).
				And().Key("management.managing_entity").Equals().Value("some-entity"),
			expectSQL: `1=1 AND "text_search" @@ to_tsquery('english', $1) AND ` +
				`("object"->'grants'->'allow_private_investments'=$2 OR "object"->'requested_products'->'private_investments'=$3)` +
				` AND "object"->'management'->'managing_entity'=$4`,
			expectCount: 4,
		},
		{
			// An expression embedded as the first statement, then a second
			// expression that itself contains a nested expression (cursor
			// pagination shape: created_at < a OR (created_at = a AND id < b)).
			name: "nested_expression_after_expression",
			where: Where().Expression(
				Where().
					Key("email.to").Equals().Value("e1").
					Or().
					Key("push.to").Equals().Value("e1"),
			).
				And().Expression(
				Where().
					Key("created_at").LessThan().Value("at").
					Or().
					Expression(
						Where().
							Key("created_at").Equals().Value("at").
							And().
							Key("message_id").LessThan().Value("mid"),
					),
			),
			expectSQL: `("object"->'email'->'to'=$1 OR "object"->'push'->'to'=$2) AND ` +
				`("object"->'created_at'<$3 OR ("object"->'created_at'=$4 AND "object"->'message_id'<$5))`,
			expectCount: 5,
		},
		{
			// IN-list before an expression holding two IN-lists over the
			// same values (find-offers shape).
			name: "expression_with_in_lists_after_one_param",
			where: Where().Noop().
				And().Key("user_id").In().Values("u1").
				And().Expression(
				Where().
					Key("user_id").In().Values("a", "b").
					Or().
					Key("ad_user_id").In().Values("a", "b"),
			),
			expectSQL: `1=1 AND "object"->'user_id' IN ($1) AND ` +
				`("object"->'user_id' IN ($2,$3) OR "object"->'ad_user_id' IN ($4,$5))`,
			expectCount: 5,
		},
		{
			// Ten params on each side: renumbering $1 must not also rewrite
			// the "$1" prefix of "$10".
			name: "expression_with_double_digit_placeholders",
			where: Where().Noop().
				And().Key("a").In().Values(manyValues(10)...).
				And().Expression(
				Where().
					Key("b").In().Values(manyValues(10)...),
			),
			expectSQL: `1=1 AND "object"->'a' IN ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) AND ` +
				`("object"->'b' IN ($11,$12,$13,$14,$15,$16,$17,$18,$19,$20))`,
			expectCount: 20,
		},
		{
			// Two consecutive 2-param expressions from a zero-param base:
			// already worked before the fix and must keep working.
			name: "consecutive_expressions_from_zero_params",
			where: Where().Noop().
				And().Expression(
				Where().
					Key("grants.allow_investments").Equals().Value(true).
					Or().
					Key("requested_products.notes_investments").Equals().Value(true),
			).
				And().Expression(
				Where().
					Key("grants.allow_private_investments").Equals().Value(true).
					Or().
					Key("requested_products.private_investments").Equals().Value(true),
			),
			expectSQL: `1=1 AND ` +
				`("object"->'grants'->'allow_investments'=$1 OR "object"->'requested_products'->'notes_investments'=$2) AND ` +
				`("object"->'grants'->'allow_private_investments'=$3 OR "object"->'requested_products'->'private_investments'=$4)`,
			expectCount: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, params, err := tt.where.statement()
			if err != nil {
				t.Fatalf("statement() failed: %v", err)
			}
			if len(params) != tt.expectCount {
				t.Errorf("expected %d params, got %d", tt.expectCount, len(params))
			}
			if sql != tt.expectSQL {
				t.Errorf("unexpected SQL\n want: %s\n  got: %s", tt.expectSQL, sql)
			}
			assertPlaceholdersConsistent(t, sql, params)
		})
	}
}
