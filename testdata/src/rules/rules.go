package rules

// This comment deliberately exceeds the configured line width and must be split across multiple lines. // want "comment exceeds 80 columns"
// See https://example.com/a/path/that/is/too/long/to/wrap/without/changing/the/meaning/of/the/resource/identifier
//go:generate a-command-with-an-intentionally-long-argument-that-must-remain-on-one-line-because-go-generate-reads-directives-line-by-line

func assignmentRules() {
	x := 1

	queryText := `
		SELECT *
		FROM users
	`
	_ = queryText

	var declared int
	_ = declared
	var typed int = 1
	_ = typed

	empty := []int{}
	_ = empty

	y :=
		1 // want "invalid line break after operator :="

	var z int
	z =
		2 // want "invalid line break after operator ="

	a, b := 10, 20
	a, b = b,
		a // want "invalid line break after assignment comma"

	_, _, _, _, _ = x, y, z, a, b
}

func binaryOperatorRules() {
	sum := 1 +
		2 // want "invalid line break after binary operator"

	_ = sum
}

func declarationBlockRules() {
	var singleVar = 1
	const singleConst = 2

	var (
		aVar    = 1
		another = 2
	)

	const (
		aConst = "a"
		bConst = "b"
	)

	var (
		badA, badB = 1,
			2 // want "invalid line break after assignment comma"
	)

	_, _, _, _, _, _ = singleVar, singleConst, aVar, another, aConst, bConst
	_, _ = badA, badB
}

func validFunc(a int, b string) string {
	return b
}

func invalidFunc( // want "function signature must be on a single line"
	a int,
	b string,
) string {
	return b
}

func returnExamples() (int, string) {
	return 1, "ok"

	return 1,
		"error" // want "return statement must be on a single line"
}

func rawStringReturn() string {
	return `
		SELECT *
		FROM users
	`
}

func emptyReturn() {
	return
}

func returnExceptionExample() []int {
	// Valid (Exception): Return itself is single-line, but returns a
	// multi-line literal. The numbers below are padded so the
	// reconstructed single line genuinely exceeds 80 columns - otherwise
	// this literal would have to collapse to one line instead.
	return []int{
		11111111111,
		22222222222,
		33333333333,
		44444444444,
		55555555555,
	}

	return []int{11111111111, 22222222222, 33333333333, 44444444444, 55555555555} // want "literal exceeds 80 columns and must be split across multiple lines"
}

func invalidLiteralClosingLayout() {
	_ = []string{ // want "invalid multi-line literal spacing"
		"111111111111111111111111111111111111111111111111111111111111111111111111111111",
		"222222222222222222222222222222222222222222222222222222222222222222222222222222"}
}

func callRules() {
	add(1, 2)

	// No func/composite literal argument, so a call NEVER breaks lines,
	// regardless of how it's split.
	add(
		1,
		2,
	) // want "function call must be on a single line"

	add(
		1, 2) // want "function call must be on a single line"

	query(`
		SELECT *
		FROM users
		WHERE id = ?
	`)

	query( // want "invalid multi-line arguments spacing"
		`SELECT *
		FROM users`)
}

func conditionalRules() {
	x, y := 10, 20

	if x > 5 {
	}

	if x > 5 &&
		y > 5 { // want "invalid line break after binary operator"
	}

	if x := 10; x > 5 { // want "if statements with initialization are not allowed"
	}
}

func loopRules(items []int) {
	for i := 0; i < len(items); i++ {
	}

	for i := 0; // want "invalid line break in for statement"
	i < len(items); i++ {
	}

	for k, v := range items {
		_, _ = k, v
	}
}

func switchRules(val int) {
	switch val {
	case 1:
	default:
	}

	switch {
	case val > 5 &&
		val < 20: // want "invalid line break after binary operator"
	}

	switch x := 10; x { // want "switch statements with initialization are not allowed"
	case 10:
	}
}

func deferAndGoRules() {
	defer assignmentRules()

	defer func() {
	}()
}

type ValidUser struct {
	Name string
	Age  int
}

type InvalidUser struct{ Name string } // want "type declaration must be multi-line"

type ValidReader interface {
	Read(p []byte) (n int, err error)
}

type InvalidReader interface {
	Read(p []byte) (n int, err error)
}

type EmptyStructValid struct{}

type EmptyStructInvalid struct { // want "empty type declaration must be on a single line"
}

type EmptyInterfaceValid interface{}

type EmptyInterfaceInvalid interface { // want "empty type declaration must be on a single line"
}

func typeAssertionAndSwitchRules(i interface{}) {
	str, ok := i.(string)

	switch v := i.(type) {
	case string:
		_ = v
	}

	_, _ = str, ok
}

func compositeLiteralRules() {
	s1 := []int{1, 2, 3}
	u1 := ValidUser{Name: "Alice", Age: 30}

	s2 := []int{ // want "literal fits within 80 columns and must be on a single line"
		1,
		2,
		3,
	}

	u2 := ValidUser{ // want "literal fits within 80 columns and must be on a single line"
		Name: "Bob",
		Age:  25,
	}

	// Lineguard measures the compact form, ignoring gofmt alignment.
	_ = map[string][]string{
		"short":             []string{"12345678", "12345678", "12345678", "12345678"},
		"a-much-longer-key": []string{"one"},
	}

	longSliceValid := []string{
		"this_is_a_very_long_string_that_helps_us_exceed_the_eighty_characters_limit_1",
		"this_is_a_very_long_string_that_helps_us_exceed_the_eighty_characters_limit_2",
	}

	_ = []string{"long_string_1...", "long_string_2...", "long_string_3...", "long_string_4..."} // want "literal exceeds 80 columns and must be split across multiple lines"

	_ = []string{"long_string_1...", // want "invalid multi-line literal spacing"
		"long_string_2...",
		"long_string_3...",
		"long_string_4..."}

	_ = []string{ // want "invalid multi-line literal spacing"
		"this_is_a_very_long_string_that_helps_us_exceed_the_eighty_characters_limit_1", "this_is_a_very_long_string_that_helps_us_exceed_the_eighty_characters_limit_2",
	}

	_, _, _, _ = s1, u1, s2, u2
	_ = longSliceValid
}

