package formdata

import (
	"strconv"
	"time"
)

// alternate names
// formhandler
// webform
// htmlform
// reformer
// reform
// formed
// formdata
// formdatahandler
// dform
// stdform
// genericform
// simpleform

// include http.Handler helpers

// the overwhelming majority of forms are not nested and do not have arrays

type Form struct {
	Fields []*Field
}

func (f *Form) New() *FormData {
	return f.Load(map[string]any{})
}

func (f *Form) Load(params map[string]any) *FormData {
	fd := &FormData{
		Form:        f,
		FieldValues: make(map[string]*FieldData),
	}

	for _, field := range f.Fields {
		fd.FieldValues[field.Name] = &FieldData{
			Value: params[field.Name],
		}
	}

	return fd
}

func (f *Form) Parse(params map[string]any) *FormData {
	fd := &FormData{
		Form:        f,
		FieldValues: make(map[string]*FieldData),
	}

	for _, field := range f.Fields {
		fieldData := &FieldData{}
		if submittedValue, ok := params[field.Name].(string); ok {
			fieldData.SubmittedValue = submittedValue
			switch field.Type {
			case "text", "longtext", "password":
				fieldData.Value = submittedValue
			case "duration":
				value, err := time.ParseDuration(submittedValue)
				if err != nil {
					fieldData.Error = err.Error()
				} else {
					fieldData.Value = value
				}
			case "number":
				value, err := strconv.ParseFloat(submittedValue, 64)
				if err != nil {
					fieldData.Error = err.Error()
				} else {
					fieldData.Value = value
				}
			default:
				panic("unknown field type")
			}
		}
		fd.FieldValues[field.Name] = fieldData
	}

	return fd

}

type Field struct {
	// Form      *Form
	Label    string
	Name     string
	Type     string
	Required bool
}

type FormData struct {
	Form *Form

	FieldValues map[string]*FieldData
	Errors      []string
}

type FieldData struct {
	// FormData *FormData
	// Field    *Field

	SubmittedValue string
	Value          any
	Error          string
}
