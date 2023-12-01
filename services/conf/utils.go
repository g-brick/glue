package conf

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/fatih/structs"
)

func DealConfigData(data map[string]interface{}, c interface{}) error {
	s := structs.New(c)
	finial := make(map[string]interface{})

	if err := DealFields(s.Fields(), data, finial); err != nil {
		return err
	}

	content, err := json.Marshal(finial)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(content, c); err != nil {
		return err
	}

	return nil
}


func DealFields(fields []*structs.Field, data map[string]interface{}, finial map[string]interface{}) error {
	for _, fd := range fields {
		if !fd.IsExported() {
			continue
		}

		if fd.Kind() != reflect.Ptr && fd.Kind() != reflect.Struct {
			tag := fd.Tag("json")
			if tag != "" {
				value, ok := data[tag]
				if !ok {
					continue
				}

				v := value.(string)
				v = strings.ReplaceAll(v, "\\n", "")

				switch fd.Kind() {
				case reflect.String:
					finial[tag] = v
				default:
					var tmp interface{}
					if err := json.Unmarshal([]byte(v), &tmp); err != nil {
						return err
					}
					finial[tag] = tmp
				}
			}
			continue
		}

		if len(fd.Fields()) > 0 {
			tmpFinial := make(map[string]interface{})
			if fd.Name() != "Config" {
				finial[fd.Name()] = tmpFinial
			} else {
				tmpFinial = finial
			}

			if err := DealFields(fd.Fields(), data, tmpFinial); err != nil {
				return err
			}
		}
	}
	return nil
}

