package core

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// Validator collects field errors from a set of rules applied to a
// map[string]string of input values (typically built from Context.Input)
// or directly from struct fields via ValidateStruct.
type Validator struct {
	data   map[string]string
	errors map[string][]string
}

func NewValidator(data map[string]string) *Validator {
	return &Validator{data: data, errors: map[string][]string{}}
}

func (v *Validator) addError(field, msg string) {
	v.errors[field] = append(v.errors[field], msg)
}

func (v *Validator) Required(field string) *Validator {
	if strings.TrimSpace(v.data[field]) == "" {
		v.addError(field, fmt.Sprintf("%s is required", field))
	}
	return v
}

func (v *Validator) Email(field string) *Validator {
	val := v.data[field]
	if val != "" && !emailRegex.MatchString(val) {
		v.addError(field, fmt.Sprintf("%s must be a valid email address", field))
	}
	return v
}

func (v *Validator) MinLen(field string, min int) *Validator {
	if val := v.data[field]; val != "" && len(val) < min {
		v.addError(field, fmt.Sprintf("%s must be at least %d characters", field, min))
	}
	return v
}

func (v *Validator) MaxLen(field string, max int) *Validator {
	if val := v.data[field]; val != "" && len(val) > max {
		v.addError(field, fmt.Sprintf("%s must be at most %d characters", field, max))
	}
	return v
}

func (v *Validator) Numeric(field string) *Validator {
	val := v.data[field]
	if val != "" {
		if _, err := strconv.ParseFloat(val, 64); err != nil {
			v.addError(field, fmt.Sprintf("%s must be numeric", field))
		}
	}
	return v
}

// ValidateStruct inspects struct fields using reflection and validates
// them according to the `validate` struct tag (e.g. validate:"required,email,min=6").
func (v *Validator) ValidateStruct(s interface{}) *Validator {
	val := reflect.ValueOf(s)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return v
	}

	t := val.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("validate")
		if tag == "" || tag == "-" {
			continue
		}

		fieldVal := val.Field(i)
		fieldName := field.Tag.Get("json")
		if fieldName == "" {
			fieldName = field.Tag.Get("db")
		}
		if fieldName == "" || fieldName == "-" {
			fieldName = strings.ToLower(field.Name)
		}

		var strVal string
		isZero := fieldVal.IsZero()

		switch fieldVal.Kind() {
		case reflect.String:
			strVal = fieldVal.String()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			strVal = strconv.FormatInt(fieldVal.Int(), 10)
		case reflect.Float32, reflect.Float64:
			strVal = strconv.FormatFloat(fieldVal.Float(), 'f', -1, 64)
		case reflect.Bool:
			strVal = strconv.FormatBool(fieldVal.Bool())
		default:
			strVal = fmt.Sprintf("%v", fieldVal.Interface())
		}

		if v.data == nil {
			v.data = make(map[string]string)
		}
		v.data[fieldName] = strVal

		rules := strings.Split(tag, ",")
		for _, rule := range rules {
			rule = strings.TrimSpace(rule)
			if rule == "" {
				continue
			}

			parts := strings.SplitN(rule, "=", 2)
			ruleName := strings.ToLower(parts[0])
			var ruleVal string
			if len(parts) > 1 {
				ruleVal = parts[1]
			}

			switch ruleName {
			case "required":
				if isZero || strings.TrimSpace(strVal) == "" {
					v.addError(fieldName, fmt.Sprintf("%s is required", fieldName))
				}
			case "email":
				if strVal != "" && !emailRegex.MatchString(strVal) {
					v.addError(fieldName, fmt.Sprintf("%s must be a valid email address", fieldName))
				}
			case "numeric":
				if strVal != "" {
					if _, err := strconv.ParseFloat(strVal, 64); err != nil {
						v.addError(fieldName, fmt.Sprintf("%s must be numeric", fieldName))
					}
				}
			case "min":
				minVal, _ := strconv.Atoi(ruleVal)
				if fieldVal.Kind() == reflect.String {
					if len(strVal) < minVal {
						v.addError(fieldName, fmt.Sprintf("%s must be at least %d characters", fieldName, minVal))
					}
				} else if fieldVal.Kind() >= reflect.Int && fieldVal.Kind() <= reflect.Int64 {
					if fieldVal.Int() < int64(minVal) {
						v.addError(fieldName, fmt.Sprintf("%s must be at least %d", fieldName, minVal))
					}
				} else if fieldVal.Kind() == reflect.Float32 || fieldVal.Kind() == reflect.Float64 {
					if fieldVal.Float() < float64(minVal) {
						v.addError(fieldName, fmt.Sprintf("%s must be at least %d", fieldName, minVal))
					}
				}
			case "max":
				maxVal, _ := strconv.Atoi(ruleVal)
				if fieldVal.Kind() == reflect.String {
					if len(strVal) > maxVal {
						v.addError(fieldName, fmt.Sprintf("%s must be at most %d characters", fieldName, maxVal))
					}
				} else if fieldVal.Kind() >= reflect.Int && fieldVal.Kind() <= reflect.Int64 {
					if fieldVal.Int() > int64(maxVal) {
						v.addError(fieldName, fmt.Sprintf("%s must be at most %d", fieldName, maxVal))
					}
				} else if fieldVal.Kind() == reflect.Float32 || fieldVal.Kind() == reflect.Float64 {
					if fieldVal.Float() > float64(maxVal) {
						v.addError(fieldName, fmt.Sprintf("%s must be at most %d", fieldName, maxVal))
					}
				}
			}
		}
	}
	return v
}

func (v *Validator) Passes() bool {
	return len(v.errors) == 0
}

func (v *Validator) Errors() map[string][]string {
	return v.errors
}

func (v *Validator) FirstError() string {
	for _, msgs := range v.errors {
		if len(msgs) > 0 {
			return msgs[0]
		}
	}
	return ""
}
