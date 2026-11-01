package pipeline

// Record 是管道中的最小数据单元：字段名到字符串值的映射。
type Record map[string]string

// Schema 描述一批记录的字段顺序与数值属性。
type Schema struct {
	Fields  []string
	Numeric map[string]bool
}

// NewSchema 创建 schema。
func NewSchema(fields ...string) *Schema {
	return &Schema{Fields: fields, Numeric: map[string]bool{}}
}

// WithNumeric 标记数值字段。
func (s *Schema) WithNumeric(fs ...string) *Schema {
	for _, f := range fs {
		s.Numeric[f] = true
	}
	return s
}

// LineSchema 用于纯文本流。
func LineSchema() *Schema {
	return NewSchema("line")
}
