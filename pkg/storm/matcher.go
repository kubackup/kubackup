package storm

import (
	"github.com/asdine/storm/v3/q"
	"reflect"
	"strings"
)

type like struct {
	val string
}

func (l *like) MatchField(v interface{}) (bool, error) {
	refV := reflect.ValueOf(v)
	if refV.Kind() == reflect.String {
		vs := v.(string)
		if strings.Contains(vs, l.val) {
			return true, nil
		}
	}
	return false, nil
}

func Like(fieldName string, val string) q.Matcher {
	return q.NewFieldMatcher(fieldName, &like{val: val})
}