type PeerLayout struct {
	Name string
	Note string
}

func compositeSiblingLayoutRules() {
	siblingSpacing := []struct {
		name, value string
		count       int
	}{
		{ // want "invalid multi-line literal spacing"
			name:  "first",
			value: "first", count: 1,
		},
		{
			name:  "second value with enough text to require multiline layout",
			value: "second value with enough text to require multiline layout",
			count: 10,
		},
	}
	_ = siblingSpacing

	peers := []PeerLayout{
		{
			Name: "short",
		},
		{
			Name: "this_name_is_long_enough_to_make_the_composite_literal_exceed_eighty_columns",
			Note: "this_note_is_also_long_enough_to_make_the_composite_literal_exceed_eighty_columns",
		},
	}
	_ = peers

	otherPeers := []PeerLayout{
		{Name: "short"}, // want "literal must use multiline sibling layout"
		{
			Name: "this_name_is_long_enough_to_make_the_composite_literal_exceed_eighty_columns",
			Note: "this_note_is_also_long_enough_to_make_the_composite_literal_exceed_eighty_columns",
		},
	}
	_ = otherPeers

	fixes := []struct{ Message string }{{
		Message: "this_message_is_long_enough_to_make_the_nested_literal_exceed_eighty_columns",
	}}
	_ = fixes

	invalidFixes := []struct {
		Message string
	}{ // want "single-element array or slice must keep its element on the opening line"
		{
			Message: "this_message_is_long_enough_to_make_the_nested_literal_exceed_eighty_columns",
		},
	}
	_ = invalidFixes

	compactPointer := &[]struct {
		Name  string
		Count int
		Index int
	}{
		{Name: "example", Count: 5, Index: 0},
	}
	_ = compactPointer
}

type Query struct{}

func query(s string) {}
func noArgs()        {}

func (q Query) Where(s any) Query    { return q }
func (q Query) Order(s string) Query { return q }
func (q Query) Limit(i int) Query    { return q }

type ChainRoot struct {
	StructField StructField
}

type LongChainRoot struct {
	A LongChainA
}
type LongChainA struct {
	B LongChainB
}
type LongChainB struct {
	C string
}

type StructField struct {
	Field string
}

func (s StructField) Method() string {
	return s.Field
}

func methodChainRules() {
	var db Query

	db.Where("id = 1").Order("name").Limit(10)

	db.Where([]string{
		"this_is_a_very_long_string_that_helps_us_exceed_the_eighty_characters_limit_1",
		"this_is_a_very_long_string_that_helps_us_exceed_the_eighty_characters_limit_2",
	}).Order("name").Limit(10)

	db.Where(func() {
		x := 1
		_ = x
	}).Order("name").Limit(10)

	db.Where(`
		SELECT *
		FROM users
		WHERE id = ?
	`).Order("name").Limit(10)

	db.Where("id = 1").
		Order("name").
		Limit(10) // want "chained methods must be on a single line"

	db.Where([]string{ // want "single-element array or slice must keep its element on the opening line"
		"this_is_a_very_long_string_that_helps_us_exceed_the_eighty_characters_limit_1",
	}).
		Order("name").Limit(10) // want "chained methods must be on a single line"

	db.Where("very_long_conditional_expression_that_forces_this_line_to_exceed_eighty_characters = 1").Order("name").Limit(5)

	db.
		Where("id = 1").Order("name").Limit(10) // want "chained methods must be on a single line"
}

func selectorChainRules() {
	var root ChainRoot
	var longRoot LongChainRoot

	_ = root.StructField.Field

	_ = root. // want "chained selections must be on a single line"
			StructField.Field

	_ = root. // want "chained selections must be on a single line"
			StructField.
			Field

	_ = root.StructField.Method()

	_ = root. // want "chained selections must be on a single line"
			StructField.Method()

	_ = longRoot. // want "chained selections must be on a single line" "chained selections must be on a single line"
			A.
			B.
			C
}

type CallbackRunner struct{}

func (cr CallbackRunner) Run(cb func(), data []int) {}

func nestedCallArgumentRules() {
	var runner CallbackRunner

	runner.Run(func() {
		x := 1
		_ = x
	}, []int{1, 2})

	runner.Run(func() {
	},
		[]int{1, 2}) // want "invalid multi-line arguments spacing"

	runner.Run(func() {}, []int{
		11111111,
		22222222,
		33333333,
		44444444,
		55555555,
	})

	runner.Run(func() {}, []int{11111111, // want "invalid multi-line literal spacing"
		22222222,
		33333333,
		44444444,
		55555555})

	runner.Run(func() {}, []int{ // want "literal fits within 80 columns and must be on a single line"
		1,
		2,
	})
}

func nestedReturnRules() (func(), []int) {
	return func() {
		x := 1
		_ = x
	}, []int{1, 2}

	return func() {
		},
		[]int{1, 2} // want "invalid multi-line return spacing"

	return func() {}, []int{ // want "literal fits within 80 columns and must be on a single line"
		1,
		2,
	}
}

func add(a, b int) int { return a + b }
